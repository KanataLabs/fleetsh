// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
)

type hostRow struct {
	ID string `json:"id"`
	inventory.Host
}

func (a *application) inventoryCommands(root *cobra.Command) {
	var tag string
	ls := &cobra.Command{Use: "ls [SELECTOR]", Short: "List hosts, groups and tags", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		inv, err := a.load()
		if err != nil {
			return err
		}
		selector := "all"
		if len(args) > 0 {
			selector = args[0]
		}
		ids, err := inv.Select(selector, tag)
		if len(inv.Hosts) == 0 && (selector == "all" || selector == "@all") {
			ids = []string{}
			err = nil
		}
		if err != nil {
			return err
		}
		rows := []hostRow{}
		for _, id := range ids {
			rows = append(rows, hostRow{id, inv.Hosts[id]})
		}
		if a.json {
			return a.writeJSON(rows)
		}
		w := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tHOST\tUSER\tAUTH\tGROUPS\tTAGS\tCONNECTION")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.ID, r.Address(), r.User, r.Auth, strings.Join(r.Groups, ","), strings.Join(r.Tags, ","), r.Connection)
		}
		return w.Flush()
	}}
	ls.Flags().StringVar(&tag, "tag", "", "filter by tag")
	root.AddCommand(ls)
	root.AddCommand(&cobra.Command{Use: "show HOST", Short: "Show host configuration without secret values", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		inv, err := a.load()
		if err != nil {
			return err
		}
		h, ok := inv.Hosts[args[0]]
		if !ok {
			return errors.New("unknown host")
		}
		return a.writeJSON(hostRow{args[0], h})
	}})
	for _, action := range []string{"add", "edit"} {
		h := inventory.Host{}
		options := hostOptions{}
		cmd := &cobra.Command{Use: action + " HOST", Short: action + " a host", Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if action == "add" {
				return requiredHostOptions(h)
			}
			return nil
		}}
		cmd.Flags().StringVar(&h.Host, "host", "", "hostname or IP (required for add; edit keeps it when omitted)")
		cmd.Flags().IntVar(&h.Port, "port", 22, "SSH port")
		cmd.Flags().StringVar(&h.User, "user", "", "SSH username (required for SSH; edit keeps the saved value)")
		cmd.Flags().StringVar(&h.Auth, "auth", "agent", "agent, key or password")
		cmd.Flags().StringVar(&h.Connection, "connection", "ssh", "ssh or console-only")
		cmd.Flags().StringVar(&h.Credential, "credential", "", "password/key-passphrase reference")
		cmd.Flags().StringVar(&h.Key, "key", "", "private key file path (required with --auth key; edit keeps the saved value)")
		cmd.Flags().StringVar(&h.Proxy, "proxy", "", "SOCKS5/HTTP/HTTPS proxy URL, direct, or empty to inherit")
		cmd.Flags().StringVar(&h.ProxyJump, "proxy-jump", "", "jump-host alias")
		cmd.Flags().StringVar(&h.ProxyCredential, "proxy-credential", "", "username:password credential reference")
		cmd.Flags().StringVar(&h.SudoCredential, "sudo-credential", "", "sudo password reference")
		cmd.Flags().StringSliceVar(&h.Groups, "groups", nil, "all group memberships (edit replaces the list)")
		cmd.Flags().StringSliceVar(&h.Tags, "tags", nil, "comma-separated tags")
		cmd.Flags().StringVar(&h.Description, "description", "", "host description")
		cmd.Flags().BoolVar(&options.savePassword, "save-password", false, "prompt and save a new SSH password for this host")
		cmd.Flags().BoolVar(&options.noSavePassword, "no-save-password", false, "prompt on each connection instead of storing the SSH password")
		if action == "add" {
			_ = cmd.MarkFlagRequired("host")
		}
		if action == "edit" {
			cmd.Flags().StringSliceVar(&options.addGroups, "add-groups", nil, "add group memberships without replacing existing groups")
			cmd.Flags().StringSliceVar(&options.removeGroups, "remove-groups", nil, "remove selected group memberships")
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return a.mutateHost(cmd, args[0], action, h, options)
		}
		root.AddCommand(cmd)
	}
	root.AddCommand(&cobra.Command{Use: "rm HOST", Short: "Remove a host from inventory", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.change(func(inv *inventory.Inventory) error {
			if _, ok := inv.Hosts[args[0]]; !ok {
				return errors.New("unknown host")
			}
			delete(inv.Hosts, args[0])
			return nil
		})
	}})
}
