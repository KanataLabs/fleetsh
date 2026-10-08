// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/actions"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestMonitoringDefaultsToAllHostsAndUsesSSH(t *testing.T) {
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		switch command {
		case actions.Alive:
			return testutil.Reply{Out: "alive\n"}
		case actions.Stats:
			return testutil.Reply{Out: "USER: fixture\nCPU: 1.0%\nMEMORY: 1 / 2 MiB\nSWAP: 0 / 0 MiB\nDISK: fixture\n"}
		}
		return testutil.Reply{Code: 1}
	})
	inv := inventory.Inventory{Hosts: map[string]inventory.Host{"a": server.Host(), "console": {Host: "console.example", Connection: "console-only"}}}
	if err := inv.Validate(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	data, err := toml.Marshal(inv)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "known_hosts"), []byte(knownhosts.Line([]string{knownhosts.Normalize(server.Address)}, server.Key.PublicKey())+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"alive", "stats", "monitor"} {
		t.Run(command, func(t *testing.T) {
			var out, errOut bytes.Buffer
			app := &application{ctx: t.Context(), in: strings.NewReader(""), out: &out, errOut: &errOut, store: testutil.Store{"login": testutil.Password}}
			root := app.root("test")
			root.SetArgs([]string{"--config", path, command, "--json"})
			if err := root.ExecuteContext(t.Context()); err != nil {
				t.Fatal(err)
			}
			var report executor.Report
			if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.Summary.Success != 1 || report.Summary.Skipped != 1 {
				t.Fatalf("report=%s err=%v", out.String(), err)
			}
			if command != "alive" && !strings.Contains(report.Results[0].Stdout, "SWAP:") {
				t.Fatal("stats payload missing")
			}
		})
	}
}
func TestForwardDryRunRequiresNoCredentialOrConnection(t *testing.T) {
	inv := inventory.Inventory{Hosts: map[string]inventory.Host{"a": {Host: "unreachable.example", User: "root", Auth: "password", Forwards: []inventory.Forward{
		{Name: "web", Type: "local", Listen: "127.0.0.1:8080", Destination: "127.0.0.1:80"},
		{Name: "socks", Type: "dynamic", Listen: "127.0.0.1:1080"},
	}}}}
	if err := inv.Validate(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	data, _ := toml.Marshal(inv)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"forward", "a", "--dry-run", "--json"}, {"forward", "a", "web", "--dry-run", "--json"}} {
		var out, errOut bytes.Buffer
		if code := RunContext(t.Context(), append([]string{"--config", path}, args...), "test", strings.NewReader(""), &out, &errOut); code != 0 {
			t.Fatalf("dry run=%d %s", code, errOut.String())
		}
		var plan struct {
			DryRun   bool                `json:"dry_run"`
			Forwards []inventory.Forward `json:"forwards"`
		}
		if err := json.Unmarshal(out.Bytes(), &plan); err != nil || !plan.DryRun || len(plan.Forwards) == 0 {
			t.Fatalf("plan=%s", out.String())
		}
		if len(args) == 5 && (len(plan.Forwards) != 1 || plan.Forwards[0].Name != "web") {
			t.Fatal("named selection failed")
		}
	}
	var out, errOut bytes.Buffer
	if code := RunContext(t.Context(), []string{"--config", path, "forward", "a", "unknown", "--dry-run"}, "test", strings.NewReader(""), &out, &errOut); code != 2 {
		t.Fatal("unknown profile accepted")
	}
}
