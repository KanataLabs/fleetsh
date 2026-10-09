// SPDX-License-Identifier: GPL-3.0-only
package transport_test

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"golang.org/x/crypto/ssh/agent"
)

func TestAgentAuthentication(t *testing.T) {
	server := testutil.StartSSH(t, nil)
	listener := agentListener(t)
	t.Setenv("SSH_AUTH_SOCK", listener.Addr().String())
	ring := agent.NewKeyring()
	if err := ring.Add(agent.AddedKey{PrivateKey: server.Private}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	connections := map[net.Conn]bool{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			connections[conn] = true
			mu.Unlock()
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				defer func() { mu.Lock(); delete(connections, conn); mu.Unlock() }()
				_ = agent.ServeAgent(ring, conn)
			}()
		}
	}()
	t.Cleanup(func() {
		listener.Close()
		mu.Lock()
		for c := range connections {
			c.Close()
		}
		mu.Unlock()
		wg.Wait()
	})
	h := server.Host()
	h.Auth = "agent"
	h.Credential = ""
	m := testutil.Manager(t, map[string]inventory.Host{"a": h}, server)
	if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if !r.Success {
		t.Fatalf("agent authentication failed: %+v", r)
	}
}

func TestAgentPreparationErrorsExplainAuthenticationAndRecovery(t *testing.T) {
	for _, scenario := range []struct{ name, message string }{
		{"unavailable", "SSH agent unavailable"},
		{"empty", "SSH agent has no usable keys"},
		{"broken", "cannot read keys from SSH agent"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			listener := agentListener(t)
			t.Setenv("SSH_AUTH_SOCK", listener.Addr().String())
			if scenario.name == "unavailable" {
				listener.Close()
			} else {
				done := make(chan struct{})
				go func() {
					defer close(done)
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					defer conn.Close()
					if scenario.name == "empty" {
						_ = agent.ServeAgent(agent.NewKeyring(), conn)
					}
				}()
				t.Cleanup(func() { listener.Close(); <-done })
			}
			m := testutil.Manager(t, map[string]inventory.Host{"racknerd": {Host: "vps.example", User: "root", Auth: "agent"}})
			m.Prompt = func(string) (string, error) { t.Fatal("agent mode unexpectedly requested a password"); return "", nil }
			err := m.Prepare(context.Background(), []string{"racknerd"}, false)
			if err == nil {
				t.Fatal("agent failure accepted")
			}
			for _, text := range []string{scenario.message, "auth=agent", "fleetsh edit racknerd --auth password", "fleetsh edit racknerd --auth key --key PATH", "start SSH agent and load a key"} {
				if !strings.Contains(err.Error(), text) {
					t.Fatalf("missing recovery text %q: %v", text, err)
				}
			}
			if m.Inventory.Hosts["racknerd"].Auth != "agent" {
				t.Fatal("agent failure silently changed saved mode")
			}
		})
	}
}
