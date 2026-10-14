// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/transport"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
)

type transferPlan struct {
	Host      string `json:"host"`
	Direction string `json:"direction"`
	Local     string `json:"local"`
	Remote    string `json:"remote"`
	Overwrite bool   `json:"overwrite"`
	Skipped   bool   `json:"skipped"`
}

func downloadDirectory(alias string) string {
	sum := sha256.Sum256([]byte(alias))
	return "host-" + strings.TrimRight(alias, ".") + "-" + fmt.Sprintf("%x", sum[:4])
}
func (a *application) transferCommands(root *cobra.Command) {
	for _, direction := range []string{"upload", "download"} {
		o := remoteOptions{}
		var dry, overwrite bool
		usage := "upload SELECTOR LOCAL_FILE REMOTE_FILE"
		short := "Upload one file through SFTP to selected hosts"
		if direction == "download" {
			usage = "download SELECTOR REMOTE_FILE LOCAL_PATH"
			short = "Download one file through SFTP; groups use separate host directories"
		}
		cmd := &cobra.Command{Use: usage, Short: short, Args: cobra.ExactArgs(3)}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if o.sudo {
				return usageError{cmd, fmt.Errorf("SFTP does not support --sudo. Use a remote path writable by the SSH user.")}
			}
			inv, ids, n, err := a.targets(cmd, args, o)
			if err != nil {
				return err
			}
			local, remote := args[1], args[2]
			if direction == "download" {
				remote, local = args[1], args[2]
			}
			local, err = transport.ExpandPath(local)
			if err != nil {
				return err
			}
			if strings.TrimSpace(local) == "" || strings.TrimSpace(remote) == "" || strings.ContainsAny(remote, "\x00\r\n") || path.Base(remote) == "/" || path.Base(remote) == "." {
				return usageError{cmd, fmt.Errorf("Provide a local path and a remote file path, for example /tmp/report.txt.")}
			}
			if direction == "download" && !filepath.IsLocal(path.Base(remote)) {
				return usageError{cmd, fmt.Errorf("The remote filename cannot be used as a local filename. Choose a portable filename without path separators or reserved names.")}
			}
			if direction == "upload" {
				info, err := os.Stat(local)
				if err != nil {
					return err
				}
				if !info.Mode().IsRegular() {
					return fmt.Errorf("Upload source %s is not a regular file. Choose one file; directories are not supported.", local)
				}
			}
			plans := make([]transferPlan, 0, len(ids))
			for _, id := range ids {
				destination := local
				if direction == "download" {
					if len(ids) > 1 {
						destination = filepath.Join(local, downloadDirectory(id), path.Base(remote))
					} else if info, err := os.Stat(local); err == nil && info.IsDir() {
						destination = filepath.Join(local, path.Base(remote))
					}
				}
				plans = append(plans, transferPlan{id, direction, destination, remote, overwrite, inv.Hosts[id].Connection == "console-only"})
			}
			if dry {
				if a.json {
					return a.writeJSON(struct {
						DryRun bool           `json:"dry_run"`
						Plans  []transferPlan `json:"plans"`
					}{true, plans})
				}
				for _, p := range plans {
					if p.Skipped {
						fmt.Fprintf(a.out, "%s: skipped (console-only).\n", p.Host)
					} else {
						fmt.Fprintf(a.out, "%s: would %s %s <-> %s (overwrite: %t).\n", p.Host, p.Direction, p.Local, p.Remote, p.Overwrite)
					}
				}
				return nil
			}
			m, err := a.manager(inv)
			if err != nil {
				return err
			}
			defer m.Close()
			if err := m.Prepare(a.ctx, ids, false); err != nil {
				return err
			}
			byHost := map[string]transferPlan{}
			for _, p := range plans {
				byHost[p.Host] = p
			}
			rows := executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
				return transferFile(ctx, m, byHost[id], o.timeout)
			})
			return a.results(rows)
		}
		o.flags(cmd)
		cmd.Flags().BoolVar(&dry, "dry-run", false, "show paths and selected hosts without connecting or writing files")
		cmd.Flags().BoolVar(&overwrite, "overwrite", false, "explicitly allow replacing an existing destination file")
		root.AddCommand(cmd)
	}
}
func transferFile(ctx context.Context, m *transport.Manager, p transferPlan, timeout time.Duration) (r executor.Result) {
	started := time.Now()
	r = executor.Result{Host: p.Host, ExitCode: -1}
	defer func() { r.Duration = time.Since(started).Seconds(); r.Error = credentials.Redact(r.Error, m.Secrets) }()
	if timeout == 0 {
		timeout, _ = time.ParseDuration(m.Inventory.Defaults.CommandTimeout)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	c, err := m.Dial(ctx, p.Host)
	if err != nil {
		r.Error = err.Error()
		r.ErrorKind = executor.Classify(err)
		return
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	client, err := sftp.NewClient(c.Client)
	if err != nil {
		r.Error = "The server could not start SFTP. Check that its SFTP subsystem is enabled."
		r.ErrorKind = "transfer"
		return
	}
	defer client.Close()
	var count int64
	if p.Direction == "upload" {
		count, err = uploadFile(client, p.Local, p.Remote, p.Overwrite)
	} else {
		count, err = downloadFile(client, p.Remote, p.Local, p.Overwrite)
	}
	if err != nil {
		r.Error = err.Error()
		r.ErrorKind = "transfer"
		if ctx.Err() != nil {
			r.ErrorKind = "timeout"
			if ctx.Err() == context.Canceled {
				r.ErrorKind = "canceled"
			}
		}
		return
	}
	r.Success = true
	r.ExitCode = 0
	r.Started = true
	if p.Direction == "upload" {
		r.Stdout = fmt.Sprintf("Uploaded %d bytes to %s.\n", count, p.Remote)
	} else {
		r.Stdout = fmt.Sprintf("Downloaded %d bytes to %s.\n", count, p.Local)
	}
	return
}
func uploadFile(client *sftp.Client, local, remote string, overwrite bool) (int64, error) {
	source, err := os.Open(local)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if !overwrite {
		if _, err := client.Lstat(remote); err == nil {
			return 0, fmt.Errorf("Remote file %s already exists. Use another path or --overwrite to replace it.", remote)
		} else if !os.IsNotExist(err) {
			return 0, fmt.Errorf("Cannot check remote destination %s: %w", remote, err)
		}
	}
	if overwrite {
		if info, err := client.Lstat(remote); err == nil && !info.Mode().IsRegular() {
			return 0, fmt.Errorf("Remote destination %s is not a regular file; it will not be replaced.", remote)
		}
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	destination, err := client.OpenFile(remote, flags)
	if os.IsExist(err) {
		return 0, fmt.Errorf("Remote file %s already exists. Use another path or --overwrite to replace it.", remote)
	}
	if err != nil {
		return 0, fmt.Errorf("Cannot open remote file %s for upload. Check its parent directory and SSH-user permissions: %w", remote, err)
	}
	n, err := io.Copy(destination, source)
	closeErr := destination.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return n, fmt.Errorf("Upload to %s did not finish; a partial file may remain. Check it before retrying: %w", remote, err)
	}
	return n, nil
}
func downloadFile(client *sftp.Client, remote, local string, overwrite bool) (int64, error) {
	info, err := client.Lstat(remote)
	if err != nil {
		return 0, fmt.Errorf("Cannot read remote file %s. Check the path and SSH-user permissions: %w", remote, err)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("Remote source %s is not a regular file. Choose one file; directories and symlinks are not supported.", remote)
	}
	source, err := client.Open(remote)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(local), 0700); err != nil {
		return 0, err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		if info, err := os.Lstat(local); err == nil && !info.Mode().IsRegular() {
			return 0, fmt.Errorf("Local destination %s is not a regular file; it will not be replaced.", local)
		}
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	destination, err := os.OpenFile(local, flags, 0600)
	if os.IsExist(err) {
		return 0, fmt.Errorf("Local file %s already exists. Use another path or --overwrite to replace it.", local)
	}
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(destination, source)
	if err == nil {
		err = destination.Sync()
	}
	closeErr := destination.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return n, fmt.Errorf("Download to %s did not finish; a partial file may remain. Check it before retrying: %w", local, err)
	}
	return n, nil
}
