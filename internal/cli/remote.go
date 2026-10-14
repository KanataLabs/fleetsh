// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KanataLabs/fleetsh/internal/actions"
	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type remoteOptions struct {
	tag              string
	parallel         int
	serial, sudo     bool
	timeout, connect time.Duration
}

func (o *remoteOptions) flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.tag, "tag", "", "filter targets by tag")
	cmd.Flags().IntVar(&o.parallel, "parallel", 0, "worker count (default: inventory; reboot: 2)")
	cmd.Flags().BoolVar(&o.serial, "serial", false, "run on one host at a time")
	cmd.Flags().BoolVar(&o.sudo, "sudo", false, "use sudo (credential reference or passwordless sudo)")
	cmd.Flags().DurationVar(&o.timeout, "timeout", 0, "remote command timeout (default: inventory)")
	cmd.Flags().DurationVar(&o.connect, "connect-timeout", 0, "connection timeout (default: inventory)")
}
func (a *application) targets(cmd *cobra.Command, args []string, o remoteOptions) (*inventory.Inventory, []string, int, error) {
	if cmd.Flags().Changed("connect-timeout") && o.connect <= 0 {
		return nil, nil, 0, usageError{cmd, errors.New("--connect-timeout must be positive, for example 30s")}
	}
	if cmd.Flags().Changed("timeout") && o.timeout <= 0 {
		return nil, nil, 0, usageError{cmd, errors.New("--timeout must be positive, for example 60s")}
	}
	if o.parallel < 0 || o.parallel > 256 || (cmd.Flags().Changed("parallel") && o.parallel == 0) {
		return nil, nil, 0, usageError{cmd, errors.New("--parallel must be between 1 and 256")}
	}
	if o.serial && cmd.Flags().Changed("parallel") {
		return nil, nil, 0, usageError{cmd, errors.New("Choose either --serial or --parallel; they cannot be used together.")}
	}
	inv, err := a.load()
	if err != nil {
		return nil, nil, 0, err
	}
	if cmd.Flags().Changed("connect-timeout") {
		inv.Defaults.ConnectTimeout = o.connect.String()
	}
	n := o.parallel
	if n == 0 {
		n = inv.Defaults.Parallel
	}
	if o.serial {
		n = 1
	}
	ids, err := inv.Select(args[0], o.tag)
	return inv, ids, n, err
}
func (a *application) results(rows []executor.Result) error {
	report := executor.Summarize(rows)
	if a.json {
		if err := a.writeJSON(report); err != nil {
			return err
		}
	} else {
		for _, r := range rows {
			if r.Skipped {
				fmt.Fprintf(a.out, "%s SKIPPED\n", r.Host)
				if r.Error != "" {
					fmt.Fprintln(a.out, r.Error)
				}
				continue
			}
			state := "OK"
			if !r.Success {
				state = "FAILED"
			}
			fmt.Fprintf(a.out, "%s %s (%.2fs, exit %d)\n", r.Host, state, r.Duration, r.ExitCode)
			if r.Stdout != "" {
				fmt.Fprint(a.out, r.Stdout)
				if !strings.HasSuffix(r.Stdout, "\n") {
					fmt.Fprintln(a.out)
				}
			}
			if r.Stderr != "" {
				fmt.Fprintf(a.errOut, "[%s stderr]\n%s\n", r.Host, r.Stderr)
			}
			if r.Error != "" {
				fmt.Fprintf(a.errOut, "[%s] %s\n", r.Host, remoteFailure(r.Host, r.ErrorKind, r.Error))
				if r.ErrorKind == "preparation" || r.ErrorKind == "command" {
					fmt.Fprintln(a.errOut, r.Error)
				}
			}
			if r.Truncated {
				fmt.Fprintf(a.errOut, "[%s] output truncated at %d bytes per stream\n", r.Host, executor.OutputLimit)
			}
		}
		fmt.Fprintf(a.out, "Summary: %d succeeded, %d failed, %d skipped.\n", report.Summary.Success, report.Summary.Failed, report.Summary.Skipped)
	}
	if code := executor.ExitCode(rows); code != 0 {
		return exitError{code}
	}
	return nil
}
func (a *application) remoteCommands(root *cobra.Command) {
	var sshTimeout time.Duration
	sshCmd := &cobra.Command{Use: "ssh HOST", Short: "Open an in-process SSH shell; verify unknown fingerprints interactively", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if a.json {
			return errors.New("--json is not available for an interactive SSH shell")
		}
		inv, err := a.load()
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("connect-timeout") {
			if sshTimeout <= 0 {
				return errors.New("connect-timeout must be positive")
			}
			inv.Defaults.ConnectTimeout = sshTimeout.String()
		}
		h, ok := inv.Hosts[args[0]]
		if !ok || h.Connection == "console-only" {
			if !ok {
				return unknownHostError(args[0])
			}
			return fmt.Errorf("Host %s is console-only. Use your provider console, or edit its connection settings to enable SSH.", args[0])
		}
		m, err := a.manager(inv)
		if err != nil {
			return err
		}
		defer m.Close()
		if err = m.Prepare(a.ctx, args, false); err != nil {
			return err
		}
		err = executor.Interactive(a.ctx, m, args[0], a.in, a.out, a.errOut)
		if err != nil {
			fmt.Fprintln(a.errOut, remoteFailure(args[0], executor.Classify(err), credentials.Redact(err.Error(), m.Secrets)))
			var remoteExit *ssh.ExitError
			if errors.As(err, &remoteExit) {
				return exitError{1}
			}
			return exitError{3}
		}
		return nil
	}}
	sshCmd.Flags().DurationVar(&sshTimeout, "connect-timeout", 0, "connection timeout (default: inventory)")
	root.AddCommand(sshCmd)
	execOptions := remoteOptions{}
	execCmd := &cobra.Command{Use: "exec SELECTOR COMMAND", Short: "Execute a quoted shell command on selected hosts", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("exec requires a host/group SELECTOR and exactly one quoted COMMAND")
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		inv, ids, n, err := a.targets(cmd, args, execOptions)
		if err != nil {
			return err
		}
		m, err := a.manager(inv)
		if err != nil {
			return err
		}
		defer m.Close()
		if err = m.Prepare(a.ctx, ids, execOptions.sudo); err != nil {
			return err
		}
		rows := executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
			return executor.Run(ctx, m, id, args[1], executor.Options{Timeout: execOptions.timeout, Sudo: execOptions.sudo})
		})
		return a.results(rows)
	}}
	execOptions.flags(execCmd)
	root.AddCommand(execCmd)
	a.updateCommand(root)
	a.rebootCommand(root)
}

