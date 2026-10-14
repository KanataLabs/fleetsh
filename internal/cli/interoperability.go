// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/transport"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

func (a *application) interoperabilityCommands(root *cobra.Command) {
	var dry bool
	var user string
	importCmd := &cobra.Command{Use: "import FILE", Short: "Import supported OpenSSH host settings without overwriting existing hosts", Args: cobra.ExactArgs(1)}
	importCmd.Flags().BoolVar(&dry, "dry-run", false, "preview hosts and warnings without changing the inventory")
	importCmd.Flags().StringVar(&user, "user", "", "fallback SSH username for entries without User")
	importCmd.RunE = func(cmd *cobra.Command, args []string) error {
		filePath, err := transport.ExpandPath(args[0])
		if err != nil {
			return err
		}
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()
		imported, err := inventory.ParseSSHConfig(f, user)
		if err != nil {
			return err
		}
		names := make([]string, 0, len(imported.Hosts))
		for name := range imported.Hosts {
			names = append(names, name)
		}
		slices.Sort(names)
		if !a.json {
			for _, warning := range imported.Warnings {
				fmt.Fprintln(a.errOut, "Import note:", warning)
			}
		}
		if len(names) == 0 {
			return fmt.Errorf("No usable hosts were found. Check the import notes and provide literal Host entries with User settings.")
		}
		inv, err := a.load()
		if errors.Is(err, os.ErrNotExist) {
			inv = &inventory.Inventory{}
		} else if err != nil {
			return err
		}
		if inv.Hosts == nil {
			inv.Hosts = map[string]inventory.Host{}
		}
		for _, name := range names {
			if _, exists := inv.Hosts[name]; exists {
				return fmt.Errorf("Host %s already exists. Import will not overwrite it; rename the SSH alias or edit the existing host.", name)
			}
			inv.Hosts[name] = imported.Hosts[name]
		}
		if err := inv.Validate(); err != nil {
			return fmt.Errorf("Cannot import these hosts: %w", err)
		}
		if dry {
			if a.json {
				return a.writeJSON(struct {
					DryRun bool `json:"dry_run"`
					inventory.SSHImport
				}{true, imported})
			}
			for _, name := range names {
				h := inv.Hosts[name]
				fmt.Fprintf(a.out, "Would import %s: %s as %s (%s)\n", name, h.Address(), h.User, h.Auth)
			}
			return nil
		}
		if err := a.changeOrCreate(func(current *inventory.Inventory) error {
			for _, name := range names {
				if _, exists := current.Hosts[name]; exists {
					return fmt.Errorf("Host %s was added by another process. Import stopped without overwriting it.", name)
				}
			}
			for _, name := range names {
				current.Hosts[name] = imported.Hosts[name]
			}
			return nil
		}); err != nil {
			return err
		}
		if a.json {
			return a.writeJSON(imported)
		}
		fmt.Fprintf(a.out, "Imported %d hosts. Run fleetsh ls to review them; use fleetsh doctor HOST to check a login.\n", len(names))
		return nil
	}
	root.AddCommand(importCmd)
	var format, output string
	var overwrite bool
	exportCmd := &cobra.Command{Use: "export", Short: "Export host settings as TOML, JSON or OpenSSH config; saved secrets are excluded", Args: cobra.NoArgs}
	exportCmd.Flags().StringVar(&format, "format", "toml", "toml, json or ssh-config")
	exportCmd.Flags().StringVar(&output, "output", "", "write to this file instead of stdout")
	exportCmd.Flags().BoolVar(&overwrite, "overwrite", false, "explicitly replace an existing output file")
	exportCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if format != "toml" && format != "json" && format != "ssh-config" {
			return usageError{cmd, fmt.Errorf("Unknown export format %q. Choose toml, json or ssh-config.", format)}
		}
		inv, err := a.load()
		if err != nil {
			return err
		}
		clean := inventory.Inventory{Defaults: inv.Defaults, Hosts: map[string]inventory.Host{}}
		clean.Defaults.ProxyCredential = ""
		names := make([]string, 0, len(inv.Hosts))
		for id, h := range inv.Hosts {
			h.Credential = ""
			h.ProxyCredential = ""
			h.SudoCredential = ""
			clean.Hosts[id] = h
			names = append(names, id)
		}
		slices.Sort(names)
		var data []byte
		switch format {
		case "json":
			data, err = json.MarshalIndent(clean, "", "  ")
			data = append(data, '\n')
		case "toml":
			data, err = toml.Marshal(clean)
		case "ssh-config":
			var builder strings.Builder
			builder.WriteString("# Exported by fleetsh. Saved passwords and credential references are excluded.\n")
			for _, id := range names {
				h := clean.Hosts[id]
				if h.Connection == "console-only" {
					fmt.Fprintf(a.errOut, "Export note: %s is console-only and was omitted.\n", id)
					continue
				}
				proxy, _ := inv.ProxyFor(h)
				if proxy != "" || len(h.Forwards) > 0 {
					fmt.Fprintf(a.errOut, "Export note: %s proxy URLs and port-forward profiles are not represented in OpenSSH output.\n", id)
				}
				fmt.Fprintf(&builder, "\nHost %s\n    HostName %s\n    User %s\n    Port %d\n", id, sshConfigQuote(h.Host), sshConfigQuote(h.User), h.Port)
				if h.Key != "" && h.Auth == "key" {
					fmt.Fprintf(&builder, "    IdentityFile %s\n", sshConfigQuote(h.Key))
				}
				if h.Auth == "password" {
					builder.WriteString("    PreferredAuthentications password,keyboard-interactive\n")
				}
				if h.ProxyJump != "" {
					fmt.Fprintf(&builder, "    ProxyJump %s\n", sshConfigQuote(h.ProxyJump))
				}
			}
			data = []byte(builder.String())
		}
		if err != nil {
			return err
		}
		if output == "" {
			_, err = a.out.Write(data)
			return err
		}
		outputPath, err := transport.ExpandPath(output)
		if err != nil {
			return err
		}
		flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		if overwrite {
			flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		f, err := os.OpenFile(outputPath, flags, 0600)
		if os.IsExist(err) {
			return fmt.Errorf("Output file %s already exists. Choose another path or use --overwrite to replace it.", outputPath)
		}
		if err != nil {
			return err
		}
		_, writeErr := f.Write(data)
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintf(a.errOut, "Exported host settings to %s. System credential-store values and references were excluded.\n", outputPath)
		return nil
	}
	root.AddCommand(exportCmd)
}
func sshConfigQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return "\"" + value + "\""
}
