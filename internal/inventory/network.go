// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Forward describes a named TCP tunnel, started explicitly by fleetsh forward.
type Forward struct {
	Name        string `toml:"name" json:"name"`
	Type        string `toml:"type" json:"type"`
	Listen      string `toml:"listen" json:"listen"`
	Destination string `toml:"destination,omitempty" json:"destination,omitempty"`
}

// ProxyFor resolves inheritance without writing effective settings into a host.
func (inv *Inventory) ProxyFor(h Host) (string, string) {
	if h.ProxyJump != "" || h.Proxy == "direct" {
		return "", ""
	}
	if h.Proxy != "" {
		return h.Proxy, h.ProxyCredential
	}
	if inv.Defaults.Proxy == "direct" {
		return "", ""
	}
	ref := h.ProxyCredential
	if ref == "" {
		ref = inv.Defaults.ProxyCredential
	}
	return inv.Defaults.Proxy, ref
}

func validateProxy(value string) error {
	if value == "" || value == "direct" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "socks5" && u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(u.Hostname(), " \t\r\n\x00\\") {
		return errors.New("use a SOCKS5/HTTP/HTTPS proxy URL without userinfo; store secrets by reference")
	}
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("invalid proxy port")
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return errors.New("invalid proxy port")
	}
	return nil
}

func validateForwards(forwards []Forward) error {
	names := map[string]bool{}
	for _, f := range forwards {
		if !ValidName(f.Name) || names[f.Name] {
			return errors.New("port forward names must be valid and unique per host")
		}
		names[f.Name] = true
		if f.Type != "local" && f.Type != "remote" && f.Type != "dynamic" {
			return fmt.Errorf("forward %s: type must be local, remote or dynamic", f.Name)
		}
		if err := endpoint(f.Listen, true); err != nil {
			return fmt.Errorf("forward %s: invalid listen address: %w", f.Name, err)
		}
		if f.Type == "dynamic" {
			if f.Destination != "" {
				return fmt.Errorf("forward %s: dynamic forwarding has no destination", f.Name)
			}
		} else if err := endpoint(f.Destination, false); err != nil {
			return fmt.Errorf("forward %s: invalid destination: %w", f.Name, err)
		}
	}
	return nil
}

func endpoint(value string, listen bool) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" || strings.ContainsAny(host, " \t\r\n\x00/\\") {
		return errors.New("use HOST:PORT (IPv6: [ADDRESS]:PORT)")
	}
	n, err := strconv.Atoi(port)
	min := 1
	if listen {
		min = 0
	}
	if err != nil || n < min || n > 65535 {
		return errors.New("invalid TCP port")
	}
	if listen && net.ParseIP(host) == nil && host != "localhost" {
		return errors.New("listen host must be an IP address or localhost")
	}
	return nil
}
