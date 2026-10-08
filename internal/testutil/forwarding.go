// SPDX-License-Identifier: GPL-3.0-only
package testutil

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"

	"golang.org/x/crypto/ssh"
)

// forwardingRequests provides real loopback-only remote TCP forwarding for tests.
func (s *Server) forwardingRequests(ctx context.Context, connection *ssh.ServerConn, requests <-chan *ssh.Request) {
	defer s.wg.Done()
	var mu sync.Mutex
	listeners := map[string]net.Listener{}
	closeAll := func() {
		mu.Lock()
		defer mu.Unlock()
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}
	stop := context.AfterFunc(ctx, closeAll)
	defer func() { stop(); closeAll() }()
	for req := range requests {
		if req.Type != "tcpip-forward" && req.Type != "cancel-tcpip-forward" {
			_ = req.Reply(false, nil)
			continue
		}
		var p struct {
			Address string
			Port    uint32
		}
		if ssh.Unmarshal(req.Payload, &p) != nil || (p.Address != "127.0.0.1" && p.Address != "::1") || p.Port > 65535 {
			_ = req.Reply(false, nil)
			continue
		}
		key := net.JoinHostPort(p.Address, strconv.Itoa(int(p.Port)))
		if req.Type == "cancel-tcpip-forward" {
			mu.Lock()
			ln, ok := listeners[key]
			delete(listeners, key)
			mu.Unlock()
			if ok {
				_ = ln.Close()
			}
			_ = req.Reply(ok, nil)
			continue
		}
		ln, err := net.Listen("tcp", key)
		if err != nil {
			_ = req.Reply(false, nil)
			continue
		}
		port := uint32(ln.Addr().(*net.TCPAddr).Port)
		key = net.JoinHostPort(p.Address, strconv.Itoa(int(port)))
		mu.Lock()
		listeners[key] = ln
		mu.Unlock()
		if err := req.Reply(true, ssh.Marshal(struct{ Port uint32 }{port})); err != nil {
			_ = ln.Close()
			continue
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					defer conn.Close()
					stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
					defer stop()
					origin := conn.RemoteAddr().(*net.TCPAddr)
					payload := ssh.Marshal(struct {
						Address    string
						Port       uint32
						Origin     string
						OriginPort uint32
					}{p.Address, port, origin.IP.String(), uint32(origin.Port)})
					ch, reqs, err := connection.OpenChannel("forwarded-tcpip", payload)
					if err != nil {
						return
					}
					defer ch.Close()
					go ssh.DiscardRequests(reqs)
					done := make(chan struct{})
					go func() { _, _ = io.Copy(ch, conn); _ = ch.CloseWrite(); close(done) }()
					_, _ = io.Copy(conn, ch)
					_ = conn.Close()
					_ = ch.Close()
					<-done
				}()
			}
		}()
	}
}
