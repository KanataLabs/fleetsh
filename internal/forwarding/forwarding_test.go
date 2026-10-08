// SPDX-License-Identifier: GPL-3.0-only
package forwarding_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/KanataLabs/fleetsh/internal/forwarding"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
	"golang.org/x/net/proxy"
)

func echoServer(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer conn.Close(); _, _ = io.Copy(conn, conn) }()
		}
	}()
	return ln
}
func TestLocalRemoteAndDynamicTraffic(t *testing.T) {
	for _, kind := range []string{"local", "remote", "dynamic"} {
		t.Run(kind, func(t *testing.T) {
			echo := echoServer(t)
			ssh := testutil.StartSSH(t, nil)
			m := testutil.Manager(t, map[string]inventory.Host{"a": ssh.Host()}, ssh)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := m.Prepare(ctx, []string{"a"}, false); err != nil {
				t.Fatal(err)
			}
			client, err := m.Dial(ctx, "a")
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			f := inventory.Forward{Name: "fixture", Type: kind, Listen: "127.0.0.1:0", Destination: echo.Addr().String()}
			if kind == "dynamic" {
				f.Destination = ""
			}
			session, err := forwarding.Start(ctx, client.Client, []inventory.Forward{f}, time.Second, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			address := session.Bindings[0].Listen
			var conn net.Conn
			if kind == "dynamic" {
				dialer, err := proxy.SOCKS5("tcp", address, nil, &net.Dialer{Timeout: time.Second})
				if err != nil {
					t.Fatal(err)
				}
				conn, err = dialer.(proxy.ContextDialer).DialContext(ctx, "tcp", echo.Addr().String())
			} else {
				conn, err = net.DialTimeout("tcp", address, time.Second)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			payload := "real bidirectional tunnel traffic"
			if _, err := io.WriteString(conn, payload); err != nil {
				t.Fatal(err)
			}
			got := make([]byte, len(payload))
			if _, err := io.ReadFull(conn, got); err != nil || string(got) != payload {
				t.Fatalf("echo=%q err=%v", got, err)
			}
			cancel()
			done := make(chan error, 1)
			go func() { done <- session.Wait() }()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("cancellation did not close active streams")
			}
			// A remote listener closes after the SSH peer observes the disconnect.
			// Session.Wait only waits for resources owned by this client.
			deadline := time.Now().Add(2 * time.Second)
			for {
				c, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
				if err != nil {
					break
				}
				_ = c.Close()
				if kind != "remote" || time.Now().After(deadline) {
					t.Fatal("listener survived cancellation")
				}
				time.Sleep(10 * time.Millisecond)
			}
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			if _, err := conn.Read(make([]byte, 1)); err == nil {
				t.Fatal("active stream survived cancellation")
			}
		})
	}
}
func TestStartupFailureRollsBackListeners(t *testing.T) {
	ssh := testutil.StartSSH(t, nil)
	m := testutil.Manager(t, map[string]inventory.Host{"a": ssh.Host()}, ssh)
	if err := m.Prepare(t.Context(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	client, err := m.Dial(t.Context(), "a")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	first := reserved.Addr().String()
	reserved.Close()
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	_, err = forwarding.Start(t.Context(), client.Client, []inventory.Forward{
		{Name: "first", Type: "dynamic", Listen: first},
		{Name: "busy", Type: "dynamic", Listen: busy.Addr().String()},
	}, time.Second, nil)
	if err == nil {
		t.Fatal("occupied listener was accepted")
	}
	ln, err := net.Listen("tcp", first)
	if err != nil {
		t.Fatalf("startup leaked listener: %v", err)
	}
	ln.Close()
}
func TestUnexpectedSSHDisconnectStopsListener(t *testing.T) {
	ssh := testutil.StartSSH(t, nil)
	m := testutil.Manager(t, map[string]inventory.Host{"a": ssh.Host()}, ssh)
	if err := m.Prepare(t.Context(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	client, err := m.Dial(t.Context(), "a")
	if err != nil {
		t.Fatal(err)
	}
	session, err := forwarding.Start(t.Context(), client.Client, []inventory.Forward{{Name: "socks", Type: "dynamic", Listen: "127.0.0.1:0"}}, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	client.Close()
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("disconnect was reported as success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("disconnect did not stop forwarding")
	}
}
