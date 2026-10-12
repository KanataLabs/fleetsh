// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/config"
	"github.com/KanataLabs/fleetsh/internal/inventory"
)

func TestAddCreatesInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "nested", "config.toml")
	store := &recordingStore{values: map[string]string{}}
	prompts := 0
	out, err := hostCommand(t, path, store, func(string) (string, error) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("inventory created before password confirmation")
		}
		prompts++
		return "auto-create-fixture", nil
	}, "add", "racknerd", "--host", "vps.example", "--user", "root", "--json")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	h := inv.Hosts["racknerd"]
	if prompts != 2 || h.Auth != "password" || store.values[h.Credential] != "auto-create-fixture" || inv.Defaults.Parallel != 10 {
		t.Fatal("first add did not initialize and save password reference")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "auto-create-fixture") || strings.Contains(string(data), "auto-create-fixture") {
		t.Fatal("secret leaked")
	}
	if _, err := hostCommand(t, path, store, nil, "add", "second", "--host", "second.example", "--user", "root", "--auth", "agent"); err != nil {
		t.Fatal(err)
	}
	inv, err = inventory.Load(path)
	if err != nil || len(inv.Hosts) != 2 || inv.Hosts["racknerd"].Credential != h.Credential {
		t.Fatal("second add lost first host")
	}
	before, _ := os.ReadFile(path)
	_, err = hostCommand(t, path, store, nil, "add", "racknerd", "--host", "changed.example", "--user", "root", "--auth", "agent")
	if err == nil || !strings.Contains(err.Error(), "fleetsh edit racknerd") {
		t.Fatalf("duplicate missing edit hint: %v", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("duplicate add changed inventory")
	}
}

func TestAddCreatesDefaultInventory(t *testing.T) {
	home := t.TempDir()
	for _, env := range []string{"APPDATA", "XDG_CONFIG_HOME", "HOME"} {
		t.Setenv(env, home)
	}
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := RunContext(context.Background(), []string{"add", "vps", "--host", "vps.example", "--user", "root", "--no-save-password"}, "test", strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("default add: %d %s", code, &errOut)
	}
	inv, err := inventory.Load(path)
	if err != nil || inv.Hosts["vps"].Auth != "password" {
		t.Fatalf("default inventory not created: %v", err)
	}
}

func TestFailedFirstAddDoesNotCreateFile(t *testing.T) {
	for _, name := range []string{"invalid", "cancel", "mismatch", "store"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "new", "config.toml")
			store := &recordingStore{values: map[string]string{}}
			prompts := 0
			prompt := func(string) (string, error) {
				prompts++
				if name == "cancel" {
					return "", context.Canceled
				}
				if name == "mismatch" && prompts == 2 {
					return "other", nil
				}
				return "fixture", nil
			}
			if name == "store" {
				store.setErr = errors.New("fixture store error")
			}
			args := []string{"add", "vps", "--host", "vps.example", "--user", "root"}
			if name == "invalid" {
				args = append(args, "--port", "99999")
			}
			if _, err := hostCommand(t, path, store, prompt, args...); err == nil {
				t.Fatal("failure accepted")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) || len(store.values) != 0 {
				t.Fatal("failed add created file or stored secret")
			}
			if name == "invalid" && prompts != 0 {
				t.Fatal("invalid host prompted")
			}
		})
	}
}

func TestAddDoesNotReplaceBadInventory(t *testing.T) {
	for _, kind := range []string{"invalid", "directory", "blocked-parent"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "config.toml")
			const contents = "invalid inventory PRIVATE-FIXTURE"
			preserved := path
			switch kind {
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				preserved = filepath.Join(path, "keep.txt")
			case "blocked-parent":
				preserved = filepath.Join(root, "blocked")
				path = filepath.Join(preserved, "config.toml")
			}
			if err := os.WriteFile(preserved, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			store := &recordingStore{values: map[string]string{}}
			_, err := hostCommand(t, path, store, nil, "add", "vps", "--host", "vps.example", "--user", "root", "--no-save-password")
			if err == nil || strings.Contains(err.Error(), "PRIVATE-FIXTURE") {
				t.Fatalf("bad inventory handling: %v", err)
			}
			after, readErr := os.ReadFile(preserved)
			if readErr != nil || string(after) != contents || len(store.values) != 0 {
				t.Fatal("existing file replaced")
			}
		})
	}
}

func TestMissingInventoryHints(t *testing.T) {
	for _, args := range [][]string{{"ls"}, {"show", "vps"}, {"edit", "vps", "--port", "2222"}, {"ssh", "vps"}} {
		t.Run(args[0], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "new", "config.toml")
			var out, errOut bytes.Buffer
			code := RunContext(context.Background(), append([]string{"--config", path}, args...), "test", strings.NewReader(""), &out, &errOut)
			if code != 2 || out.Len() != 0 {
				t.Fatalf("code=%d out=%s", code, &out)
			}
			for _, marker := range []string{"inventory not found", path, "add a host", "fleetsh init", "same --config PATH"} {
				if !strings.Contains(errOut.String(), marker) {
					t.Fatalf("missing %q: %s", marker, &errOut)
				}
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("read/edit command created inventory")
			}
		})
	}
}
