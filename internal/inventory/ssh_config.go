// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type sshBlock struct {
	patterns []string
	values   map[string]string
}
type SSHImport struct {
	Hosts    map[string]Host `json:"hosts"`
	Warnings []string        `json:"warnings"`
}

func sshFields(line string) ([]string, error) {
	var fields []string
	var word strings.Builder
	var quote rune
	escaped := false
	started := false
	flush := func() {
		if started {
			fields = append(fields, word.String())
			word.Reset()
			started = false
		}
	}
	for _, r := range line {
		if escaped {
			word.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			started = true
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			started = true
		case '#':
			flush()
			return fields, nil
		case ' ', '\t', '=':
			flush()
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("Unclosed quote or escape")
	}
	flush()
	return fields, nil
}
func matchesSSH(patterns []string, alias string) bool {
	matched := false
	for _, p := range patterns {
		negative := strings.HasPrefix(p, "!")
		p = strings.TrimPrefix(p, "!")
		yes, _ := filepath.Match(strings.ToLower(p), strings.ToLower(alias))
		if yes && negative {
			return false
		}
		matched = matched || yes
	}
	return matched
}

// ParseSSHConfig imports literal host aliases and honors first-value-wins wildcard defaults.
// It never runs ProxyCommand/LocalCommand or follows Include/Match directives.
func ParseSSHConfig(reader io.Reader, fallbackUser string) (SSHImport, error) {
	result := SSHImport{Hosts: map[string]Host{}, Warnings: []string{}}
	blocks := []sshBlock{{patterns: []string{"*"}, values: map[string]string{}}}
	aliases := map[string]bool{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		fields, err := sshFields(scanner.Text())
		if err != nil {
			return result, fmt.Errorf("SSH config line %d: %s. No hosts were imported.", line, err)
		}
		if len(fields) == 0 {
			continue
		}
		key := strings.ToLower(fields[0])
		values := fields[1:]
		if key == "include" || key == "match" {
			return result, fmt.Errorf("SSH config line %d uses %s, which this importer cannot resolve. Provide a file with explicit Host sections; no hosts were imported.", line, fields[0])
		}
		if len(values) == 0 {
			return result, fmt.Errorf("SSH config line %d: %s needs a value.", line, fields[0])
		}
		if key == "host" {
			blocks = append(blocks, sshBlock{patterns: values, values: map[string]string{}})
			for _, alias := range values {
				if ValidName(alias) && alias != "all" {
					aliases[alias] = true
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Line %d: pattern %q is used as a default, not imported as a host.", line, alias))
				}
			}
			continue
		}
		switch key {
		case "hostname", "user", "port", "identityfile", "proxyjump", "proxycommand":
			b := &blocks[len(blocks)-1]
			if _, exists := b.values[key]; !exists {
				b.values[key] = strings.Join(values, " ")
			} else if key == "identityfile" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Line %d: only the first IdentityFile is imported.", line))
			}
		default:
			result.Warnings = append(result.Warnings, fmt.Sprintf("Line %d: %s is not supported and is omitted.", line, fields[0]))
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("Cannot read SSH config: %w", err)
	}
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, alias := range names {
		values := map[string]string{}
		for _, b := range blocks {
			if matchesSSH(b.patterns, alias) {
				for key, value := range b.values {
					if _, exists := values[key]; !exists {
						values[key] = value
					}
				}
			}
		}
		if value := values["proxycommand"]; value != "" && value != "none" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Host %s skipped: ProxyCommand cannot be translated safely. Configure an SSH proxy or jump host explicitly.", alias))
			continue
		}
		h := Host{Host: alias, User: values["user"], Port: 22, Auth: "agent", Connection: "ssh"}
		if values["hostname"] != "" {
			h.Host = strings.ReplaceAll(values["hostname"], "%h", alias)
		}
		if h.User == "" {
			h.User = fallbackUser
		}
		if h.User == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Host %s skipped: no User is configured. Repeat import with --user USER.", alias))
			continue
		}
		if value := values["port"]; value != "" {
			port, err := strconv.Atoi(value)
			if err != nil {
				return result, fmt.Errorf("Host %s has an invalid Port. Use a number between 1 and 65535.", alias)
			}
			h.Port = port
		}
		if value := values["identityfile"]; value != "" && value != "none" {
			h.Key = strings.NewReplacer("%h", h.Host, "%n", alias, "%r", h.User).Replace(value)
			h.Auth = "key"
		}
		if value := values["proxyjump"]; value != "" && value != "none" {
			if !ValidName(value) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Host %s skipped: ProxyJump must name a single imported or existing host alias.", alias))
				continue
			}
			h.ProxyJump = value
		}
		if strings.Contains(h.Host, "%") || strings.Contains(h.Key, "%") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Host %s skipped: unsupported SSH expansion token.", alias))
			continue
		}
		result.Hosts[alias] = h
	}
	return result, nil
}
