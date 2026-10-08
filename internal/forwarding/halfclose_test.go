// SPDX-License-Identifier: GPL-3.0-only
package forwarding_test

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/KanataLabs/fleetsh/internal/forwarding"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
)

func TestLocalForwardPreservesResponseAfterHalfClose(t *testing.T) {
	destination, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	replied := make(chan error, 1)
	go func() {
		conn, err := destination.Accept()
		if err != nil {
			replied <- err
			return
		}
		defer conn.Close()
		request, err := io.ReadAll(conn)
		if err == nil {
			_, err = io.WriteString(conn, "reply:"+string(request))
		}
		replied <- err
	}()
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
	session, err := forwarding.Start(t.Context(), client.Client, []inventory.Forward{{Name: "half-close", Type: "local", Listen: "127.0.0.1:0", Destination: destination.Addr().String()}}, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	conn, err := net.DialTimeout("tcp", session.Bindings[0].Listen, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, "request"); err != nil {
		t.Fatal(err)
	}
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(conn)
	if err != nil || string(got) != "reply:request" {
		t.Fatalf("response=%q err=%v", got, err)
	}
	if err := <-replied; err != nil {
		t.Fatal(err)
	}
}
