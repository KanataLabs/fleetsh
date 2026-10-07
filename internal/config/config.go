// SPDX-License-Identifier: GPL-3.0-only
// Package config manages the initial inventory file. It never stores secrets.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed example.toml
var Example []byte

// DefaultPath uses the operating system's user configuration directory.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find configuration directory: %w", err)
	}
	return filepath.Join(dir, "fleetsh", "config.toml"), nil
}

// Init creates a private example configuration and never overwrites any file.
func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create configuration (existing files are preserved): %w", err)
	}
	if _, err := f.Write(Example); err != nil {
		_ = f.Close()
		return fmt.Errorf("write configuration: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close configuration: %w", err)
	}
	return nil
}
