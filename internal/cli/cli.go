// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KanataLabs/fleetsh/internal/config"
	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/transport"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type application struct {
	ctx          context.Context
	in           io.Reader
	out, errOut  io.Writer
	store        credentials.Store
	path         string
	json         bool
	promptFailed bool
	readSecret   func(string) (string, error)
	readLine     func(string) (string, error)
}
type exitError struct{ code int }

func (e exitError) Error() string { return "" }
func Run(args []string, version string, out, errOut io.Writer) int {
	return RunContext(context.Background(), args, version, os.Stdin, out, errOut)
}
func RunContext(ctx context.Context, args []string, version string, in io.Reader, out, errOut io.Writer) int {
	a := &application{ctx: ctx, in: in, out: out, errOut: errOut, store: credentials.Native{}}
	root := a.root(version)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	if ctx.Err() != nil {
		return 1
	}
	var code exitError
	if errors.As(err, &code) {
		return code.code
	}
	fmt.Fprintf(errOut, "fleetsh: %s\n", readableError(err))
	var usage usageError
	if errors.As(err, &usage) {
		fmt.Fprintln(errOut)
		previousOut := usage.command.OutOrStdout()
		usage.command.SetOut(errOut)
		_ = usage.command.Help()
		usage.command.SetOut(previousOut)
	}
	return 2
}
func (a *application) root(version string) *cobra.Command {
	root := &cobra.Command{Use: "fleetsh", Short: "Lightweight, agentless VPS fleet management",
		Args: cobra.NoArgs, SilenceErrors: true, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	root.SetIn(a.in)
	root.SetOut(a.out)
	root.SetErr(a.errOut)
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().StringVar(&a.path, "config", "", "inventory file (default: OS user config directory)")
	root.PersistentFlags().BoolVar(&a.json, "json", false, "JSON for inventory, diagnostics, batch results and dry runs")
	root.Version = version
	root.SetVersionTemplate("fleetsh {{.Version}}\n")
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print the build version", Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { fmt.Fprintf(a.out, "fleetsh %s\n", version) }})
	root.AddCommand(&cobra.Command{Use: "init", Short: "Create a private example config without overwriting files", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		path, err := a.configPath()
		if err != nil {
			return err
		}
		if err = config.Init(path); err != nil {
			return err
		}
		fmt.Fprintf(a.out, "Created %s\nNo credentials were stored.\n", path)
		return nil
	}})
	a.inventoryCommands(root)
	a.credentialCommands(root)
	a.remoteCommands(root)
	a.monitoringCommands(root)
	a.forwardingCommand(root)
	a.hostkeyCommands(root)
	a.transferCommands(root)
	a.interoperabilityCommands(root)
	a.runCommand(root)
	a.doctorCommand(root)
	a.documentationCommand(root)
	a.completionCommand(root)
	documentCommands(root)
	return root
}
func (a *application) configPath() (string, error) {
	if a.path != "" {
		return transport.ExpandPath(a.path)
	}
	return config.DefaultPath()
}
func (a *application) load() (*inventory.Inventory, error) {
	path, err := a.configPath()
	if err != nil {
		return nil, err
	}
	inv, err := inventory.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, missingInventoryError{path: path, cause: err}
	}
	if err != nil {
		return nil, fmt.Errorf("cannot load inventory %s; check --config PATH and the file's content/access: %w", path, err)
	}
	return inv, nil
}

type missingInventoryError struct {
	path  string
	cause error
}

func (e missingInventoryError) Error() string {
	return fmt.Sprintf("inventory not found: %s\nNext: add a host to create it automatically, or run fleetsh init for an empty inventory.\nFor a custom inventory, use the same --config PATH on every command.", e.path)
}
func (e missingInventoryError) Unwrap() error { return e.cause }

func (a *application) change(fn func(*inventory.Inventory) error) error {
	return a.updateInventory(fn, false)
}

func (a *application) changeOrCreate(fn func(*inventory.Inventory) error) error {
	return a.updateInventory(fn, true)
}

func (a *application) updateInventory(fn func(*inventory.Inventory) error, create bool) error {
	path, err := a.configPath()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	if create {
		return inventory.UpdateOrCreate(ctx, path, fn)
	}
	return inventory.Update(ctx, path, fn)
}
func (a *application) writeJSON(value any) error {
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
func (a *application) isTerminal() bool {
	f, ok := a.in.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
func (a *application) confirm(message string) bool { return a.confirmContext(a.ctx, message) }
func (a *application) confirmContext(ctx context.Context, message string) bool {
	if !a.isTerminal() {
		fmt.Fprintln(a.errOut, "Interactive confirmation required; use --yes for this action.")
		return false
	}
	fmt.Fprint(a.errOut, message+" [y/N]: ")
	if a.promptFailed {
		return false
	}
	answer := make(chan bool, 1)
	go func() {
		text, err := bufio.NewReader(a.in).ReadString('\n')
		answer <- err == nil && strings.EqualFold(strings.TrimSpace(text), "y")
	}()
	select {
	case yes := <-answer:
		return yes
	case <-ctx.Done():
		a.promptFailed = true
		fmt.Fprintln(a.errOut, "\nConfirmation timed out or canceled.")
		return false
	}
}
func (a *application) keys() (*transport.HostKeys, error) {
	path, err := a.configPath()
	if err != nil {
		return nil, err
	}
	return &transport.HostKeys{Path: filepath.Join(filepath.Dir(path), "known_hosts"), Trust: func(ctx context.Context, addr string, key ssh.PublicKey) bool {
		if !a.isTerminal() {
			return false
		}
		return a.confirmContext(ctx, fmt.Sprintf("Trust %s %s key %s?", addr, key.Type(), ssh.FingerprintSHA256(key)))
	}}, nil
}
func (a *application) manager(inv *inventory.Inventory) (*transport.Manager, error) {
	keys, err := a.keys()
	if err != nil {
		return nil, err
	}
	return &transport.Manager{Inventory: inv, Keys: keys, Store: a.store, Prompt: func(label string) (string, error) { return a.secretInput(label) }}, nil
}
