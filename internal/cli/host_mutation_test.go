// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/config"
	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"golang.org/x/crypto/ssh/knownhosts"
)

type recordingStore struct {
	values map[string]string
	setErr error
	onSet  func()
}

func (s *recordingStore) Get(ref string) (string, error) {
	if value, ok := s.values[ref]; ok {
		return value, nil
	}
	return "", credentials.ErrNotFound
}
func (s *recordingStore) Set(ref, value string) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.values[ref] = value
	if s.onSet != nil {
		s.onSet()
	}
	return nil
}
func (s *recordingStore) Delete(ref string) error { delete(s.values, ref); return nil }

func hostCommand(t *testing.T, path string, store credentials.Store, prompt func(string) (string, error), args ...string) (string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	app := &application{ctx: context.Background(), in: strings.NewReader(""), out: &out, errOut: &errOut, store: store, path: path, readSecret: prompt}
	root := app.root("test")
	root.SetArgs(append([]string{"--config", path}, args...))
	err := root.Execute()
	return out.String() + errOut.String(), err
}
func emptyInventory(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Init(path); err != nil {
		t.Fatal(err)
	}
	return path
}
func passwordPrompt(value string) func(string) (string, error) {
	return func(string) (string, error) { return value, nil }
}

func TestAddPasswordHostSavesAndAssociatesWithoutCredentialCommand(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{}}
	const secret = "fixture-password-not-in-inventory"
	prompts := 0
	out, err := hostCommand(t, path, store, func(string) (string, error) { prompts++; return secret, nil },
		"add", "vps", "--host", "vps.example", "--user", "ubuntu", "--auth", "password", "--groups", "web,asia,web", "--json")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	host := inv.Hosts["vps"]
	if prompts != 2 || !strings.HasPrefix(host.Credential, credentials.ReferencePrefix) || store.values[host.Credential] != secret || !slices.Contains(inv.Credentials, host.Credential) {
		t.Fatal("password was not prompted, saved and associated")
	}
	if !slices.Equal(host.Groups, []string{"web", "asia"}) {
		t.Fatalf("groups: %v", host.Groups)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), secret) || strings.Contains(out, secret) {
		t.Fatal("secret leaked")
	}
	if !strings.Contains(out, host.Credential) {
		t.Fatal("saved reference absent from JSON")
	}
}
func TestPasswordHostFailureLeavesInventoryAndStoreUnchanged(t *testing.T) {
	for _, name := range []string{"mismatch", "cancel", "store", "invalid", "duplicate", "nonterminal"} {
		t.Run(name, func(t *testing.T) {
			path := emptyInventory(t)
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
			args := []string{"add", "vps", "--host", "vps.example", "--user", "ubuntu", "--auth", "password"}
			if name == "store" {
				store.setErr = errors.New("fixture store unavailable")
			}
			if name == "invalid" {
				args = append(args, "--port", "-1")
			}
			if name == "duplicate" {
				if _, err := hostCommand(t, path, store, nil, "add", "vps", "--host", "existing.example", "--user", "ubuntu"); err != nil {
					t.Fatal(err)
				}
			}
			if name == "nonterminal" {
				prompt = nil
			}
			before, _ := os.ReadFile(path)
			if _, err := hostCommand(t, path, store, prompt, args...); err == nil {
				t.Fatal("failure accepted")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) || len(store.values) != 0 {
				t.Fatal("failed add changed inventory or store")
			}
			if (name == "invalid" || name == "duplicate") && prompts != 0 {
				t.Fatal("prompt occurred before validation")
			}
		})
	}
}
func TestExistingReferenceAndNoSaveAllowNonTerminalCreation(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{"shared": "fixture"}}
	for _, row := range []struct {
		alias string
		args  []string
	}{
		{"existing", []string{"--credential", "shared"}},
		{"prompted", []string{"--no-save-password"}},
	} {
		args := append([]string{"add", row.alias, "--host", "vps.example", "--user", "ubuntu", "--auth", "password"}, row.args...)
		if _, err := hostCommand(t, path, store, nil, args...); err != nil {
			t.Fatal(err)
		}
	}
	inv, _ := inventory.Load(path)
	if inv.Hosts["existing"].Credential != "shared" || inv.Hosts["prompted"].Credential != "" || len(store.values) != 1 {
		t.Fatal("explicit credential selection changed")
	}
}
func TestSavePasswordDoesNotOverwriteSharedCredential(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{"shared": "original"}}
	for _, alias := range []string{"one", "two"} {
		if _, err := hostCommand(t, path, store, nil, "add", alias, "--host", "vps.example", "--user", "ubuntu", "--auth", "password", "--credential", "shared"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hostCommand(t, path, store, passwordPrompt("replacement"), "edit", "one", "--save-password"); err != nil {
		t.Fatal(err)
	}
	inv, _ := inventory.Load(path)
	ref := inv.Hosts["one"].Credential
	if ref == "shared" || store.values[ref] != "replacement" || store.values["shared"] != "original" || inv.Hosts["two"].Credential != "shared" {
		t.Fatal("shared credential was overwritten")
	}
}
func TestSwitchingAuthenticationCreatesPasswordReference(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{"key-passphrase": "original"}}
	if _, err := hostCommand(t, path, store, nil, "add", "vps", "--host", "vps.example", "--user", "ubuntu", "--auth", "key", "--key", "fixture.pem", "--credential", "key-passphrase"); err != nil {
		t.Fatal(err)
	}
	if _, err := hostCommand(t, path, store, passwordPrompt("ssh-password"), "edit", "vps", "--auth", "password"); err != nil {
		t.Fatal(err)
	}
	inv, _ := inventory.Load(path)
	ref := inv.Hosts["vps"].Credential
	if ref == "key-passphrase" || store.values[ref] != "ssh-password" || store.values["key-passphrase"] != "original" {
		t.Fatal("key passphrase reused for password auth")
	}
	if _, err := hostCommand(t, path, store, nil, "edit", "vps", "--no-save-password"); err != nil {
		t.Fatal(err)
	}
	inv, _ = inventory.Load(path)
	if inv.Hosts["vps"].Credential != "" || store.values[ref] != "ssh-password" {
		t.Fatal("no-save deleted an existing secret")
	}
}
func TestInventorySaveFailureRemovesNewCredential(t *testing.T) {
	path := emptyInventory(t)
	before, _ := os.ReadFile(path)
	store := &recordingStore{values: map[string]string{}}
	store.onSet = func() {
		if err := os.Rename(path, path+".original"); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := hostCommand(t, path, store, passwordPrompt("fixture"), "add", "vps", "--host", "vps.example", "--user", "ubuntu", "--auth", "password"); err == nil {
		t.Fatal("inventory failure accepted")
	}
	original, _ := os.ReadFile(path + ".original")
	if len(store.values) != 0 || !bytes.Equal(before, original) {
		t.Fatal("new secret not rolled back or original modified")
	}
}
func TestPasswordPromptRejectsConcurrentHostChange(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{}}
	if _, err := hostCommand(t, path, store, nil, "add", "vps", "--host", "original.example", "--user", "ubuntu"); err != nil {
		t.Fatal(err)
	}
	changed := false
	prompt := func(string) (string, error) {
		if !changed {
			changed = true
			if err := inventory.Update(context.Background(), path, func(inv *inventory.Inventory) error {
				h := inv.Hosts["vps"]
				h.Host = "changed.example"
				inv.Hosts["vps"] = h
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return "fixture", nil
	}
	if _, err := hostCommand(t, path, store, prompt, "edit", "vps", "--auth", "password"); err == nil {
		t.Fatal("concurrent host change ignored")
	}
	inv, _ := inventory.Load(path)
	if len(store.values) != 0 || inv.Hosts["vps"].Host != "changed.example" || inv.Hosts["vps"].Auth != "agent" {
		t.Fatal("concurrent change overwritten")
	}
}
func TestIncrementalGroupsPreserveOtherMembershipsAndSelectors(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{}}
	if _, err := hostCommand(t, path, store, nil, "add", "vps", "--host", "vps.example", "--user", "ubuntu", "--auth", "password", "--no-save-password", "--groups", "asia,web"); err != nil {
		t.Fatal(err)
	}
	if _, err := hostCommand(t, path, store, nil, "edit", "vps", "--add-groups", "production,web", "--remove-groups", "asia"); err != nil {
		t.Fatal(err)
	}
	inv, _ := inventory.Load(path)
	if !slices.Equal(inv.Hosts["vps"].Groups, []string{"web", "production"}) {
		t.Fatalf("groups: %v", inv.Hosts["vps"].Groups)
	}
	for _, group := range []string{"web", "production"} {
		ids, err := inv.Select("@"+group, "")
		if err != nil || !slices.Equal(ids, []string{"vps"}) {
			t.Fatalf("selector %s: %v %v", group, ids, err)
		}
	}
	for _, args := range [][]string{
		{"--groups", "only", "--add-groups", "extra"},
		{"--add-groups", "same", "--remove-groups", "same"},
		{"--remove-groups", "bad group"},
	} {
		before, _ := os.ReadFile(path)
		if _, err := hostCommand(t, path, store, nil, append([]string{"edit", "vps"}, args...)...); err == nil {
			t.Fatal("invalid group edit accepted")
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			t.Fatal("invalid group edit changed inventory")
		}
	}
	if _, err := hostCommand(t, path, store, nil, "edit", "vps", "--groups", ""); err != nil {
		t.Fatal(err)
	}
	inv, _ = inventory.Load(path)
	if len(inv.Hosts["vps"].Groups) != 0 {
		t.Fatal("groups were not cleared")
	}
}

// The stored reference must work end-to-end without another credential command.
func TestAutomaticallySavedPasswordAuthenticatesGroupedExecution(t *testing.T) {
	path := emptyInventory(t)
	store := &recordingStore{values: map[string]string{}}
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		if command != "uptime" {
			t.Errorf("unexpected command: %s", command)
		}
		return testutil.Reply{Out: "fixture uptime\n"}
	})
	host := server.Host()
	if _, err := hostCommand(t, path, store, passwordPrompt(testutil.Password), "add", "vps",
		"--host", host.Host, "--port", fmt.Sprint(host.Port), "--user", host.User, "--auth", "password", "--groups", "lab"); err != nil {
		t.Fatal(err)
	}
	trust := knownhosts.Line([]string{knownhosts.Normalize(server.Address)}, server.Key.PublicKey()) + "\n"
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "known_hosts"), []byte(trust), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := hostCommand(t, path, store, nil, "exec", "@lab", "uptime", "--json")
	if err != nil || !strings.Contains(out, "fixture uptime") {
		t.Fatalf("saved credential did not authenticate: %v %s", err, out)
	}
}
