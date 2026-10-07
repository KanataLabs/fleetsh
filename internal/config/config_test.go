// SPDX-License-Identifier: GPL-3.0-only
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInitPreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	if err := Init(path); err != nil {
		t.Fatal(err)
	}
	original := []byte("# user-owned configuration\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Init(path); err == nil {
		t.Fatal("expected refusal to overwrite configuration")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("existing configuration changed: %q, %v", got, err)
	}
}

func TestInitWritesPrivateExample(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fleetsh", "config.toml")
	if err := Init(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, Example) {
		t.Fatalf("configuration mismatch: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("configuration is accessible to others: %o", info.Mode().Perm())
		}
	}
}
