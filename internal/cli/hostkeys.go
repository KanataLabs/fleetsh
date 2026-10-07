// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func (a *application) hostkeyCommands(root *cobra.Command) {
	parent := &cobra.Command{Use: "hostkey", Short: "Inspect or reset explicitly trusted host keys"}
	parent.AddCommand(&cobra.Command{Use: "show HOST", Short: "Show trusted fingerprints for the target address", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		inv, err := a.load()
		if err != nil {
			return err
		}
		h, ok := inv.Hosts[args[0]]
		if !ok {
			return errors.New("unknown host")
		}
		keys, err := a.keys()
		if err != nil {
			return err
		}
		records, err := keys.Lookup(h.Address())
		if err != nil {
			return err
		}
		for _, record := range records {
			fmt.Fprintf(a.out, "%s %s %s\n", args[0], record.Key.Type(), ssh.FingerprintSHA256(record.Key))
		}
		if len(records) == 0 {
			fmt.Fprintln(a.out, "No trusted key; connect interactively to verify the fingerprint.")
		}
		return nil
	}})
	var yes bool
	reset := &cobra.Command{Use: "reset HOST", Short: "Remove trust after independent fingerprint verification", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		inv, err := a.load()
		if err != nil {
			return err
		}
		h, ok := inv.Hosts[args[0]]
		if !ok {
			return errors.New("unknown host")
		}
		if !yes && !a.confirm("Remove trusted key for "+args[0]+"?") {
			return errors.New("host key reset canceled")
		}
		keys, err := a.keys()
		if err != nil {
			return err
		}
		return keys.Reset(a.ctx, h.Address())
	}}
	reset.Flags().BoolVar(&yes, "yes", false, "confirm removal of trust")
	parent.AddCommand(reset)
	root.AddCommand(parent)
}
