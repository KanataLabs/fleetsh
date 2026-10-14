// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/KanataLabs/fleetsh/internal/actions"
	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/spf13/cobra"
)

type diagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
type diagnosticReport struct {
	Host    string            `json:"host"`
	Config  string            `json:"config"`
	Checks  []diagnosticCheck `json:"checks"`
	Success bool              `json:"success"`
}

func (a *application) doctorCommand(root *cobra.Command) {
	var timeout time.Duration
	cmd := &cobra.Command{Use: "doctor HOST", Short: "Explain the connection route and diagnose SSH login problems", Args: cobra.ExactArgs(1)}
	cmd.Flags().DurationVar(&timeout, "connect-timeout", 0, "connection timeout (default: inventory)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		inv, err := a.load()
		if err != nil {
			return err
		}
		id := args[0]
		h, ok := inv.Hosts[id]
		if !ok {
			return unknownHostError(id)
		}
		if cmd.Flags().Changed("connect-timeout") {
			if timeout <= 0 {
				return fmt.Errorf("--connect-timeout must be positive, for example 30s")
			}
			inv.Defaults.ConnectTimeout = timeout.String()
		}
		configPath, err := a.configPath()
		if err != nil {
			return err
		}
		report := diagnosticReport{Host: id, Config: configPath, Checks: []diagnosticCheck{}}
		var mu sync.Mutex
		add := func(name, status, message string) {
			mu.Lock()
			defer mu.Unlock()
			report.Checks = append(report.Checks, diagnosticCheck{name, status, message})
		}
		finish := func(code int) error {
			mu.Lock()
			snapshot := report
			snapshot.Checks = slices.Clone(report.Checks)
			mu.Unlock()
			return a.printDiagnostics(snapshot, code)
		}
		add("Inventory", "ok", fmt.Sprintf("%s: %s as %s (authentication: %s)", id, h.Address(), h.User, h.Auth))
		route := "Direct SSH to " + h.Address()
		proxy, _ := inv.ProxyFor(h)
		if proxy != "" {
			route = "Proxy " + proxy + " -> " + h.Address()
		}
		if h.ProxyJump != "" {
			route = h.Address()
			for next := h.ProxyJump; next != ""; next = inv.Hosts[next].ProxyJump {
				jump := inv.Hosts[next]
				route = next + " (" + jump.Address() + ") -> " + route
				if p, _ := inv.ProxyFor(jump); p != "" {
					route = "Proxy " + p + " -> " + route
				}
			}
		}
		add("Route", "ok", route)
		if h.Connection == "console-only" {
			add("SSH", "skipped", "This host is console-only. Use your provider's console.")
			return finish(0)
		}
		m, err := a.manager(inv)
		if err != nil {
			return err
		}
		defer m.Close()
		// Diagnosis does not modify trust. First login must verify the key explicitly.
		m.Keys.Trust = nil
		m.Keys.ReadOnly = true
		if err := m.Prepare(a.ctx, []string{id}, false); err != nil {
			add("Local authentication", "failed", credentials.Redact(readableError(err), m.Secrets))
			return finish(2)
		}
		add("Local authentication", "ok", "The configured login method is ready.")
		m.Trace = func(host, stage string) {
			add(stage, "ok", "Completed for "+host)
		}
		r := executor.Run(a.ctx, m, id, actions.Alive, executor.Options{Timeout: 10 * time.Second})
		if r.Success {
			add("Remote command", "ok", "SSH login succeeded and the server executed the health check.")
			report.Success = true
		} else {
			add("SSH check", "failed", remoteFailure(id, r.ErrorKind, r.Error))
			add("Details", "failed", r.Error)
		}
		return finish(executor.ExitCode([]executor.Result{r}))
	}
	root.AddCommand(cmd)
}
func (a *application) printDiagnostics(report diagnosticReport, code int) error {
	if a.json {
		if err := a.writeJSON(report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(a.out, "Connection diagnosis: %s\nConfiguration: %s\n", report.Host, report.Config)
		for _, check := range report.Checks {
			fmt.Fprintf(a.out, "[%s] %s: %s\n", check.Status, check.Name, check.Message)
		}
	}
	if code != 0 {
		return exitError{code}
	}
	return nil
}
