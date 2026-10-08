// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/forwarding"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
)

func (a *application) forwardingCommand(root *cobra.Command) {
	var dry bool
	var connect time.Duration
	cmd := &cobra.Command{Use: "forward HOST [NAME]", Short: "Run configured local, remote or dynamic SSH port forwards", Args: cobra.RangeArgs(1, 2)}
	cmd.Flags().BoolVar(&dry, "dry-run", false, "show configured tunnels without connecting or listening")
	cmd.Flags().DurationVar(&connect, "connect-timeout", 0, "connection and tunnel setup timeout (default: inventory)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if a.json && !dry {
			return errors.New("--json is available for forward only with --dry-run")
		}
		inv, err := a.load()
		if err != nil {
			return err
		}
		h, ok := inv.Hosts[args[0]]
		if !ok || h.Connection == "console-only" {
			return errors.New("host is unknown or console-only")
		}
		specs := append([]inventory.Forward(nil), h.Forwards...)
		if len(args) == 2 {
			specs = nil
			for _, f := range h.Forwards {
				if f.Name == args[1] {
					specs = append(specs, f)
				}
			}
		}
		if len(specs) == 0 {
			return errors.New("no matching port forwards; configure hosts.HOST.forwards in the inventory")
		}
		timeout, _ := time.ParseDuration(inv.Defaults.ConnectTimeout)
		if cmd.Flags().Changed("connect-timeout") {
			if connect <= 0 {
				return errors.New("connect-timeout must be positive")
			}
			timeout = connect
			inv.Defaults.ConnectTimeout = connect.String()
		}
		if dry {
			if a.json {
				return a.writeJSON(struct {
					Host     string              `json:"host"`
					DryRun   bool                `json:"dry_run"`
					Forwards []inventory.Forward `json:"forwards"`
				}{args[0], true, specs})
			}
			for _, f := range specs {
				fmt.Fprintf(a.out, "%s/%s %s %s -> %s\n", args[0], f.Name, f.Type, f.Listen, f.Destination)
			}
			return nil
		}
		m, err := a.manager(inv)
		if err != nil {
			return err
		}
		defer m.Close()
		if err := m.Prepare(a.ctx, []string{args[0]}, false); err != nil {
			return err
		}
		client, err := m.Dial(a.ctx, args[0])
		if err != nil {
			fmt.Fprintln(a.errOut, "SSH connection failed:", credentials.Redact(err.Error(), m.Secrets))
			return exitError{3}
		}
		defer client.Close()
		var diagnostics sync.Mutex
		session, err := forwarding.Start(a.ctx, client.Client, specs, timeout, func(err error) {
			diagnostics.Lock()
			defer diagnostics.Unlock()
			fmt.Fprintln(a.errOut, credentials.Redact(err.Error(), m.Secrets))
		})
		if err != nil {
			return fmt.Errorf("port forward setup failed: %s", credentials.Redact(err.Error(), m.Secrets))
		}
		defer session.Close()
		for _, binding := range session.Bindings {
			if _, err := fmt.Fprintf(a.out, "%s/%s %s %s -> %s\n", args[0], binding.Name, binding.Type, binding.Listen, binding.Destination); err != nil {
				return err
			}
		}
		fmt.Fprintln(a.out, "Forwarding active; press Ctrl+C to stop.")
		err = session.Wait()
		if a.ctx.Err() != nil {
			return nil
		}
		if err != nil {
			fmt.Fprintln(a.errOut, credentials.Redact(err.Error(), m.Secrets))
			return exitError{3}
		}
		return nil
	}
	root.AddCommand(cmd)
}
