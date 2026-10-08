// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProxyInheritanceAndHostOverrides(t *testing.T) {
	defaults := Defaults{Proxy: "socks5://127.0.0.1:1080", ProxyCredential: "global"}
	for _, tc := range []struct {
		name       string
		host       Host
		proxy, ref string
	}{
		{"inherited", Host{}, "socks5://127.0.0.1:1080", "global"},
		{"credential override", Host{ProxyCredential: "host"}, "socks5://127.0.0.1:1080", "host"},
		{"explicit route", Host{Proxy: "http://127.0.0.1:7890"}, "http://127.0.0.1:7890", ""},
		{"explicit credential", Host{Proxy: "http://127.0.0.1:7890", ProxyCredential: "host"}, "http://127.0.0.1:7890", "host"},
		{"direct", Host{Proxy: "direct"}, "", ""},
		{"jump", Host{ProxyJump: "gateway"}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inv := Inventory{Defaults: defaults}
			before := tc.host
			route, ref := inv.ProxyFor(tc.host)
			if route != tc.proxy || ref != tc.ref || !reflect.DeepEqual(before, tc.host) {
				t.Fatalf("effective route=%q ref=%q", route, ref)
			}
		})
	}
}
func TestNetworkConfigurationRoundTrip(t *testing.T) {
	text := "[defaults]\nproxy='http://127.0.0.1:7890'\nproxy_credential='global'\n[hosts.a]\nhost='a.example.com'\nuser='root'\n[[hosts.a.forwards]]\nname='web'\ntype='local'\nlisten='127.0.0.1:8080'\ndestination='127.0.0.1:80'\n[[hosts.a.forwards]]\nname='reverse'\ntype='remote'\nlisten='127.0.0.1:0'\ndestination='127.0.0.1:3000'\n[[hosts.a.forwards]]\nname='socks'\ntype='dynamic'\nlisten='[::1]:1080'\n"
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	inv, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Update(t.Context(), path, func(current *Inventory) error {
		h := current.Hosts["a"]
		h.Description = "changed"
		current.Hosts["a"] = h
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inv.Hosts["a"].Forwards, after.Hosts["a"].Forwards) || after.Defaults.Proxy != inv.Defaults.Proxy || after.Hosts["a"].Proxy != "" {
		t.Fatal("host editing lost profiles or persisted inherited proxy")
	}
}
func TestInvalidNetworkSettings(t *testing.T) {
	for _, value := range []string{"http://user:secret@localhost:7890", "socks5://localhost:0", "https://localhost:99999", "ftp://localhost", "http://localhost/path", "http://localhost:", "http://localhost?secret=abc"} {
		t.Run(value, func(t *testing.T) {
			if validateProxy(value) == nil {
				t.Fatal("invalid proxy accepted")
			}
		})
	}
	valid := Forward{Name: "web", Type: "local", Listen: "127.0.0.1:8080", Destination: "localhost:80"}
	for _, tc := range []struct {
		name     string
		forwards []Forward
	}{
		{"duplicate", []Forward{valid, valid}},
		{"unknown type", []Forward{{Name: "a", Type: "udp", Listen: "127.0.0.1:0"}}},
		{"no listen host", []Forward{{Name: "a", Type: "dynamic", Listen: ":1080"}}},
		{"no target", []Forward{{Name: "a", Type: "local", Listen: "127.0.0.1:0"}}},
		{"target on dynamic", []Forward{{Name: "a", Type: "dynamic", Listen: "127.0.0.1:0", Destination: "localhost:80"}}},
		{"zero target port", []Forward{{Name: "a", Type: "remote", Listen: "127.0.0.1:0", Destination: "localhost:0"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if validateForwards(tc.forwards) == nil {
				t.Fatal("invalid forward accepted")
			}
		})
	}
}
