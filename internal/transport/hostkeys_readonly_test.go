// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestReadOnlyHostKeysNeverTrustUnknownKey(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	for _, exists := range []bool{false, true} {
		name := "missing file"
		if exists {
			name = "existing empty file"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "known_hosts")
			if exists {
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			keys := &HostKeys{Path: path, ReadOnly: true, Trust: func(context.Context, string, ssh.PublicKey) bool { calls++; return true }}
			err := keys.Callback(t.Context())("hk1.example:22", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 22}, signer.PublicKey())
			if err == nil || !strings.Contains(err.Error(), "unknown host key") || calls != 0 {
				t.Fatalf("error=%v trust calls=%d", err, calls)
			}
			data, readErr := os.ReadFile(path)
			if exists {
				if readErr != nil || len(data) != 0 {
					t.Fatal("read-only verification wrote a key")
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatal("read-only verification created a file")
			}
		})
	}
}
