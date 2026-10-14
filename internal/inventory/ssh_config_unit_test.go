// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"slices"
	"strings"
	"testing"
)

func TestSSHFieldsHandleQuotesCommentsAndEquals(t *testing.T) {
	for _, test := range []struct {
		name, input string
		want        []string
		bad         bool
	}{
		{"blank", "  # comment", nil, false},
		{"assignment", "Port=2222 # comment", []string{"Port", "2222"}, false},
		{"quoted path", `IdentityFile "~/.ssh/key with spaces"`, []string{"IdentityFile", "~/.ssh/key with spaces"}, false},
		{"quoted comment", `HostName "hash#inside.example"`, []string{"HostName", "hash#inside.example"}, false},
		{"escaped quote", `IdentityFile "key\"name"`, []string{"IdentityFile", `key"name`}, false},
		{"unterminated", `IdentityFile "do-not-echo-this-value`, nil, true},
		{"unfinished escape", `HostName example\`, nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields, err := sshFields(test.input)
			if (err != nil) != test.bad {
				t.Fatalf("error=%v bad=%v", err, test.bad)
			}
			if !test.bad && !slices.Equal(fields, test.want) {
				t.Fatalf("fields=%q want=%q", fields, test.want)
			}
			if err != nil && strings.Contains(err.Error(), "do-not-echo-this-value") {
				t.Fatal("parser echoed a field value")
			}
		})
	}
}
func TestSSHMatchingHonorsNegationAndCase(t *testing.T) {
	for _, test := range []struct {
		patterns []string
		host     string
		want     bool
	}{
		{[]string{"*", "!blocked"}, "blocked", false},
		{[]string{"!blocked", "*"}, "hk1", true},
		{[]string{"!blocked"}, "hk1", false},
		{[]string{"HK?"}, "hk1", true},
		{[]string{"hk[12]"}, "hk2", true},
		{[]string{"hk*"}, "sg1", false},
	} {
		if got := matchesSSH(test.patterns, test.host); got != test.want {
			t.Fatalf("patterns=%v host=%s got=%v", test.patterns, test.host, got)
		}
	}
}
func TestSSHImportRejectsInvalidPorts(t *testing.T) {
	for _, port := range []string{"0", "-1", "65536", "text"} {
		input := "Host hk1\n User root\n Port " + port + "\n"
		if _, err := ParseSSHConfig(strings.NewReader(input), ""); err == nil {
			t.Fatalf("invalid port %s accepted", port)
		}
	}
	for _, port := range []string{"1", "65535"} {
		if _, err := ParseSSHConfig(strings.NewReader("Host hk1\n User root\n Port "+port+"\n"), ""); err != nil {
			t.Fatal(port, err)
		}
	}
}
func TestSSHImportUsesFirstValueAndFallbackUser(t *testing.T) {
	input := "Host *\n Port 2222\nHost hk1 hk2\n Port 2200\n HostName %h.example\n IdentityFile ~/.ssh/%n-key\n"
	result, err := ParseSSHConfig(strings.NewReader(input), "ubuntu")
	if err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"hk1", "hk2"} {
		h := result.Hosts[alias]
		if h.Port != 2222 || h.User != "ubuntu" || h.Host != alias+".example" || h.Key != "~/.ssh/"+alias+"-key" {
			t.Fatalf("%s: %+v", alias, h)
		}
	}
	withoutUser, err := ParseSSHConfig(strings.NewReader("Host hk1\n HostName hk1.example\n"), "")
	if err != nil || len(withoutUser.Hosts) != 0 || len(withoutUser.Warnings) == 0 {
		t.Fatal("missing User was silently accepted")
	}
}
func TestSSHImportDoesNotEchoRejectedValues(t *testing.T) {
	for _, line := range []string{`IdentityFile "private-marker`, "Include private-marker", "Match exec private-marker"} {
		_, err := ParseSSHConfig(strings.NewReader("Host hk1\n User root\n "+line+"\n"), "")
		if err == nil || strings.Contains(err.Error(), "private-marker") {
			t.Fatalf("unsafe diagnostic: %v", err)
		}
	}
}
