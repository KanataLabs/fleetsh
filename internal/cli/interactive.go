// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func (a *application) canEditInteractively() bool {
	return !a.json && (a.isTerminal() || a.readLine != nil)
}
func changedHostOptions(cmd *cobra.Command) bool {
	changed := false
	cmd.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name != "help" {
			changed = changed || flag.Changed
		}
	})
	return changed
}
func (a *application) line(label string) (string, error) {
	if err := a.ctx.Err(); err != nil {
		return "", err
	}
	if a.readLine != nil {
		return a.readLine(label)
	}
	fmt.Fprint(a.errOut, label+": ")
	type answer struct {
		text string
		err  error
	}
	done := make(chan answer, 1)
	// Read one byte at a time to avoid buffering a later hidden password prompt.
	go func() {
		var result strings.Builder
		var b [1]byte
		for {
			n, err := a.in.Read(b[:])
			if n > 0 {
				if b[0] == '\n' {
					done <- answer{strings.TrimSpace(result.String()), nil}
					return
				}
				if b[0] != '\r' {
					result.WriteByte(b[0])
				}
			}
			if err != nil {
				done <- answer{"", err}
				return
			}
		}
	}()
	select {
	case <-a.ctx.Done():
		return "", a.ctx.Err()
	case value := <-done:
		if value.err != nil {
			return "", errors.New("Input ended before the change was completed. No host settings were saved.")
		}
		return value.text, nil
	}
}
func (a *application) askFlag(cmd *cobra.Command, name, label, current string) error {
	if current != "" {
		label += " [" + current + "]"
	}
	value, err := a.line(label)
	if err != nil {
		return err
	}
	if value == "" {
		value = current
	}
	return cmd.Flags().Set(name, value)
}
func (a *application) setupHost(cmd *cobra.Command, alias, action string, h *inventory.Host) error {
	if action == "add" {
		if h.Host == "" {
			if err := a.askFlag(cmd, "host", "Hostname or IP address", ""); err != nil {
				return err
			}
		}
		if h.Connection == "ssh" && h.User == "" {
			if err := a.askFlag(cmd, "user", "SSH username", "root"); err != nil {
				return err
			}
		}
		if h.Auth == "key" && h.Key == "" {
			if err := a.askFlag(cmd, "key", "Private key file", ""); err != nil {
				return err
			}
		}
		return requiredHostOptions(*h)
	}
	inv, err := a.load()
	if err != nil {
		return err
	}
	old, exists := inv.Hosts[alias]
	if !exists {
		return unknownHostError(alias)
	}
	fmt.Fprintf(a.errOut, "Edit %s (%s, user %s, authentication %s)\n", alias, old.Address(), old.User, old.Auth)
	fmt.Fprintln(a.errOut, "1  Address\n2  SSH username\n3  Saved SSH password\n4  Groups\n5  Tags\n6  Description\n7  Authentication\nq  Cancel")
	choice, err := a.line("Choose a field")
	if err != nil {
		return err
	}
	switch strings.ToLower(choice) {
	case "1", "address":
		return a.askFlag(cmd, "host", "Hostname or IP address", old.Host)
	case "2", "user":
		return a.askFlag(cmd, "user", "SSH username", old.User)
	case "3", "password":
		fmt.Fprintln(a.errOut, "Enter the server's current password. This updates the local saved copy only.")
		if err := cmd.Flags().Set("auth", "password"); err != nil {
			return err
		}
		return cmd.Flags().Set("save-password", "true")
	case "4", "groups":
		value, err := a.line("Groups, separated by commas (empty clears all)")
		if err != nil {
			return err
		}
		return cmd.Flags().Set("groups", value)
	case "5", "tags":
		value, err := a.line("Tags, separated by commas (empty clears all)")
		if err != nil {
			return err
		}
		return cmd.Flags().Set("tags", value)
	case "6", "description":
		value, err := a.line("Description (empty clears it)")
		if err != nil {
			return err
		}
		return cmd.Flags().Set("description", value)
	case "7", "auth":
		if err := a.askFlag(cmd, "auth", "Authentication: password, key or agent", old.Auth); err != nil {
			return err
		}
		if h.Auth == "key" && old.Key == "" {
			return a.askFlag(cmd, "key", "Private key file", "")
		}
		return nil
	case "q", "cancel", "":
		return fmt.Errorf("Editing %s canceled. No host settings were saved.", alias)
	default:
		return errors.New("Unknown menu choice. Run edit HOST again and choose a number from 1 to 7, or q to cancel.")
	}
}
