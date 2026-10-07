// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidCommandsReturnUsageError(t *testing.T) {
	for _, args := range [][]string{
		{"exec", "@all"},
		{"init", "--unknown"},
		{"init", "extra"},
		{"version", "extra"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := Run(args, "dev", &out, &errOut); code != 2 || errOut.Len() == 0 {
				t.Fatalf("want usage error; got code=%d stderr=%q", code, errOut.String())
			}
		})
	}
}

func TestInitHonorsExplicitPathAndPreservesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	var out, errOut bytes.Buffer
	args := []string{"init", "--config", path}
	if code := Run(args, "dev", &out, &errOut); code != 0 {
		t.Fatalf("init failed: %d %s", code, &errOut)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if code := Run(args, "dev", &out, &errOut); code != 2 {
		t.Fatalf("repeat init should fail, got %d", code)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("repeat init modified the configuration")
	}
}

func TestVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run([]string{"version"}, "v0.0.0-test", &out, &errOut); code != 0 || out.String() != "fleetsh v0.0.0-test\n" {
		t.Fatalf("unexpected version output: %d %q", code, out.String())
	}
}
