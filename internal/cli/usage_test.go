// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyntaxErrorsShowRelevantHelpWithoutLoadingInventory(t *testing.T) {
	for _, test := range []struct {
		name              string
		args              []string
		usage, diagnostic string
	}{
		{"exec empty", []string{"exec"}, "fleetsh exec SELECTOR COMMAND", "exec requires"},
		{"exec command only", []string{"exec", "whoami"}, "fleetsh exec SELECTOR COMMAND", "exec requires"},
		{"exec selector only", []string{"exec", "@all"}, "fleetsh exec SELECTOR COMMAND", "exec requires"},
		{"exec unquoted command", []string{"exec", "@all", "echo", "hello"}, "fleetsh exec SELECTOR COMMAND", "exec requires"},
		{"exec unknown flag", []string{"exec", "@all", "whoami", "--unknown"}, "fleetsh exec SELECTOR COMMAND", "unknown flag"},
		{"exec invalid duration", []string{"exec", "@all", "whoami", "--timeout", "invalid"}, "fleetsh exec SELECTOR COMMAND", "invalid argument"},
		{"ssh missing host", []string{"ssh"}, "fleetsh ssh HOST", "received 0"},
		{"nested credential missing reference", []string{"credential", "add"}, "fleetsh credential add REFERENCE", "received 0"},
		{"init extra", []string{"init", "extra"}, "fleetsh init", "unknown command"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing.toml")
			args := append([]string{"--config", path, "--json"}, test.args...)
			var out, errOut bytes.Buffer
			code := RunContext(context.Background(), args, "test", strings.NewReader(""), &out, &errOut)
			if code != 2 || out.Len() != 0 {
				t.Fatalf("syntax error: code=%d stdout=%q stderr=%q", code, &out, &errOut)
			}
			for _, want := range []string{test.diagnostic, "Usage:", test.usage, "Examples:", "Flags:", "Documentation: " + documentationURL} {
				if !strings.Contains(errOut.String(), want) {
					t.Fatalf("missing %q in stderr: %s", want, &errOut)
				}
			}
			if test.args[0] == "exec" && !strings.Contains(errOut.String(), "fleetsh exec '@all' \"whoami\"") {
				t.Fatalf("missing actionable all-host example: %s", &errOut)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("syntax error touched inventory: %v", err)
			}
			if strings.Contains(errOut.String(), "missing.toml") {
				t.Fatalf("syntax error attempted inventory loading: %s", &errOut)
			}
		})
	}
}

func TestRuntimeErrorsDoNotPrintCommandHelp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	var out, errOut bytes.Buffer
	code := RunContext(context.Background(), []string{"--config", path, "exec", "@all", "whoami"},
		"test", strings.NewReader(""), &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "missing.toml") {
		t.Fatalf("want inventory error, got code=%d stderr=%q", code, &errOut)
	}
	if strings.Contains(errOut.String(), "Usage:") || strings.Contains(errOut.String(), "Examples:") {
		t.Fatalf("runtime error printed syntax help: %s", &errOut)
	}
}

func TestOfflineHelpUsesDirectHTTPSDocumentation(t *testing.T) {
	for _, args := range [][]string{{"exec", "--help"}, {"docs"}, {"docs", "quickstart"}, {"docs", "passwords"}} {
		var out, errOut bytes.Buffer
		code := RunContext(context.Background(), append([]string{"--config", filepath.Join(t.TempDir(), "missing.toml")}, args...),
			"test", strings.NewReader(""), &out, &errOut)
		if code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "https://kanatalabs.com/fleetsh/en/") {
			t.Fatalf("help %v: code=%d stdout=%q stderr=%q", args, code, &out, &errOut)
		}
		if strings.Contains(out.String(), "http://") || strings.Contains(out.String(), "kanatalabs.github.io") {
			t.Fatalf("help contains redirecting documentation URL: %s", &out)
		}
		if len(args) == 1 {
			for _, lang := range []string{"ja", "zh"} {
				if !strings.Contains(out.String(), documentationSiteURL+lang+"/") {
					t.Fatalf("missing %s edition: %s", lang, &out)
				}
			}
		}
	}
}
