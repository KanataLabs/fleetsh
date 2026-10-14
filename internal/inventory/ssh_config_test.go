// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"strings"
	"testing"
)

func TestSSHImportDefaultsAndUnsupportedRoutes(t *testing.T) {
	input := `Host good
 HostName good.example
 User ubuntu
 IdentityFile "~/.ssh/key with spaces"
Host blocked
 HostName blocked.example
 ProxyCommand secret-command-value
Host *
 User root
 Port 2222
Host !blocked *
 ServerAliveInterval 30
`
	imported, err := ParseSSHConfig(strings.NewReader(input), "")
	if err != nil {
		t.Fatal(err)
	}
	h := imported.Hosts["good"]
	if h.User != "ubuntu" || h.Port != 2222 || h.Auth != "key" || h.Key != "~/.ssh/key with spaces" {
		t.Fatalf("effective import: %+v", h)
	}
	if _, ok := imported.Hosts["blocked"]; ok {
		t.Fatal("unsupported proxy route imported as direct")
	}
	for _, warning := range imported.Warnings {
		if strings.Contains(warning, "secret-command-value") {
			t.Fatal("ignored value leaked")
		}
	}
	if len(imported.Warnings) == 0 {
		t.Fatal("unsupported directive not reported")
	}
	for _, input := range []string{"Include ~/.ssh/conf.d/*", "Match user root"} {
		if _, err := ParseSSHConfig(strings.NewReader(input), "root"); err == nil {
			t.Fatal("unsupported semantics accepted")
		}
	}
}
func TestSavedCommandValidation(t *testing.T) {
	for _, c := range []Command{{Command: ""}, {Command: "uptime", Timeout: "0s"}, {Command: "uptime", Timeout: "invalid"}} {
		inv := Inventory{Commands: map[string]Command{"probe": c}}
		if err := inv.Validate(); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
}
