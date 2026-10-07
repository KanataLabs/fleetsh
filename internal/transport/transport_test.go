// SPDX-License-Identifier: GPL-3.0-only
package transport_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestKeyAuthentication(t *testing.T) {
	for _, pass := range []string{"", "encrypted-fixture"} {
		t.Run(pass, func(t *testing.T) {
			server := testutil.StartSSH(t, nil)
			h := server.Host()
			h.Auth = "key"
			h.Key = testutil.KeyFile(t, server, pass)
			m := testutil.Manager(t, map[string]inventory.Host{"a": h}, server)
			if pass != "" {
				m.Store.(testutil.Store)["login"] = pass
			}
			if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
				t.Fatal(err)
			}
			r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
			if !r.Success {
				t.Fatalf("key authentication failed: %+v", r)
			}
		})
	}
}
func TestProxyJumpAndSOCKS(t *testing.T) {
	target := testutil.StartSSH(t, nil)
	jump := testutil.StartSSH(t, nil)
	for _, mode := range []string{"jump", "socks", "authenticated-socks"} {
		t.Run(mode, func(t *testing.T) {
			h := target.Host()
			hosts := map[string]inventory.Host{"a": h}
			switch mode {
			case "jump":
				h.ProxyJump = "jump"
				hosts["jump"] = jump.Host()
			default:
				h.Proxy = testutil.StartSOCKS(t, mode == "authenticated-socks")
				if mode == "authenticated-socks" {
					h.ProxyCredential = "proxy"
				}
			}
			hosts["a"] = h
			m := testutil.Manager(t, hosts, target, jump)
			if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
				t.Fatal(err)
			}
			r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
			if !r.Success {
				t.Fatalf("%s failed: %+v", mode, r)
			}
		})
	}
}
func TestHostKeyTrustChangedKeyAndReset(t *testing.T) {
	server := testutil.StartSSH(t, nil)
	m := testutil.Manager(t, map[string]inventory.Host{"a": server.Host()})
	if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if r.Success || r.ErrorKind != "hostkey" {
		t.Fatalf("unknown key accepted: %+v", r)
	}
	called := 0
	m.Keys.Trust = func(ctx context.Context, addr string, key ssh.PublicKey) bool { called++; return true }
	if r = executor.Run(context.Background(), m, "a", "uptime", executor.Options{}); !r.Success || called != 1 {
		t.Fatalf("explicit trust failed: %+v", r)
	}
	records, err := m.Keys.Lookup(server.Address)
	if err != nil || len(records) != 1 {
		t.Fatalf("cannot inspect trusted key: %v %v", records, err)
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	other, _ := ssh.NewPublicKey(pub)
	if err = os.WriteFile(m.Keys.Path, []byte(knownhosts.Line([]string{knownhosts.Normalize(server.Address)}, other)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r = executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if r.Success || r.ErrorKind != "hostkey" || called != 1 {
		t.Fatalf("changed key was accepted or prompted: %+v", r)
	}
	if err = m.Keys.Reset(context.Background(), server.Address); err != nil {
		t.Fatal(err)
	}
	records, err = m.Keys.Lookup(server.Address)
	if err != nil || len(records) != 0 {
		t.Fatal("reset retained trust")
	}
}
func TestJumpKeyIsVerified(t *testing.T) {
	target := testutil.StartSSH(t, nil)
	jump := testutil.StartSSH(t, nil)
	h := target.Host()
	h.ProxyJump = "jump"
	m := testutil.Manager(t, map[string]inventory.Host{"a": h, "jump": jump.Host()}, target)
	if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if r.Success || r.ErrorKind != "hostkey" {
		t.Fatalf("unknown jump key accepted: %+v", r)
	}
}
func TestAuthenticationFailureAndHandshakeTimeout(t *testing.T) {
	server := testutil.StartSSH(t, nil)
	m := testutil.Manager(t, map[string]inventory.Host{"a": server.Host()}, server)
	m.Store.(testutil.Store)["login"] = "wrong-password"
	if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if r.Success || r.ErrorKind != "authentication" || strings.Contains(r.Error, "wrong-password") {
		t.Fatalf("bad authentication handling: %+v", r)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_, _ = io.Copy(io.Discard, conn)
			conn.Close()
		}
		close(done)
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	h := server.Host()
	h.Host = host
	fmt.Sscan(port, &h.Port)
	m.Inventory.Hosts["a"] = h
	m.Inventory.Defaults.ConnectTimeout = "50ms"
	start := time.Now()
	r = executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if r.Success || time.Since(start) > time.Second {
		t.Fatalf("handshake did not time out: %+v", r)
	}
	<-done
}
