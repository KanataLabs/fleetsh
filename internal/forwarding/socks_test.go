// SPDX-License-Identifier: GPL-3.0-only
package forwarding

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestSOCKSHandshakeDomainIPv6AndRejections(t *testing.T) {
	for _, tc := range []struct {
		name        string
		request     []byte
		destination string
		code        byte
	}{
		{"domain", append([]byte{5, 1, 0, 3, 9}, append([]byte("localhost"), 0, 80)...), "localhost:80", 0},
		{"ipv6", append([]byte{5, 1, 0, 4}, append(net.ParseIP("::1").To16(), 0, 80)...), "[::1]:80", 0},
		{"udp", []byte{5, 3, 0, 1}, "", 7},
		{"unknown address", []byte{5, 1, 0, 9}, "", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()
			server.SetDeadline(time.Now().Add(time.Second))
			client.SetDeadline(time.Now().Add(time.Second))
			type result struct {
				destination string
				err         error
			}
			done := make(chan result, 1)
			go func() { destination, err := socksDestination(server); done <- result{destination, err} }()
			client.Write([]byte{5, 1, 0})
			var selected [2]byte
			if _, err := io.ReadFull(client, selected[:]); err != nil || selected != [2]byte{5, 0} {
				t.Fatalf("method=%v err=%v", selected, err)
			}
			client.Write(tc.request)
			if tc.destination == "" {
				var reply [10]byte
				if _, err := io.ReadFull(client, reply[:]); err != nil || reply[1] != tc.code {
					t.Fatalf("reply=%v err=%v", reply, err)
				}
			}
			got := <-done
			if got.destination != tc.destination || (got.err != nil) != (tc.destination == "") {
				t.Fatalf("result=%+v", got)
			}
		})
	}
}
func TestSOCKSRejectsUnavailableAuthentication(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	server.SetDeadline(time.Now().Add(time.Second))
	client.SetDeadline(time.Now().Add(time.Second))
	done := make(chan error, 1)
	go func() { _, err := socksDestination(server); done <- err }()
	client.Write([]byte{5, 1, 2})
	var reply [2]byte
	if _, err := io.ReadFull(client, reply[:]); err != nil || reply != [2]byte{5, 255} {
		t.Fatalf("reply=%v err=%v", reply, err)
	}
	if <-done == nil {
		t.Fatal("unsupported authentication accepted")
	}
}
