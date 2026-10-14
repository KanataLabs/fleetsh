// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/actions"
	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"github.com/pelletier/go-toml/v2"
)

func operationFixture(t *testing.T, hosts map[string]inventory.Host, servers ...*testutil.Server) string {
	t.Helper()
	m := testutil.Manager(t, hosts, servers...)
	path := filepath.Join(filepath.Dir(m.Keys.Path), "config.toml")
	data, err := toml.Marshal(m.Inventory)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func operation(t *testing.T, path string, store credentials.Store, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	app := &application{ctx: t.Context(), in: strings.NewReader(""), out: &out, errOut: &errOut, store: store}
	root := app.root("test")
	root.SetArgs(append([]string{"--config", path}, args...))
	err := root.ExecuteContext(t.Context())
	return out.String(), errOut.String(), err
}
func TestInteractiveSetupAndEdit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.toml")
	store := &recordingStore{values: map[string]string{}}
	var out, errOut bytes.Buffer
	answers := []string{"vps.example", "ubuntu"}
	app := &application{ctx: t.Context(), in: strings.NewReader(""), out: &out, errOut: &errOut, store: store, readSecret: passwordPrompt("fixture-password"),
		readLine: func(label string) (string, error) {
			if len(answers) == 0 {
				t.Fatalf("unexpected prompt: %s", label)
			}
			value := answers[0]
			answers = answers[1:]
			return value, nil
		}}
	run := func(args ...string) error {
		root := app.root("test")
		root.SetArgs(append([]string{"--config", path}, args...))
		return root.ExecuteContext(t.Context())
	}
	if err := run("add", "hk1"); err != nil {
		t.Fatal(err)
	}
	inv, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Hosts["hk1"].User != "ubuntu" || len(store.values) != 1 || !strings.Contains(out.String(), "Added host hk1") {
		t.Fatal("guided add failed")
	}
	answers = []string{"3"}
	app.readSecret = passwordPrompt("corrected-password")
	if err := run("edit", "hk1"); err != nil {
		t.Fatal(err)
	}
	inv, err = inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.values[inv.Hosts["hk1"].Credential] != "corrected-password" || !strings.Contains(out.String(), "Updated host hk1") {
		t.Fatal("menu password replacement failed")
	}
	answers = []string{"4", "web,asia"}
	if err := run("edit", "hk1"); err != nil {
		t.Fatal(err)
	}
	inv, _ = inventory.Load(path)
	if len(inv.Hosts["hk1"].Groups) != 2 {
		t.Fatal("menu groups not saved")
	}
	before, _ := os.ReadFile(path)
	answers = []string{"q"}
	if err := run("edit", "hk1"); err == nil {
		t.Fatal("cancel accepted as save")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("cancel changed inventory")
	}
	app.readLine = func(string) (string, error) { t.Fatal("JSON opened menu"); return "", nil }
	if err := run("edit", "hk1", "--json"); err == nil {
		t.Fatal("JSON bare edit accepted")
	}
}
func TestDoctorReportsLoginAndDoesNotTrustKeys(t *testing.T) {
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		return testutil.Reply{Out: "alive\n"}
	})
	path := operationFixture(t, map[string]inventory.Host{"hk1": server.Host()}, server)
	for _, wrong := range []bool{false, true} {
		secret := testutil.Password
		if wrong {
			secret = "incorrect-fixture-password"
		}
		out, _, err := operation(t, path, testutil.Store{"login": secret}, "doctor", "hk1", "--json")
		var report diagnosticReport
		if json.Unmarshal([]byte(out), &report) != nil {
			t.Fatal(out)
		}
		if report.Success == wrong || (err != nil) != wrong {
			t.Fatalf("wrong=%v err=%v report=%+v", wrong, err, report)
		}
		if wrong && !strings.Contains(out, "--save-password") {
			t.Fatal("missing password recovery")
		}
		if strings.Contains(out, secret) {
			t.Fatal("secret leaked")
		}
	}
	known := filepath.Join(filepath.Dir(path), "known_hosts")
	if err := os.Remove(known); err != nil {
		t.Fatal(err)
	}
	out, _, err := operation(t, path, testutil.Store{"login": testutil.Password}, "doctor", "hk1", "--json")
	if err == nil || !strings.Contains(out, "host key") {
		t.Fatalf("unknown key: %v %s", err, out)
	}
	if _, err := os.Stat(known); !os.IsNotExist(err) {
		t.Fatal("doctor created trust file")
	}
}
func TestStatsTableKeepsOtherHostsWhenCredentialMissing(t *testing.T) {
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		if command != actions.Stats {
			t.Error(command)
		}
		return testutil.Reply{Out: "USER: fixture\nCPU: 3.0%\nMEMORY: 1 / 2 MiB\nSWAP: 0 / 0 MiB\nROOT_FREE: 14G\n"}
	})
	missing := server.Host()
	missing.Credential = "missing"
	path := operationFixture(t, map[string]inventory.Host{"good": server.Host(), "bad": missing}, server)
	out, errOut, err := operation(t, path, testutil.Store{"login": testutil.Password}, "stats", "--table")
	var code exitError
	if !errors.As(err, &code) || code.code != 2 {
		t.Fatalf("preparation exit code: %v", err)
	}
	for _, text := range []string{"Snapshot:", "good", "online", "bad", "failed", "14G"} {
		if !strings.Contains(out, text) {
			t.Fatalf("missing %s: %s", text, out)
		}
	}
	if !strings.Contains(errOut, "bad") {
		t.Fatal("failed-host detail missing")
	}
}

