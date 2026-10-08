// SPDX-License-Identifier: GPL-3.0-only
package testutil

import (
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// StartHTTPProxy implements authenticated CONNECT to loopback destinations only.
func StartHTTPProxy(t *testing.T, authenticated bool) string {
	t.Helper()
	var mu sync.Mutex
	active := map[net.Conn]bool{}
	var wg sync.WaitGroup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "CONNECT required", 405)
			return
		}
		if authenticated && r.Header.Get("Proxy-Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte("proxy-user:proxy-pass")) {
			http.Error(w, "proxy-pass must not appear in reported errors", 407)
			return
		}
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil || (host != "127.0.0.1" && host != "localhost") {
			http.Error(w, "local destinations only", 403)
			return
		}
		target, err := net.Dial("tcp", r.Host)
		if err != nil {
			http.Error(w, "unavailable", 502)
			return
		}
		conn, reader, err := w.(http.Hijacker).Hijack()
		if err != nil {
			target.Close()
			return
		}
		mu.Lock()
		active[conn] = true
		active[target] = true
		mu.Unlock()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer conn.Close()
			defer target.Close()
			defer func() { mu.Lock(); delete(active, conn); delete(active, target); mu.Unlock() }()
			if _, err := io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
				return
			}
			done := make(chan struct{})
			go func() { _, _ = io.Copy(target, reader); target.Close(); close(done) }()
			_, _ = io.Copy(conn, target)
			conn.Close()
			<-done
		}()
	}))
	t.Cleanup(func() {
		server.Close()
		mu.Lock()
		for c := range active {
			c.Close()
		}
		mu.Unlock()
		wg.Wait()
	})
	return server.URL
}
