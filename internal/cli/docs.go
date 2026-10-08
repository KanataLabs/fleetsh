// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed guides/*.txt
var offlineGuides embed.FS

type offlineTopic struct {
	name, summary, page string
}

var offlineTopics = []offlineTopic{
	{"quickstart", "Start with your first VPS", "getting-started/"},
	{"config", "Inventory paths, defaults and backup", "configuration/#storage"},
	{"passwords", "Automatic password saving and shared credentials", "configuration/#passwords"},
	{"groups", "Multiple groups and incremental host edits", "configuration/#groups"},
	{"selectors", "Target hosts, group unions and tag filters", "getting-started/"},
	{"exec", "Parallel execution, quoting, sudo and JSON", "getting-started/"},
	{"proxies", "Global/per-host SOCKS5 and HTTP/HTTPS CONNECT, ProxyJump", "configuration/"},
	{"patching", "Preview and apply package updates across the fleet", "cases/patch-all-vps/"},
	{"monitoring", "SSH liveness and Linux CPU/memory/swap/disk snapshots", "getting-started/"},
	{"forwarding", "Local, remote and dynamic SSH tunnel configuration", "configuration/#forwarding"},
	{"troubleshooting", "Exit codes, host keys, authentication and keyrings", "security/"},
}

func (a *application) documentationCommand(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "docs [TOPIC]",
		Short: "Read offline guides embedded in the binary",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if _, err := fmt.Fprint(a.out, "Offline guides (English)\n\nUsage: fleetsh docs TOPIC\n\n"); err != nil {
					return err
				}
				for _, topic := range offlineTopics {
					if _, err := fmt.Fprintf(a.out, "  %-16s %s\n", topic.name, topic.summary); err != nil {
						return err
					}
				}
				_, err := fmt.Fprintln(a.out, "\nCommand help: fleetsh COMMAND --help\n\nOnline editions:\n  English: "+documentationURL+"\n  Japanese: https://kanatalabs.github.io/fleetsh/ja/\n  Chinese: https://kanatalabs.github.io/fleetsh/zh/")
				return err
			}
			for _, topic := range offlineTopics {
				if args[0] != topic.name {
					continue
				}
				content, err := offlineGuides.ReadFile("guides/" + topic.name + ".txt")
				if err != nil {
					return fmt.Errorf("read offline guide: %w", err)
				}
				if _, err := fmt.Fprint(a.out, string(content)); err != nil {
					return err
				}
				_, err = fmt.Fprintln(a.out, "\nDocumentation: "+documentationURL+topic.page)
				return err
			}
			return fmt.Errorf("unknown documentation topic %q; run fleetsh docs to list topics", args[0])
		},
	}
	for _, topic := range offlineTopics {
		cmd.ValidArgs = append(cmd.ValidArgs, topic.name)
	}
	root.AddCommand(cmd)
}
