// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KanataLabs/fleetsh/internal/actions"
	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/spf13/cobra"
)

func (a *application) monitoringCommands(root *cobra.Command) {
	for _, definition := range []struct{ name, short, command string }{
		{"alive", "Check authenticated SSH liveness (default: all hosts)", actions.Alive},
		{"reboot-required", "Check whether Linux hosts need a restart (default: all hosts)", actions.RebootRequired},
		{"stats", "Print Linux CPU, memory, swap, user and free disk space", actions.Stats},
	} {
		o := remoteOptions{}
		var table bool
		var watch time.Duration
		cmd := &cobra.Command{Use: definition.name + " [SELECTOR]", Short: definition.short, Args: cobra.MaximumNArgs(1)}
		if definition.name == "stats" {
			cmd.Aliases = []string{"monitor"}
			cmd.Flags().BoolVar(&table, "table", false, "show a compact table (also used by --watch)")
			cmd.Flags().DurationVar(&watch, "watch", 0, "repeat snapshots after this interval, e.g. 5s; Ctrl+C stops")
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("watch") && watch < time.Second {
				return usageError{cmd, fmt.Errorf("--watch must be at least 1s, for example --watch 5s")}
			}
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
			// Resolve each target independently so one unavailable key/credential does not hide others.
			prepareErrors := make(map[string]string)
			for _, id := range ids {
				if err := m.Prepare(a.ctx, []string{id}, o.sudo); err != nil {
					prepareErrors[id] = credentials.Redact(readableError(err), m.Secrets)
				}
			}
			for {
				rows := executor.Batch(a.ctx, inv, ids, n, func(ctx context.Context, id string) executor.Result {
					if problem := prepareErrors[id]; problem != "" {
						return executor.Result{Host: id, ExitCode: -1, Error: problem, ErrorKind: "preparation"}
					}
					return executor.Run(ctx, m, id, definition.command, executor.Options{Timeout: o.timeout, Sudo: o.sudo})
				})
				if a.ctx.Err() != nil {
					return a.ctx.Err()
				}
				if watch > 0 && a.json {
					if err := json.NewEncoder(a.out).Encode(struct {
						Timestamp string `json:"timestamp"`
						executor.Report
					}{time.Now().Format(time.RFC3339), executor.Summarize(rows)}); err != nil {
						return err
					}
				} else if (table || watch > 0) && !a.json {
					if err := a.statsTable(rows, time.Now()); err != nil {
						return err
					}
				} else {
					err := a.results(rows)
					if watch == 0 {
						return err
					}
				}
				if watch == 0 {
					if code := executor.ExitCode(rows); code != 0 {
						return exitError{code}
					}
					return nil
				}
				timer := time.NewTimer(watch)
				select {
				case <-a.ctx.Done():
					timer.Stop()
					return a.ctx.Err()
				case <-timer.C:
				}
			}
		}
		o.flags(cmd)
		root.AddCommand(cmd)
	}
}
