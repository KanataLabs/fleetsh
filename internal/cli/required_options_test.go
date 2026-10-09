// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/inventory"
)

func TestAddRequiredOptionsShowHelpBeforeInventoryLoading(t *testing.T) {
	for _, test := range []struct {
		name    string
		flags   []string
		missing string
	}{
		{"alias only", nil, "--host, --user (required for SSH hosts)"},
		{"address only", []string{"--host", "vps.example"}, "--user (required for SSH hosts)"},
		{"username only", []string{"--user", "ubuntu"}, "--host"},
		{"password without address", []string{"--user", "ubuntu", "--auth", "password"}, "--host"},
		{"empty address", []string{"--host=", "--user", "ubuntu"}, "--host"},
		{"blank username", []string{"--host", "vps.example", "--user", " "}, "--user (required for SSH hosts)"},
		{"missing key", []string{"--host", "vps.example", "--user", "ubuntu", "--auth", "key"}, "--key (required with --auth key)"},
		{"missing username and key", []string{"--host", "vps.example", "--auth", "key"}, "--user (required for SSH hosts), --key (required with --auth key)"},
		{"blank key", []string{"--host", "vps.example", "--user", "ubuntu", "--auth", "key", "--key", " "}, "--key (required with --auth key)"},
		{"console without address", []string{"--connection", "console-only"}, "--host"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing.toml")
			args := append([]string{"--config", path, "--json", "add", "racknerd"}, test.flags...)
			var out, errOut bytes.Buffer
			code := RunContext(context.Background(), args, "test", strings.NewReader(""), &out, &errOut)
			firstLine := strings.SplitN(errOut.String(), "\n", 2)[0]
			want := "fleetsh: missing or empty required options: " + test.missing
			if code != 2 || out.Len() != 0 || firstLine != want {
				t.Fatalf("code=%d stdout=%q stderr=%q; want %q", code, &out, &errOut, want)
			}
			for _, text := range []string{"Required arguments/options", "Local inventory alias", "Usage:", "fleetsh add HOST", "Examples:", "Flags:"} {
				if !strings.Contains(errOut.String(), text) {
					t.Fatalf("missing help %q: %s", text, &errOut)
				}
			}
			if strings.Contains(errOut.String(), "missing.toml") || strings.Contains(errOut.String(), "\nSSH password for ") {
				t.Fatalf("required validation loaded inventory or prompted: %s", &errOut)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("required validation touched inventory: %v", err)
			}
		})
	}
}

func TestEditRequiredValuesUseMergedHostAndPreserveInventoryOnFailure(t *testing.T) {
	for _, test := range []struct {
		name       string
		connection string
		flags      []string
		missing    string
	}{
		{"clear address", "ssh", []string{"--host="}, "--host"},
		{"clear SSH user", "ssh", []string{"--user="}, "--user (required for SSH hosts)"},
		{"switch to key", "ssh", []string{"--auth", "key"}, "--key (required with --auth key)"},
		{"switch console to SSH", "console-only", []string{"--connection", "ssh"}, "--user (required for SSH hosts)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := emptyInventory(t)
			flags := []string{"add", "racknerd", "--host", "vps.example", "--connection", test.connection}
			if test.connection == "ssh" {
				flags = append(flags, "--user", "ubuntu")
			}
			if _, err := hostCommand(t, path, &recordingStore{values: map[string]string{}}, nil, flags...); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			args := append([]string{"--config", path, "edit", "racknerd"}, test.flags...)
			var out, errOut bytes.Buffer
			code := RunContext(context.Background(), args, "test", strings.NewReader(""), &out, &errOut)
			firstLine := strings.SplitN(errOut.String(), "\n", 2)[0]
			if code != 2 || out.Len() != 0 || firstLine != "fleetsh: missing or empty required options: "+test.missing ||
				!strings.Contains(errOut.String(), "fleetsh edit HOST") {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, &out, &errOut)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("invalid edit changed inventory: %v", err)
			}
		})
	}
}

func TestConsoleAndKeyHostsCanReuseExistingRequiredValues(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{}}
	for _, args := range [][]string{
		{"add", "console", "--host", "console.example", "--connection", "console-only"},
		{"edit", "console", "--description", "Provider console"},
		{"add", "keyhost", "--host", "vps.example", "--user", "ubuntu", "--auth", "key", "--key", "fixture.pem"},
		{"edit", "keyhost", "--add-groups", "web"},
	} {
		if _, err := hostCommand(t, path, store, func(string) (string, error) { t.Fatal("unexpected password prompt"); return "", nil }, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	inv, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Hosts["console"].User != "" || inv.Hosts["console"].Connection != "console-only" {
		t.Fatal("console-only host unexpectedly required SSH fields")
	}
	key := inv.Hosts["keyhost"]
	if key.Host != "vps.example" || key.User != "ubuntu" || key.Key != "fixture.pem" || len(key.Groups) != 1 {
		t.Fatalf("edit failed to reuse saved required fields: %+v", key)
	}
	if len(store.values) != 0 {
		t.Fatal("non-password inventory changes touched credential store")
	}
}

func TestInvalidHostOptionShowsHelpAndDoesNotWrite(t *testing.T) {
	path := emptyInventory(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := RunContext(context.Background(), []string{"--config", path, "add", "racknerd", "--host", "vps.example", "--user", "ubuntu", "--port", "99999"},
		"test", strings.NewReader(""), &out, &errOut)
	if code != 2 || out.Len() != 0 || !strings.Contains(errOut.String(), "invalid port") || !strings.Contains(errOut.String(), "fleetsh add HOST") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, &out, &errOut)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("invalid options changed inventory")
	}
}

func TestConflictingHostOptionsShowHelpBeforeInventoryLoading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	var out, errOut bytes.Buffer
	code := RunContext(context.Background(), []string{"--config", path, "add", "racknerd", "--host", "vps.example", "--user", "ubuntu", "--auth", "password", "--save-password", "--no-save-password"},
		"test", strings.NewReader(""), &out, &errOut)
	if code != 2 || out.Len() != 0 || !strings.HasPrefix(errOut.String(), "fleetsh: choose --save-password or --no-save-password\n") || !strings.Contains(errOut.String(), "fleetsh add HOST") || strings.Contains(errOut.String(), "missing.toml") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, &out, &errOut)
	}
}
