// SPDX-License-Identifier: GPL-3.0-only
package testutil

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
)

func StartSOCKS(t *testing.T, authenticated bool) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	connections := map[net.Conn]bool{}
	var wg sync.WaitGroup
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
				header := make([]byte, 2)
				if _, err := io.ReadFull(conn, header); err != nil || header[0] != 5 {
					return
				}
				methods := make([]byte, int(header[1]))
				if _, err := io.ReadFull(conn, methods); err != nil {
					return
				}
				method := byte(0)
				if authenticated {
					method = 2
				}
				_, _ = conn.Write([]byte{5, method})
				if authenticated {
					if _, err := io.ReadFull(conn, header); err != nil || header[0] != 1 {
						return
					}
					user := make([]byte, int(header[1]))
					if _, err := io.ReadFull(conn, user); err != nil {
						return
					}
					size := make([]byte, 1)
					if _, err := io.ReadFull(conn, size); err != nil {
						return
					}
					pass := make([]byte, int(size[0]))
					if _, err := io.ReadFull(conn, pass); err != nil {
						return
					}
					if string(user) != "proxy-user" || string(pass) != "proxy-pass" {
						_, _ = conn.Write([]byte{1, 1})
						return
					}
					_, _ = conn.Write([]byte{1, 0})
				}
				request := make([]byte, 4)
				if _, err := io.ReadFull(conn, request); err != nil || request[1] != 1 {
					return
				}
				var host string
				switch request[3] {
				case 1:
					addr := make([]byte, 4)
					if _, err := io.ReadFull(conn, addr); err != nil {
						return
					}
					host = net.IP(addr).String()
				case 3:
					size := make([]byte, 1)
					if _, err := io.ReadFull(conn, size); err != nil {
						return
					}
					addr := make([]byte, int(size[0]))
					if _, err := io.ReadFull(conn, addr); err != nil {
						return
					}
					host = string(addr)
				default:
					return
				}
				port := make([]byte, 2)
				if _, err := io.ReadFull(conn, port); err != nil {
					return
				}
				if host != "127.0.0.1" && host != "localhost" {
					return
				}
				target, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(port)))))
				if err != nil {
					return
				}
				defer target.Close()
				_, _ = conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
				done := make(chan struct{})
				go func() { _, _ = io.Copy(target, conn); target.Close(); close(done) }()
				_, _ = io.Copy(conn, target)
				conn.Close()
				<-done
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
	return "socks5://" + listener.Addr().String()
}
