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
	"sync"
	"testing"
	"time"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/spf13/cobra"
)

func TestChangedHostOptionsIgnoreGlobalAndHelpFlags(t *testing.T) {
	for _, test := range []struct {
		name, flag, value string
		want              bool
	}{
		{"none", "", "", false}, {"configuration", "config", "custom.toml", false},
		{"json", "json", "true", false}, {"help disabled", "help", "false", false},
		{"help enabled", "help", "true", false}, {"empty groups", "groups", "", true},
		{"explicit default port", "port", "22", true}, {"description", "description", "new", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := &cobra.Command{Use: "fleetsh"}
			root.PersistentFlags().String("config", "", "")
			root.PersistentFlags().Bool("json", false, "")
			cmd := &cobra.Command{Use: "edit HOST"}
			cmd.Flags().Bool("help", false, "")
			cmd.Flags().String("groups", "", "")
			cmd.Flags().Int("port", 22, "")
			cmd.Flags().String("description", "", "")
			root.AddCommand(cmd)
			if test.flag != "" {
				flags := cmd.Flags()
				if test.flag == "config" || test.flag == "json" {
					flags = root.PersistentFlags()
				}
				if err := flags.Set(test.flag, test.value); err != nil {
					t.Fatal(err)
				}
			}
			if got := changedHostOptions(cmd); got != test.want {
				t.Fatalf("changed=%v want=%v", got, test.want)
			}
		})
	}
}
func TestLineLeavesNextPromptInputUnread(t *testing.T) {
	input := strings.NewReader("  hk1.example  \r\nprivate-next-input")
	var errOut bytes.Buffer
	app := &application{ctx: t.Context(), in: input, errOut: &errOut}
	value, err := app.line("Address")
	if err != nil || value != "hk1.example" {
		t.Fatalf("value=%q error=%v", value, err)
	}
	rest, err := io.ReadAll(input)
	if err != nil || string(rest) != "private-next-input" {
		t.Fatalf("read ahead into next prompt: %q %v", rest, err)
	}
	if errOut.String() != "Address: " {
		t.Fatalf("prompt=%q", errOut.String())
	}
}

type signaledInput struct {
	reader  io.Reader
	started chan struct{}
	once    sync.Once
}

func (r *signaledInput) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.reader.Read(p)
}
func TestLineCancellationAndEOF(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	input := &signaledInput{reader: reader, started: make(chan struct{})}
	app := &application{ctx: ctx, in: input, errOut: io.Discard}
	done := make(chan error, 1)
	go func() { _, err := app.line("Address"); done <- err }()
	<-input.started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("prompt ignored cancellation")
	}
	app = &application{ctx: t.Context(), in: strings.NewReader(""), errOut: io.Discard}
	if _, err := app.line("Address"); err == nil || !strings.Contains(err.Error(), "No host settings were saved") {
		t.Fatalf("EOF: %v", err)
	}
}
func TestReadableFileErrorsRetainHostContext(t *testing.T) {
	for _, test := range []struct {
		name  string
		cause error
		want  string
	}{
		{"missing", os.ErrNotExist, "does not exist"}, {"permission", os.ErrPermission, "Access was denied"},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := fmt.Errorf("host hk1: %w", &os.PathError{Op: "GetFileAttributesEx", Path: "key.pem", Err: test.cause})
			text := readableError(raw)
			for _, want := range []string{"host hk1", "key.pem", test.want} {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q: %s", want, text)
				}
			}
			if strings.Contains(text, "GetFileAttributesEx") {
				t.Fatal("OS implementation leaked into user error")
			}
		})
	}
	missing := missingInventoryError{path: "inventory.toml", cause: &os.PathError{Op: "open", Path: "inventory.toml", Err: os.ErrNotExist}}
	text := readableError(missing)
	if !strings.Contains(text, "Next:") || !strings.Contains(text, "same --config PATH") {
		t.Fatal("lost missing-inventory recovery")
	}
}
func TestReadablePreparationErrorsGiveNextAction(t *testing.T) {
	for _, message := range []string{"passwords do not match", "system credential store unavailable or locked", "credential not found", "secret input requires an interactive terminal", "inventory lock timed out"} {
		text := readableError(errors.New(message))
		if !strings.Contains(text, message) || !strings.Contains(text, "Next:") {
			t.Fatalf("missing cause/action: %s", text)
		}
	}
	original := errors.New("custom fixture failure")
	if readableError(original) != original.Error() {
		t.Fatal("unknown failure was misclassified")
	}
}
func TestDownloadDirectoriesRemainDistinctAndPortable(t *testing.T) {
	aliases := []string{"hk1", "HK1", "hk1.", "CON", "NUL", "LPT1"}
	seen := map[string]bool{}
	for _, alias := range aliases {
		name := downloadDirectory(alias)
		folded := strings.ToLower(name)
		if seen[folded] || !filepath.IsLocal(name) || strings.ContainsAny(name, "/\\") {
			t.Fatalf("alias %q produced unsafe/colliding directory %q", alias, name)
		}
		seen[folded] = true
		if name != downloadDirectory(alias) {
			t.Fatal("directory changed between calls")
		}
	}
}
func TestStatsTableShowsMissingDataAndFailedHosts(t *testing.T) {
	var out, errOut bytes.Buffer
	app := &application{out: &out, errOut: &errOut}
	rows := []executor.Result{
		{Host: "good", Success: true, Stdout: "USER: ubuntu\nCPU: unavailable\nMEMORY: 100 / 200 MiB\nROOT_FREE: 4G\n"},
		{Host: "offline", ErrorKind: "connection_timeout"},
		{Host: "console", Skipped: true},
	}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	if err := app.statsTable(rows, now); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"2026-10-14T12:00:00Z", "good", "ubuntu", "4G", "unavailable", "offline", "failed", "console", "skipped"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing %q: %s", text, &out)
		}
	}
	if !strings.Contains(errOut.String(), "timed out") || !strings.Contains(errOut.String(), "doctor offline") {
		t.Fatal("failed host lacks recovery guidance")
	}
}
func TestHostCompletionKeepsSelectorPrefix(t *testing.T) {
	path := emptyInventory(t)
	if _, err := hostCommand(t, path, nil, nil, "add", "hk1", "--host", "hk1.example", "--user", "root", "--auth", "agent", "--groups", "web"); err != nil {
		t.Fatal(err)
	}
	if _, err := hostCommand(t, path, nil, nil, "add", "sg1", "--host", "sg1.example", "--user", "root", "--auth", "agent"); err != nil {
		t.Fatal(err)
	}
	app := &application{ctx: t.Context(), path: path}
	values, directive := app.completeHost(true)(&cobra.Command{}, nil, "hk1,@w")
	if !slices.Equal(values, []string{"hk1,@web"}) || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion=%v directive=%v", values, directive)
	}
	values, _ = app.completeHost(false)(&cobra.Command{}, nil, "@")
	if len(values) != 0 {
		t.Fatal("single-host command offered a group")
	}
	app.path = filepath.Join(t.TempDir(), "missing.toml")
	values, directive = app.completeHost(true)(&cobra.Command{}, nil, "")
	if len(values) != 0 || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatal("missing inventory completion was not silent")
	}
}
