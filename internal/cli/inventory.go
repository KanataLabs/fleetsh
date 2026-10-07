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
		cmd := &cobra.Command{Use: action + " HOST", Short: action + " a host", Args: cobra.ExactArgs(1)}
		cmd.Flags().StringVar(&h.Host, "host", "", "hostname or IP")
		cmd.Flags().IntVar(&h.Port, "port", 22, "SSH port")
		cmd.Flags().StringVar(&h.User, "user", "", "SSH username")
		cmd.Flags().StringVar(&h.Auth, "auth", "agent", "agent, key or password")
		cmd.Flags().StringVar(&h.Connection, "connection", "ssh", "ssh or console-only")
		cmd.Flags().StringVar(&h.Credential, "credential", "", "password/key-passphrase reference")
		cmd.Flags().StringVar(&h.Key, "key", "", "private key file path")
		cmd.Flags().StringVar(&h.Proxy, "proxy", "", "SOCKS5 URL without credentials")
		cmd.Flags().StringVar(&h.ProxyJump, "proxy-jump", "", "jump-host alias")
		cmd.Flags().StringVar(&h.ProxyCredential, "proxy-credential", "", "username:password credential reference")
		cmd.Flags().StringVar(&h.SudoCredential, "sudo-credential", "", "sudo password reference")
		cmd.Flags().StringSliceVar(&h.Groups, "groups", nil, "comma-separated groups")
		cmd.Flags().StringSliceVar(&h.Tags, "tags", nil, "comma-separated tags")
		cmd.Flags().StringVar(&h.Description, "description", "", "host description")
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if !inventory.ValidName(args[0]) || args[0] == "all" {
				return errors.New("invalid or reserved host alias")
			}
			return a.change(func(inv *inventory.Inventory) error {
				old, exists := inv.Hosts[args[0]]
				if action == "add" {
					if exists {
						return errors.New("host already exists")
					}
					inv.Hosts[args[0]] = h
					return nil
				}
				if !exists {
					return errors.New("unknown host")
				}
				changes := map[string]func(){
					"host": func() { old.Host = h.Host }, "port": func() { old.Port = h.Port }, "user": func() { old.User = h.User },
					"auth": func() { old.Auth = h.Auth }, "connection": func() { old.Connection = h.Connection },
					"credential": func() { old.Credential = h.Credential }, "key": func() { old.Key = h.Key },
					"proxy": func() { old.Proxy = h.Proxy }, "proxy-jump": func() { old.ProxyJump = h.ProxyJump },
					"proxy-credential": func() { old.ProxyCredential = h.ProxyCredential }, "sudo-credential": func() { old.SudoCredential = h.SudoCredential },
					"groups": func() { old.Groups = h.Groups }, "tags": func() { old.Tags = h.Tags }, "description": func() { old.Description = h.Description },
				}
				for flag, apply := range changes {
					if cmd.Flags().Changed(flag) {
						apply()
					}
				}
				inv.Hosts[args[0]] = old
				return nil
			})
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
