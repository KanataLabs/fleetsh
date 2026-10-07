// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/pelletier/go-toml/v2"
)

type Defaults struct {
	ConnectTimeout string `toml:"connect_timeout" json:"connect_timeout"`
	CommandTimeout string `toml:"command_timeout" json:"command_timeout"`
	Parallel       int    `toml:"parallel" json:"parallel"`
}
type Host struct {
	Host            string   `toml:"host" json:"host"`
	Port            int      `toml:"port,omitempty" json:"port"`
	User            string   `toml:"user,omitempty" json:"user"`
	Connection      string   `toml:"connection,omitempty" json:"connection"`
	Auth            string   `toml:"auth,omitempty" json:"auth"`
	Credential      string   `toml:"credential,omitempty" json:"credential,omitempty"`
	Key             string   `toml:"key,omitempty" json:"key,omitempty"`
	Proxy           string   `toml:"proxy,omitempty" json:"proxy,omitempty"`
	ProxyJump       string   `toml:"proxy_jump,omitempty" json:"proxy_jump,omitempty"`
	ProxyCredential string   `toml:"proxy_credential,omitempty" json:"proxy_credential,omitempty"`
	SudoCredential  string   `toml:"sudo_credential,omitempty" json:"sudo_credential,omitempty"`
	Groups          []string `toml:"groups,omitempty" json:"groups,omitempty"`
	Tags            []string `toml:"tags,omitempty" json:"tags,omitempty"`
	Description     string   `toml:"description,omitempty" json:"description,omitempty"`
}

func (h Host) Address() string { return net.JoinHostPort(h.Host, strconv.Itoa(h.Port)) }

