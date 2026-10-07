//go:build !windows

// SPDX-License-Identifier: GPL-3.0-only
package transport_test

import (
	"net"
	"path/filepath"
	"testing"
)

func agentListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("unix", filepath.Join(t.TempDir(), "agent.sock"))
	if err != nil {
		t.Fatal(err)
	}
	return listener
}
