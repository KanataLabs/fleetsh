// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
)

func (a *application) runCommand(root *cobra.Command) {
	o := remoteOptions{}
	var dry bool
	cmd := &cobra.Command{Use: "run SELECTOR NAME", Short: "Execute a named command from your inventory", Args: cobra.ExactArgs(2)}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		inv, ids, n, err := a.targets(cmd, args, o)
		if err != nil {
			return err
		}
		c, ok := inv.Commands[args[1]]
		if !ok {
			names := make([]string, 0, len(inv.Commands))
			for name := range inv.Commands {
				names = append(names, name)
			}
			slices.Sort(names)
			available := strings.Join(names, ", ")
			if available == "" {
				available = "none configured"
			}
			return fmt.Errorf("Unknown saved command %q. Available commands: %s.\nNext: add a [commands.NAME] section; run fleetsh docs commands for an example.", args[1], available)
		}
		if !cmd.Flags().Changed("sudo") {
			o.sudo = c.Sudo
		}
		if !cmd.Flags().Changed("timeout") && c.Timeout != "" {
			o.timeout, _ = time.ParseDuration(c.Timeout)
		}
		if dry {
			plans := []plan{}
			for _, id := range ids {
				plans = append(plans, plan{Host: id, Command: c.Command, Sudo: o.sudo, Skipped: inv.Hosts[id].Connection == "console-only"})
			}
			return a.showPlan(plans)
		}
		return a.executeCommand(inv, ids, n, c.Command, o)
	}
	o.flags(cmd)
	cmd.Flags().BoolVar(&dry, "dry-run", false, "show selected hosts and saved command without connecting")
	root.AddCommand(cmd)
}
func (a *application) executeCommand(inv *inventory.Inventory, ids []string, n int, command string, o remoteOptions) error {
	m, err := a.manager(inv)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Prepare(a.ctx, ids, o.sudo); err != nil {
		return err
	}
	rows := executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
		return executor.Run(ctx, m, id, command, executor.Options{Timeout: o.timeout, Sudo: o.sudo})
	})
	return a.results(rows)
}
