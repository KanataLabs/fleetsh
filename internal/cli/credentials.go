// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
)

func (a *application) credentialCommands(root *cobra.Command) {
	parent := &cobra.Command{Use: "credential", Short: "Manage secrets in the OS credential store"}
	var replace bool
	add := &cobra.Command{Use: "add REFERENCE", Short: "Save a secret using hidden terminal input", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		if !inventory.ValidName(ref) {
			return errors.New("invalid credential reference")
		}
		if _, err := a.load(); err != nil {
			return err
		}
		if _, err := a.store.Get(ref); err == nil && !replace {
			return errors.New("credential exists; use --replace to overwrite")
		} else if err != nil && !errors.Is(err, credentials.ErrNotFound) {
			return err
		}
		s, err := credentials.ReadSecretContext(a.ctx, a.in, a.errOut, "Secret (proxy: username:password)")
		if err != nil {
			return err
		}
		confirmation, err := credentials.ReadSecretContext(a.ctx, a.in, a.errOut, "Confirm")
		if err != nil {
			return err
		}
		if s != confirmation {
			return errors.New("secrets do not match")
		}
		saved := false
		err = a.change(func(inv *inventory.Inventory) error {
			if _, e := a.store.Get(ref); e == nil && !replace {
				return errors.New("credential exists; use --replace to overwrite")
			} else if e != nil && !errors.Is(e, credentials.ErrNotFound) {
				return e
			}
			if e := a.store.Set(ref, s); e != nil {
				return e
			}
			saved = true
			if !slices.Contains(inv.Credentials, ref) {
				inv.Credentials = append(inv.Credentials, ref)
				sort.Strings(inv.Credentials)
			}
			return nil
		})
		if err != nil && saved {
			return errors.New("secret was saved but its local reference index could not be updated; use credential rm to clean up")
		}
		if err == nil {
			fmt.Fprintln(a.out, "Credential saved; no secret was written to inventory.")
		}
		return err
	}}
	add.Flags().BoolVar(&replace, "replace", false, "overwrite an existing secret")
	parent.AddCommand(add)
	parent.AddCommand(&cobra.Command{Use: "ls", Short: "List credential references tracked by this inventory", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		inv, err := a.load()
		if err != nil {
			return err
		}
		refs := append([]string{}, inv.Credentials...)
		for _, h := range inv.Hosts {
			for _, ref := range []string{h.Credential, h.ProxyCredential, h.SudoCredential} {
				if ref != "" && !slices.Contains(refs, ref) {
					refs = append(refs, ref)
				}
			}
		}
		sort.Strings(refs)
		if a.json {
			return a.writeJSON(refs)
		}
		for _, ref := range refs {
			fmt.Fprintln(a.out, ref)
		}
		return nil
	}})
	parent.AddCommand(&cobra.Command{Use: "rm REFERENCE", Short: "Delete a secret and its tracked reference", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		if !inventory.ValidName(ref) {
			return errors.New("invalid credential reference")
		}
		return a.change(func(inv *inventory.Inventory) error {
			if err := a.store.Delete(ref); err != nil && !errors.Is(err, credentials.ErrNotFound) {
				return err
			}
			inv.Credentials = slices.DeleteFunc(inv.Credentials, func(s string) bool { return s == ref })
			return nil
		})
	}})
	root.AddCommand(parent)
}
