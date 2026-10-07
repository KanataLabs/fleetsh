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
	"sync/atomic"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/config"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestInventoryCommandsEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Init(path); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	invoke := func(args ...string) int {
		out.Reset()
		errOut.Reset()
		return RunContext(context.Background(), append([]string{"--config", path}, args...), "test", strings.NewReader(""), &out, &errOut)
	}
	if code := invoke("add", "hk1", "--host", "hk.example", "--user", "ubuntu", "--groups", "asia,web", "--tags", "prod"); code != 0 {
		t.Fatalf("add: %d %s", code, &errOut)
	}
	if code := invoke("edit", "hk1", "--port", "2222"); code != 0 {
		t.Fatal(errOut.String())
	}
	if code := invoke("ls", "@web", "--json"); code != 0 {
		t.Fatal(errOut.String())
	}
	var rows []hostRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil || len(rows) != 1 || rows[0].Port != 2222 || rows[0].User != "ubuntu" {
		t.Fatalf("bad inventory result: %s %v", out.String(), err)
	}
	before, _ := os.ReadFile(path)
	if code := invoke("edit", "hk1", "--auth", "key"); code != 2 {
		t.Fatalf("invalid edit accepted: %d", code)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed edit modified config")
	}
	if code := invoke("rm", "hk1"); code != 0 {
		t.Fatal(errOut.String())
	}
	if code := invoke("ls", "--json"); code != 0 || strings.TrimSpace(out.String()) != "[]" {
		t.Fatalf("empty inventory: %d %s", code, out.String())
	}
}
func TestRemoteCLIJSONAndUpdatePreflight(t *testing.T) {
	var dispatched atomic.Int32
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		if command == "uptime" {
			return testutil.Reply{Out: "fixture uptime\n"}
		}
		if strings.Contains(command, "/etc/os-release") {
			return testutil.Reply{Out: "ubuntu\n"}
		}
		dispatched.Add(1)
		return testutil.Reply{Out: "updated"}
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	host := server.Host()
	host.SudoCredential = "sudo"
	inv := inventory.Inventory{Hosts: map[string]inventory.Host{"a": host, "console": {Host: "console.example", Connection: "console-only"}}}
	if err := inv.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := toml.Marshal(inv)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "known_hosts"), []byte(knownhosts.Line([]string{knownhosts.Normalize(server.Address)}, server.Key.PublicKey())+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	invoke := func(args ...string) error {
		out.Reset()
		errOut.Reset()
		app := &application{ctx: context.Background(), in: strings.NewReader(""), out: &out, errOut: &errOut, store: testutil.Store{"login": testutil.Password, "sudo": "ubuntu"}, path: path}
		root := app.root("test")
		root.SetArgs(append([]string{"--config", path}, args...))
		return root.Execute()
	}
	if err := invoke("exec", "@all", "uptime", "--json", "--parallel", "2"); err != nil {
		t.Fatal(err)
	}
	var report executor.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.Summary.Success != 1 || report.Summary.Skipped != 1 {
		t.Fatalf("JSON result: %v %s", err, out.String())
	}
	if err := invoke("update", "a", "--dry-run", "--json"); err != nil {
		t.Fatal(err)
	}
	if dispatched.Load() != 0 || !strings.Contains(out.String(), "apt-get upgrade") {
		t.Fatal("dry run dispatched an update or missed plan")
	}
	if err := invoke("update", "a"); err == nil || dispatched.Load() != 0 {
		t.Fatal("non-terminal update did not require confirmation")
	}
	if err := invoke("update", "a", "--yes", "--sudo"); err != nil || dispatched.Load() != 1 {
		t.Fatalf("confirmed update failed: %v %s", err, out.String())
	}
}
func TestRebootDryRunNeverConnects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Init(path); err != nil {
		t.Fatal(err)
	}
	if err := inventory.Update(context.Background(), path, func(inv *inventory.Inventory) error {
		inv.Hosts["a"] = inventory.Host{Host: "unreachable.example", User: "root", Auth: "password"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := RunContext(context.Background(), []string{"--config", path, "reboot", "a", "--dry-run", "--json"}, "test", strings.NewReader(""), &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "\"dry_run\": true") {
		t.Fatalf("dry run tried authentication: %d %s", code, &errOut)
	}
}