type watchOutput struct {
	bytes.Buffer
	cancel context.CancelFunc
	frames int
}

func (w *watchOutput) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	w.frames++
	if w.frames == 2 {
		w.cancel()
	}
	return n, err
}
func TestWatchJSONSnapshotsAndCancellation(t *testing.T) {
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		return testutil.Reply{Out: "USER: fixture\nCPU: 1%\n"}
	})
	path := operationFixture(t, map[string]inventory.Host{"hk1": server.Host()}, server)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := &watchOutput{cancel: cancel}
	var errOut bytes.Buffer
	app := &application{ctx: ctx, in: strings.NewReader(""), out: out, errOut: &errOut, store: testutil.Store{"login": testutil.Password}}
	root := app.root("test")
	root.SetArgs([]string{"--config", path, "stats", "--watch", "1s", "--json"})
	if err := root.ExecuteContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("watch cancellation: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(out.Bytes()))
	for range 2 {
		var report struct {
			Timestamp string `json:"timestamp"`
			executor.Report
		}
		if err := decoder.Decode(&report); err != nil {
			t.Fatal(err)
		}
		if report.Timestamp == "" || report.Summary.Success != 1 {
			t.Fatal("missing snapshot fields")
		}
	}
	if decoder.Decode(&map[string]any{}) != io.EOF {
		t.Fatal("unexpected extra snapshot")
	}
}
func TestSavedCommandAndUpdateRestartCheck(t *testing.T) {
	var commandCount, updateCount, checkCount atomic.Int32
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		switch {
		case command == "df -h":
			commandCount.Add(1)
			return testutil.Reply{Out: "fixture disk\n"}
		case command == actions.DetectOS:
			return testutil.Reply{Out: "ubuntu\n"}
		case command == actions.RebootRequired:
			checkCount.Add(1)
			return testutil.Reply{Out: "Restart required: yes.\n"}
		case strings.Contains(command, "apt-get"):
			updateCount.Add(1)
			return testutil.Reply{Out: "updated\n"}
		}
		return testutil.Reply{Code: 1}
	})
	path := operationFixture(t, map[string]inventory.Host{"hk1": server.Host()}, server)
	if err := inventory.Update(t.Context(), path, func(inv *inventory.Inventory) error {
		inv.Commands = map[string]inventory.Command{"disk": {Command: "df -h", Timeout: "30s"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := operation(t, path, testutil.Store{}, "run", "hk1", "disk", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if commandCount.Load() != 0 {
		t.Fatal("dry run executed command")
	}
	out, _, err := operation(t, path, testutil.Store{"login": testutil.Password}, "run", "hk1", "disk")
	if err != nil || !strings.Contains(out, "fixture disk") || commandCount.Load() != 1 {
		t.Fatalf("run: %v %s", err, out)
	}
	out, _, err = operation(t, path, testutil.Store{"login": testutil.Password}, "update", "hk1", "--yes", "--check-reboot")
	if err != nil || updateCount.Load() != 1 || checkCount.Load() != 1 || !strings.Contains(out, "Restart required: yes") {
		t.Fatalf("update check: %v %s", err, out)
	}
}
func TestRollingRebootStopsBeforeNextHost(t *testing.T) {
	var laterCalls atomic.Int32
	first := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		if command == "reboot" {
			return testutil.Reply{Code: 1, Err: "fixture reboot refused"}
		}
		return testutil.Reply{Out: "boot-old\n"}
	})
	later := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		laterCalls.Add(1)
		return testutil.Reply{Out: "boot-new\n"}
	})
	path := operationFixture(t, map[string]inventory.Host{"a": first.Host(), "b": later.Host()}, first, later)
	out, _, err := operation(t, path, testutil.Store{"login": testutil.Password}, "reboot", "@all", "--batch-size", "1", "--yes", "--json")
	var report executor.Report
	if json.Unmarshal([]byte(out), &report) != nil {
		t.Fatal(out)
	}
	if err == nil || laterCalls.Load() != 0 || len(report.Results) != 2 || !report.Results[1].Skipped || !strings.Contains(report.Results[1].Error, "previous batch") {
		t.Fatalf("rolling result: %v %s", err, out)
	}
}
func TestSFTPTransfersPreserveExistingFilesAndSeparateHosts(t *testing.T) {
	one := testutil.StartSSH(t, nil)
	two := testutil.StartSSH(t, nil)
	path := operationFixture(t, map[string]inventory.Host{"one": one.Host(), "two": two.Host()}, one, two)
	source := filepath.Join(t.TempDir(), "source.txt")
	payload := []byte("transfer-fixture-payload")
	if err := os.WriteFile(source, payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := testutil.Store{"login": testutil.Password}
	if out, _, err := operation(t, path, store, "upload", "@all", source, "/report.txt"); err != nil {
		t.Fatalf("upload: %v %s", err, out)
	}
	if _, _, err := operation(t, path, store, "upload", "one", source, "/report.txt"); err == nil {
		t.Fatal("remote overwrite accepted")
	}
	destination := filepath.Join(t.TempDir(), "downloads")
	if _, _, err := operation(t, path, testutil.Store{}, "download", "@all", "/report.txt", destination, "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("dry run created directories")
	}
	if out, _, err := operation(t, path, store, "download", "@all", "/report.txt", destination); err != nil {
		t.Fatalf("download: %v %s", err, out)
	}
	for _, id := range []string{"one", "two"} {
		local := filepath.Join(destination, downloadDirectory(id), "report.txt")
		data, err := os.ReadFile(local)
		if err != nil || !bytes.Equal(data, payload) {
			t.Fatalf("download %s: %v", id, err)
		}
	}
	local := filepath.Join(destination, "single.txt")
	if err := os.WriteFile(local, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := operation(t, path, store, "download", "one", "/report.txt", local); err == nil {
		t.Fatal("local overwrite accepted")
	}
	data, _ := os.ReadFile(local)
	if string(data) != "original" {
		t.Fatal("local original changed")
	}
	if _, _, err := operation(t, path, store, "download", "one", "/report.txt", local, "--overwrite"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(local)
	if !bytes.Equal(data, payload) {
		t.Fatal("explicit overwrite failed")
	}
}
func TestImportExportAndCompletion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	sshPath := filepath.Join(t.TempDir(), "ssh-config")
	config := "Host hk1\n HostName hk1.example\n User ubuntu\n IdentityFile \"~/.ssh/key with spaces\"\nHost *\n Port 2222\n"
	if err := os.WriteFile(sshPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := operation(t, path, testutil.Store{}, "import", sshPath, "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("import preview created config")
	}
	if _, _, err := operation(t, path, testutil.Store{}, "import", sshPath); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if _, _, err := operation(t, path, testutil.Store{}, "import", sshPath); err == nil {
		t.Fatal("duplicate import accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("duplicate import changed inventory")
	}
	if err := inventory.Update(t.Context(), path, func(inv *inventory.Inventory) error {
		h := inv.Hosts["hk1"]
		h.Credential = "private-reference"
		h.Groups = []string{"web"}
		inv.Hosts["hk1"] = h
		inv.Commands = map[string]inventory.Command{"disk": {Command: "df -h"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"toml", "json", "ssh-config"} {
		out, _, err := operation(t, path, testutil.Store{}, "export", "--format", format)
		if err != nil || strings.Contains(out, "private-reference") {
			t.Fatalf("export %s: %v %s", format, err, out)
		}
	}
	out, _, err := operation(t, path, testutil.Store{}, "__complete", "ssh", "hk")
	if err != nil || !strings.Contains(out, "hk1") {
		t.Fatalf("host completion: %v %s", err, out)
	}
	out, _, err = operation(t, path, testutil.Store{}, "__complete", "stats", "@w")
	if err != nil || !strings.Contains(out, "@web") {
		t.Fatalf("group completion: %v %s", err, out)
	}
	out, _, err = operation(t, path, testutil.Store{}, "__complete", "run", "@all", "di")
	if err != nil || !strings.Contains(out, "disk") {
		t.Fatalf("command completion: %v %s", err, out)
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		out, _, err := operation(t, path, testutil.Store{}, "completion", shell)
		if err != nil || !strings.Contains(out, "fleetsh") {
			t.Fatal(shell, err)
		}
	}
}
func TestHumanReadableErrors(t *testing.T) {
	for kind, phrase := range map[string]string{"authentication": "did not accept", "connection_timeout": "timed out", "hostkey": "host key", "proxy": "proxy", "command": "remote command"} {
		message := remoteFailure("hk1", kind, "private-secret-value")
		if !strings.Contains(message, phrase) || !strings.Contains(message, "Next:") || strings.Contains(message, "private-secret-value") {
			t.Fatalf("%s: %s", kind, message)
		}
	}
	var out, errOut bytes.Buffer
	if code := RunContext(t.Context(), []string{"ssh"}, "test", strings.NewReader(""), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "requires HOST") || strings.Contains(errOut.String(), "accepts 1 arg(s)") {
		t.Fatalf("required args: %d %s", code, &errOut)
	}
	if got := readableError(&os.PathError{Op: "GetFileAttributesEx", Path: "fixture.toml", Err: os.ErrNotExist}); strings.Contains(got, "GetFileAttributesEx") || !strings.Contains(got, "does not exist") {
		t.Fatal(got)
	}
}
