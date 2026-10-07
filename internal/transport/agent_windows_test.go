//go:build windows

// SPDX-License-Identifier: GPL-3.0-only
package transport_test

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"testing"

	"github.com/Microsoft/go-winio"
)

func agentListener(t *testing.T) net.Listener {
	t.Helper()
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	listener, err := winio.ListenPipe(`\\.\pipe\fleetsh-test-`+hex.EncodeToString(id), nil)
	if err != nil {
		t.Fatal(err)
	}
	return listener
}
