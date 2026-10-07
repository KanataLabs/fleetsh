// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/config"
)

func TestSelectorsAndValidation(t *testing.T) {
	inv := &Inventory{Hosts: map[string]Host{
		"hk1":     {Host: "hk.example", User: "root", Groups: []string{"asia", "web"}, Tags: []string{"prod"}},
		"sg1":     {Host: "sg.example", User: "root", Groups: []string{"asia"}},
		"console": {Host: "console.example", Connection: "console-only"},
	}}
	if err := inv.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		selector, tag string
		want          []string
	}{
		{"@asia", "", []string{"hk1", "sg1"}}, {"sg1,hk1,sg1", "", []string{"hk1", "sg1"}}, {"@all", "prod", []string{"hk1"}},
	} {
		got, err := inv.Select(tc.selector, tc.tag)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("selector %s: %v %v", tc.selector, got, err)
		}
	}
	if _, err := inv.Select("@unknown", ""); err == nil {
		t.Fatal("unknown group accepted")
	}
	inv.Hosts["hk1"] = Host{Host: "hk.example", User: "root", ProxyJump: "sg1"}
	inv.Hosts["sg1"] = Host{Host: "sg.example", User: "root", ProxyJump: "hk1"}
	if err := inv.Validate(); err == nil {
		t.Fatal("jump cycle accepted")
	}
}
func TestRejectSecretsWithoutEchoingSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	for _, text := range []string{
		"[hosts.a]\nhost='a.example'\nuser='root'\npassword='VERY-PRIVATE'\n",
		"[hosts.a]\nhost='a.example'\nuser='root'\nproxy='socks5://user:VERY-PRIVATE@127.0.0.1:1080'\n",
		"[hosts.a]\nhost='a.example'\nuser='root'\ncredential='VERY-PRIVATE\n'\n",
	} {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if err == nil {
			t.Fatal("secret configuration accepted")
		}
		if strings.Contains(err.Error(), "VERY-PRIVATE") {
			t.Fatal("error leaked configuration contents")
		}
	}
}
func TestConcurrentUpdatesAndInvalidChangePreserveFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Init(path); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := Update(context.Background(), path, func(inv *Inventory) error {
				inv.Hosts[fmt.Sprintf("h%d", i)] = Host{Host: "example.test", User: "root"}
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	inv, err := Load(path)
	if err != nil || len(inv.Hosts) != 12 {
		t.Fatalf("updates lost: %v %v", inv, err)
	}
	before, _ := os.ReadFile(path)
	if err := Update(context.Background(), path, func(inv *Inventory) error { inv.Hosts["bad"] = Host{Host: "bad.example"}; return nil }); err == nil {
		t.Fatal("invalid mutation accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid mutation changed file")
	}
}
