// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func (a *application) completionCommand(root *cobra.Command) {
	cmd := &cobra.Command{Use: "completion SHELL", Short: "Generate Bash, Zsh, Fish or PowerShell completion", Args: cobra.ExactArgs(1), ValidArgs: []string{"bash", "zsh", "fish", "powershell"}}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletionV2(a.out, true)
		case "zsh":
			return root.GenZshCompletion(a.out)
		case "fish":
			return root.GenFishCompletion(a.out, true)
		case "powershell":
			return root.GenPowerShellCompletionWithDesc(a.out)
		default:
			return usageError{cmd, fmt.Errorf("Unsupported shell %q. Choose bash, zsh, fish or powershell.", args[0])}
		}
	}
	root.AddCommand(cmd)
	for _, child := range root.Commands() {
		switch child.Name() {
		case "ssh", "edit", "rm", "show", "doctor", "forward", "hostkey":
			child.ValidArgsFunction = a.completeHost(false)
			if child.Name() == "hostkey" {
				for _, sub := range child.Commands() {
					sub.ValidArgsFunction = a.completeHost(false)
				}
			}
		case "ls", "exec", "alive", "stats", "update", "reboot", "reboot-required", "upload", "download":
			child.ValidArgsFunction = a.completeHost(true)
		case "run":
			child.ValidArgsFunction = func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
				if len(args) == 0 {
					return a.completeHost(true)(cmd, args, prefix)
				}
				if len(args) != 1 {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				inv, err := a.load()
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				var names []string
				for name := range inv.Commands {
					if strings.HasPrefix(name, prefix) {
						names = append(names, name)
					}
				}
				slices.Sort(names)
				return names, cobra.ShellCompDirectiveNoFileComp
			}
		}
	}
}
func (a *application) completeHost(groups bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveDefault
		}
		inv, err := a.load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		candidates := map[string]bool{}
		if groups {
			candidates["@all"] = true
		}
		for id, h := range inv.Hosts {
			candidates[id] = true
			if groups {
				for _, g := range h.Groups {
					candidates["@"+g] = true
				}
			}
		}
		var result []string
		// Complete the last segment of a comma-separated selector.
		segment := prefix
		lead := ""
		if i := strings.LastIndex(prefix, ","); i >= 0 {
			lead = prefix[:i+1]
			segment = prefix[i+1:]
		}
		for value := range candidates {
			if strings.HasPrefix(value, segment) {
				result = append(result, lead+value)
			}
		}
		slices.Sort(result)
		return result, cobra.ShellCompDirectiveNoFileComp
	}
}
