// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"context"

	"github.com/KanataLabs/fleetsh/internal/actions"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/spf13/cobra"
)

func (a *application) monitoringCommands(root *cobra.Command) {
	for _, definition := range []struct{ name, short, command string }{
		{"alive", "Check authenticated SSH liveness (default: all hosts)", actions.Alive},
		{"stats", "Print Linux CPU, memory, swap, user and free disk space", actions.Stats},
	} {
		o := remoteOptions{}
		cmd := &cobra.Command{Use: definition.name + " [SELECTOR]", Short: definition.short, Args: cobra.MaximumNArgs(1)}
		if definition.name == "stats" {
			cmd.Aliases = []string{"monitor"}
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"@all"}
			}
			inv, ids, n, err := a.targets(cmd, args, o)
			if err != nil {
				return err
			}
			m, err := a.manager(inv)
			if err != nil {
				return err
			}
			defer m.Close()
			if err := m.Prepare(a.ctx, ids, o.sudo); err != nil {
				return err
			}
			rows := executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
				return executor.Run(ctx, m, id, definition.command, executor.Options{Timeout: o.timeout, Sudo: o.sudo})
			})
			return a.results(rows)
		}
		o.flags(cmd)
		root.AddCommand(cmd)
	}
}
