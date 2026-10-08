// SPDX-License-Identifier: GPL-3.0-only
package transport_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
)

func TestGlobalProxyAndPerHostOverrides(t *testing.T) {
	target := testutil.StartSSH(t, nil)
	for _, mode := range []string{"global socks", "global http", "host http override", "direct override", "jump override"} {
		t.Run(mode, func(t *testing.T) {
			h := target.Host()
			hosts := map[string]inventory.Host{"a": h}
			defaults := inventory.Defaults{Proxy: "http://127.0.0.1:1", ProxyCredential: "missing"}
			switch mode {
			case "global socks":
				defaults.Proxy = testutil.StartSOCKS(t, true)
				defaults.ProxyCredential = "proxy"
			case "global http":
				defaults.Proxy = testutil.StartHTTPProxy(t, true)
				defaults.ProxyCredential = "proxy"
			case "host http override":
				h.Proxy = testutil.StartHTTPProxy(t, false)
			case "direct override":
				h.Proxy = "direct"
			case "jump override":
				jump := testutil.StartSSH(t, nil)
				j := jump.Host()
				j.Proxy = "direct"
				hosts["jump"] = j
				h.ProxyJump = "jump"
				m := testutil.Manager(t, map[string]inventory.Host{"a": h, "jump": j}, target, jump)
				m.Inventory.Defaults.Proxy = defaults.Proxy
				m.Inventory.Defaults.ProxyCredential = defaults.ProxyCredential
				if err := m.Prepare(t.Context(), []string{"a"}, false); err != nil {
					t.Fatal(err)
				}
				if r := executor.Run(t.Context(), m, "a", "uptime", executor.Options{}); !r.Success {
					t.Fatalf("jump=%+v", r)
				}
				return
			}
			hosts["a"] = h
			m := testutil.Manager(t, hosts, target)
			m.Inventory.Defaults.Proxy = defaults.Proxy
			m.Inventory.Defaults.ProxyCredential = defaults.ProxyCredential
			if err := m.Inventory.Validate(); err != nil {
				t.Fatal(err)
			}
			if err := m.Prepare(t.Context(), []string{"a"}, false); err != nil {
				t.Fatal(err)
			}
			r := executor.Run(t.Context(), m, "a", "uptime", executor.Options{})
			if !r.Success {
				t.Fatalf("proxy=%+v", r)
			}
		})
	}
}
func TestHTTPProxyAuthenticationFailureDoesNotEchoResponse(t *testing.T) {
	server := testutil.StartSSH(t, nil)
	h := server.Host()
	h.Proxy = testutil.StartHTTPProxy(t, true)
	h.ProxyCredential = "proxy"
	m := testutil.Manager(t, map[string]inventory.Host{"a": h}, server)
	m.Store.(testutil.Store)["proxy"] = "proxy-user:wrong-secret"
	if err := m.Prepare(t.Context(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	r := executor.Run(context.Background(), m, "a", "uptime", executor.Options{})
	if r.Success || r.ErrorKind != "proxy" || !strings.Contains(r.Error, "407") || strings.Contains(r.Error, "wrong-secret") || strings.Contains(r.Error, "proxy-pass") {
		t.Fatalf("bad failure=%+v", r)
	}
}
func TestHTTPSProxyRejectsUntrustedCertificate(t *testing.T) {
	tlsProxy := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unexpected", 500) }))
	defer tlsProxy.Close()
	target := testutil.StartSSH(t, nil)
	h := target.Host()
	h.Proxy = tlsProxy.URL
	m := testutil.Manager(t, map[string]inventory.Host{"a": h}, target)
	if err := m.Prepare(t.Context(), []string{"a"}, false); err != nil {
		t.Fatal(err)
	}
	r := executor.Run(t.Context(), m, "a", "uptime", executor.Options{})
	if r.Success || r.ErrorKind != "proxy" {
		t.Fatalf("untrusted TLS proxy accepted: %+v", r)
	}
}