type Inventory struct {
	Defaults    Defaults        `toml:"defaults" json:"defaults"`
	Credentials []string        `toml:"credentials,omitempty" json:"credentials,omitempty"`
	Hosts       map[string]Host `toml:"hosts" json:"hosts"`
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func ValidName(s string) bool { return namePattern.MatchString(s) }
func Load(path string) (*Inventory, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("inventory must be a regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}
	var inv Inventory
	if err = toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&inv); err != nil {
		// Parser diagnostics can echo accidentally pasted secret values.
		return nil, errors.New("invalid inventory TOML or unsupported field; plaintext passwords are forbidden")
	}
	if err = inv.Validate(); err != nil {
		return nil, err
	}
	return &inv, nil
}
func (inv *Inventory) Validate() error {
	if inv.Hosts == nil {
		inv.Hosts = map[string]Host{}
	}
	if inv.Defaults.ConnectTimeout == "" {
		inv.Defaults.ConnectTimeout = "10s"
	}
	if inv.Defaults.CommandTimeout == "" {
		inv.Defaults.CommandTimeout = "30m"
	}
	if inv.Defaults.Parallel == 0 {
		inv.Defaults.Parallel = 10
	}
	if inv.Defaults.Parallel < 1 || inv.Defaults.Parallel > 256 {
		return errors.New("parallel must be between 1 and 256")
	}
	for _, v := range []string{inv.Defaults.ConnectTimeout, inv.Defaults.CommandTimeout} {
		if d, err := time.ParseDuration(v); err != nil || d <= 0 {
			return errors.New("timeouts must be positive durations")
		}
	}
	for _, ref := range inv.Credentials {
		if !ValidName(ref) {
			return errors.New("invalid credential reference")
		}
	}
	for id, h := range inv.Hosts {
		if !ValidName(id) || id == "all" {
			return errors.New("invalid or reserved host alias")
		}
		if h.Host == "" || strings.ContainsAny(h.Host, " \t\r\n/\\") {
			return fmt.Errorf("host %s: invalid hostname", id)
		}
		if h.Port == 0 {
			h.Port = 22
		}
		if h.Port < 1 || h.Port > 65535 {
			return fmt.Errorf("host %s: invalid port", id)
		}
		if h.Connection == "" {
			h.Connection = "ssh"
		}
		if h.Connection != "ssh" && h.Connection != "console-only" {
			return fmt.Errorf("host %s: invalid connection type", id)
		}
		if h.Auth == "" {
			h.Auth = "agent"
		}
		if h.Auth != "agent" && h.Auth != "password" && h.Auth != "key" {
			return fmt.Errorf("host %s: invalid auth type", id)
		}
		if h.Connection == "ssh" && strings.TrimSpace(h.User) == "" {
			return fmt.Errorf("host %s: user is required", id)
		}
		if strings.ContainsAny(h.User, "\r\n\x00") {
			return fmt.Errorf("host %s: invalid user", id)
		}
		if h.Auth == "key" && h.Key == "" {
			return fmt.Errorf("host %s: key path is required", id)
		}
		if h.Proxy != "" {
			u, err := url.Parse(h.Proxy)
			if err != nil || u.Scheme != "socks5" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
				return fmt.Errorf("host %s: use a SOCKS5 URL without userinfo; store secrets by reference", id)
			}
			if p := u.Port(); p != "" {
				n, e := strconv.Atoi(p)
				if e != nil || n < 1 || n > 65535 {
					return fmt.Errorf("host %s: invalid proxy port", id)
				}
			}
		}
		if h.Proxy != "" && h.ProxyJump != "" {
			return fmt.Errorf("host %s: choose proxy or proxy_jump", id)
		}
		if h.ProxyCredential != "" && h.Proxy == "" {
			return fmt.Errorf("host %s: proxy_credential requires proxy", id)
		}
		for _, ref := range []string{h.Credential, h.ProxyCredential, h.SudoCredential} {
			if ref != "" && !ValidName(ref) {
				return fmt.Errorf("host %s: invalid credential reference", id)
			}
		}
		for _, n := range append(append([]string{}, h.Groups...), h.Tags...) {
			if !ValidName(n) {
				return fmt.Errorf("host %s: invalid group/tag", id)
			}
		}
		inv.Hosts[id] = h
	}
	for id := range inv.Hosts {
		seen := map[string]bool{}
		for next := id; next != ""; {
			if seen[next] {
				return fmt.Errorf("host %s: proxy_jump cycle", id)
			}
			seen[next] = true
			h, ok := inv.Hosts[next]
			if !ok || (next != id && h.Connection == "console-only") {
				return fmt.Errorf("host %s: invalid jump host", id)
			}
			next = h.ProxyJump
		}
	}
	return nil
}
func (inv *Inventory) Select(selector, tag string) ([]string, error) {
	selected := map[string]bool{}
	if selector == "" {
		selector = "all"
	}
	for _, s := range strings.Split(selector, ",") {
		found := false
		switch {
		case s == "all" || s == "@all":
			for id := range inv.Hosts {
				selected[id] = true
			}
			found = true
		case strings.HasPrefix(s, "@"):
			for id, h := range inv.Hosts {
				for _, g := range h.Groups {
					if g == s[1:] {
						selected[id] = true
						found = true
					}
				}
			}
		default:
			if _, ok := inv.Hosts[s]; ok {
				selected[s] = true
				found = true
			}
		}
		if !found {
			return nil, errors.New("selector contains an unknown host or group")
		}
	}
	ids := []string{}
	for id := range selected {
		if tag != "" {
			matched := false
			for _, t := range inv.Hosts[id].Tags {
				if t == tag {
					matched = true
				}
			}
			if !matched {
				continue
			}
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil, errors.New("no hosts match the selector")
	}
	return ids, nil
}
func Update(ctx context.Context, path string, change func(*Inventory) error) error {
	lock := flock.New(path + ".lock")
	ok, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return fmt.Errorf("lock inventory: %w", err)
	}
	if !ok {
		return errors.New("inventory lock timed out")
	}
	defer lock.Close()
	inv, err := Load(path)
	if err != nil {
		return err
	}
	if err = change(inv); err != nil {
		return err
	}
	if err = inv.Validate(); err != nil {
		return err
	}
	data, err := toml.Marshal(inv)
	if err != nil {
		return errors.New("cannot encode inventory")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".fleetsh-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("save inventory: %w", err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace inventory: %w", err)
	}
	return nil
}