type plan struct {
	Host    string `json:"host"`
	Command string `json:"command,omitempty"`
	Skipped bool   `json:"skipped"`
	Sudo    bool   `json:"sudo"`
}

func (a *application) showPlan(plans []plan) error {
	if a.json {
		return a.writeJSON(struct {
			DryRun bool   `json:"dry_run"`
			Plans  []plan `json:"plans"`
		}{true, plans})
	}
	for _, p := range plans {
		if p.Skipped {
			fmt.Fprintf(a.out, "%s SKIPPED (console-only)\n", p.Host)
		} else {
			prefix := ""
			if p.Sudo {
				prefix = "sudo: "
			}
			fmt.Fprintf(a.out, "%s would execute: %s%s\n", p.Host, prefix, p.Command)
		}
	}
	return nil
}
func (a *application) updateCommand(root *cobra.Command) {
	o := remoteOptions{}
	var dry, yes, dist, checkReboot bool
	cmd := &cobra.Command{Use: "update SELECTOR", Short: "Detect Linux package manager and update after confirmation", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		inv, ids, n, err := a.targets(cmd, args, o)
		if err != nil {
			return err
		}
		m, err := a.manager(inv)
		if err != nil {
			return err
		}
		defer m.Close()
		if err = m.Prepare(a.ctx, ids, o.sudo && !dry); err != nil {
			return err
		}
		commands := make([]string, len(ids))
		indices := make(map[string]int)
		for i, id := range ids {
			indices[id] = i
		}
		plans := []plan{}
		rows := executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
			raw, r := executor.Probe(ctx, m, id, actions.DetectOS, executor.Options{Timeout: 10 * time.Second})
			if r.Success {
				command, e := actions.UpdateCommand(raw, dist)
				if e != nil {
					r.Success = false
					r.Error = e.Error()
					r.ErrorKind = "command"
					r.ExitCode = -1
				} else {
					commands[indices[id]] = command
				}
			}
			return r
		})
		failed := false
		for i, r := range rows {
			if r.Skipped {
				plans = append(plans, plan{Host: r.Host, Skipped: true})
				continue
			}
			if !r.Success {
				failed = true
				continue
			}
			command := commands[i]

			plans = append(plans, plan{Host: r.Host, Command: command, Sudo: o.sudo})
		}
		if failed {
			fmt.Fprintln(a.errOut, "Preflight failed; no updates were dispatched.")
			for i := range rows {
				if rows[i].Success {
					rows[i].Success = false
					rows[i].Skipped = true
				}
			}
			return a.results(rows)
		}
		if dry {
			return a.showPlan(plans)
		}
		if !yes && !a.confirm(fmt.Sprintf("Update selected hosts: %s?", strings.Join(ids, ", "))) {
			return errors.New("update canceled")
		}
		rows = executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
			r := executor.Run(ctx, m, id, commands[indices[id]], executor.Options{Timeout: o.timeout, Sudo: o.sudo})
			if checkReboot && r.Success {
				check := executor.Run(ctx, m, id, actions.RebootRequired, executor.Options{Timeout: 30 * time.Second, Sudo: o.sudo})
				r.Stdout += "\n" + check.Stdout
				if !check.Success {
					r.Success = false
					r.ErrorKind = check.ErrorKind
					r.Error = "The update finished, but the restart check failed: " + check.Error
					r.ExitCode = check.ExitCode
				}
			}
			return r
		})
		return a.results(rows)
	}}
	o.flags(cmd)
	cmd.Flags().BoolVar(&dry, "dry-run", false, "probe OS and show planned commands without updates")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm updating selected hosts")
	cmd.Flags().BoolVar(&dist, "dist", false, "use apt-get dist-upgrade on Debian/Ubuntu")
	cmd.Flags().BoolVar(&checkReboot, "check-reboot", false, "check restart requirements after each successful update")
	root.AddCommand(cmd)
}
func (a *application) rebootCommand(root *cobra.Command) {
	o := remoteOptions{}
	var dry, yes bool
	var wait time.Duration
	var batchSize int
	cmd := &cobra.Command{Use: "reboot SELECTOR", Short: "Reboot Linux hosts and verify a new boot identity plus health", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		inv, ids, n, err := a.targets(cmd, args, o)
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("parallel") && !o.serial {
			n = 2
		}
		if batchSize < 0 || batchSize > 256 || (cmd.Flags().Changed("batch-size") && batchSize == 0) {
			return usageError{cmd, errors.New("--batch-size must be between 1 and 256")}
		}
		if wait <= 0 {
			return errors.New("wait-timeout must be positive")
		}
		if dry {
			plans := []plan{}
			for _, id := range ids {
				plans = append(plans, plan{Host: id, Command: "reboot", Sudo: o.sudo, Skipped: inv.Hosts[id].Connection == "console-only"})
			}
			return a.showPlan(plans)
		}
		if !yes && !a.confirm(fmt.Sprintf("Reboot selected hosts: %s?", strings.Join(ids, ", "))) {
			return errors.New("reboot canceled")
		}
		m, err := a.manager(inv)
		if err != nil {
			return err
		}
		defer m.Close()
		if err = m.Prepare(a.ctx, ids, o.sudo); err != nil {
			return err
		}
		reboot := func(ctx context.Context, id string) executor.Result {
			return actions.Reboot(ctx, m, id, executor.Options{Timeout: o.timeout, Sudo: o.sudo}, wait)
		}
		rows := []executor.Result{}
		if batchSize == 0 {
			rows = executor.Batch(a.ctx, inv, ids, n, reboot)
		} else {
			for start := 0; start < len(ids); start += batchSize {
				end := min(start+batchSize, len(ids))
				batch := executor.Batch(a.ctx, inv, ids[start:end], n, reboot)
				rows = append(rows, batch...)
				if executor.ExitCode(batch) != 0 {
					for _, id := range ids[end:] {
						rows = append(rows, executor.Result{Host: id, Skipped: true, ExitCode: -1, Error: "Not rebooted because the previous batch failed its health check."})
					}
					break
				}
			}
		}
		return a.results(rows)
	}}
	o.flags(cmd)
	cmd.Flags().BoolVar(&dry, "dry-run", false, "show targets without connecting or rebooting")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm rebooting selected hosts")
	cmd.Flags().DurationVar(&wait, "wait-timeout", 5*time.Minute, "maximum time waiting for a new boot and health probe")
	cmd.Flags().IntVar(&batchSize, "batch-size", 0, "reboot in batches; stop if a batch fails its health check")
	root.AddCommand(cmd)
}
