// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/spf13/cobra"
)

type hostOptions struct {
	savePassword, noSavePassword bool
	addGroups, removeGroups      []string
}

func (a *application) secretInput(label string) (string, error) {
	if a.readSecret != nil {
		return a.readSecret(label)
	}
	return credentials.ReadSecretContext(a.ctx, a.in, a.errOut, label)
}

func (a *application) confirmedSecret(label string) (string, error) {
	secret, err := a.secretInput(label)
	if err != nil {
		return "", err
	}
	confirmation, err := a.secretInput("Confirm password")
	if err != nil {
		return "", err
	}
	if secret != confirmation {
		return "", errors.New("passwords do not match")
	}
	if err := credentials.ValidateSecret(secret); err != nil {
		return "", err
	}
	return secret, nil
}

func hostChange(cmd *cobra.Command, action, alias string, supplied inventory.Host, options hostOptions, inv *inventory.Inventory) (inventory.Host, error) {
	old, exists := inv.Hosts[alias]
	if action == "add" {
		if exists {
			return inventory.Host{}, errors.New("host already exists")
		}
		old = supplied
	} else {
		if !exists {
			return inventory.Host{}, errors.New("unknown host")
		}
		previousAuth := old.Auth
		changes := map[string]func(){
			"host": func() { old.Host = supplied.Host }, "port": func() { old.Port = supplied.Port }, "user": func() { old.User = supplied.User },
			"auth": func() { old.Auth = supplied.Auth }, "connection": func() { old.Connection = supplied.Connection },
			"credential": func() { old.Credential = supplied.Credential }, "key": func() { old.Key = supplied.Key },
			"proxy": func() { old.Proxy = supplied.Proxy }, "proxy-jump": func() { old.ProxyJump = supplied.ProxyJump },
			"proxy-credential": func() { old.ProxyCredential = supplied.ProxyCredential }, "sudo-credential": func() { old.SudoCredential = supplied.SudoCredential },
			"groups": func() { old.Groups = supplied.Groups }, "tags": func() { old.Tags = supplied.Tags }, "description": func() { old.Description = supplied.Description },
		}
		for flag, apply := range changes {
			if cmd.Flags().Changed(flag) {
				apply()
			}
		}
		if old.Auth == "password" && previousAuth != "password" && !cmd.Flags().Changed("credential") {
			old.Credential = ""
		}
	}
	if options.noSavePassword {
		old.Credential = ""
	}
	groups := make([]string, 0, len(old.Groups)+len(options.addGroups))
	for _, group := range append(slices.Clone(old.Groups), options.addGroups...) {
		if !slices.Contains(options.removeGroups, group) && !slices.Contains(groups, group) {
			groups = append(groups, group)
		}
	}
	old.Groups = groups
	return old, nil
}

func (a *application) mutateHost(cmd *cobra.Command, alias, action string, supplied inventory.Host, options hostOptions) error {
	if !inventory.ValidName(alias) || alias == "all" {
		return errors.New("invalid or reserved host alias")
	}
	if options.savePassword && options.noSavePassword {
		return errors.New("choose --save-password or --no-save-password")
	}
	if (options.savePassword || options.noSavePassword) && cmd.Flags().Changed("credential") && supplied.Credential != "" {
		return errors.New("choose a credential reference or a password-saving option")
	}
	if cmd.Flags().Changed("groups") && (cmd.Flags().Changed("add-groups") || cmd.Flags().Changed("remove-groups")) {
		return errors.New("choose --groups replacement or --add-groups/--remove-groups")
	}
	for _, group := range append(slices.Clone(options.addGroups), options.removeGroups...) {
		if !inventory.ValidName(group) {
			return errors.New("invalid group name")
		}
		if slices.Contains(options.addGroups, group) && slices.Contains(options.removeGroups, group) {
			return errors.New("cannot add and remove the same group")
		}
	}
	inv, err := a.load()
	if err != nil {
		return err
	}
	initial := inv.Hosts[alias]
	host, err := hostChange(cmd, action, alias, supplied, options, inv)
	if err != nil {
		return err
	}
	inv.Hosts[alias] = host
	if err := inv.Validate(); err != nil {
		return err
	}
	host = inv.Hosts[alias]
	if (options.savePassword || options.noSavePassword) && (host.Auth != "password" || host.Connection != "ssh") {
		return errors.New("password options require an SSH host with password authentication")
	}
	save := options.savePassword || (!options.noSavePassword && host.Connection == "ssh" && host.Auth == "password" && host.Credential == "" &&
		(action == "add" || cmd.Flags().Changed("auth")))
	var ref, secret string
	if save {
		ref, err = credentials.NewReference(alias)
		if err != nil {
			return err
		}
		if _, err := a.store.Get(ref); err == nil {
			return errors.New("generated credential already exists; retry")
		} else if !errors.Is(err, credentials.ErrNotFound) {
			return err
		}
		secret, err = a.confirmedSecret("SSH password for " + host.User + "@" + host.Host)
		if err != nil {
			return err
		}
	}
	saved := false
	err = a.change(func(current *inventory.Inventory) error {
		if save && action == "edit" && !reflect.DeepEqual(current.Hosts[alias], initial) {
			return errors.New("host changed while entering the password; retry")
		}
		host, err := hostChange(cmd, action, alias, supplied, options, current)
		if err != nil {
			return err
		}
		if save {
			host.Credential = ref
		}
		current.Hosts[alias] = host
		if err := current.Validate(); err != nil {
			return err
		}
		if !save {
			return nil
		}
		if err := a.ctx.Err(); err != nil {
			return err
		}
		if _, err := a.store.Get(ref); err == nil {
			return errors.New("generated credential already exists; retry")
		} else if !errors.Is(err, credentials.ErrNotFound) {
			return err
		}
		if err := a.store.Set(ref, secret); err != nil {
			return err
		}
		saved = true
		if !slices.Contains(current.Credentials, ref) {
			current.Credentials = append(current.Credentials, ref)
			slices.Sort(current.Credentials)
		}
		return nil
	})
	if err != nil {
		if saved {
			if cleanupErr := a.store.Delete(ref); cleanupErr != nil && !errors.Is(cleanupErr, credentials.ErrNotFound) {
				return fmt.Errorf("host was not saved; credential cleanup failed; remove %s with credential rm", ref)
			}
		}
		return err
	}
	if saved {
		if a.json {
			return a.writeJSON(struct {
				Host       string `json:"host"`
				Credential string `json:"credential"`
			}{alias, ref})
		}
		fmt.Fprintf(a.out, "Password saved as %s; no secret was written to inventory.\n", ref)
	}
	return nil
}
