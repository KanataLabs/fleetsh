// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/net/proxy"
)

type Manager struct {
	Inventory *inventory.Inventory
	Keys      *HostKeys
	Store     credentials.Store
	Prompt    func(string) (string, error)
	Auth      map[string][]ssh.AuthMethod
	ProxyAuth map[string]*proxy.Auth
	Sudo      map[string]string
	Secrets   []string
	closers   []net.Conn
	mu        sync.Mutex
	closed    bool
	stopClose func() bool
}

func (m *Manager) secret(ref, label string) (string, error) {
	var s string
	var err error
	if ref != "" {
		s, err = m.Store.Get(ref)
	} else if m.Prompt != nil {
		s, err = m.Prompt(label)
	} else {
		err = errors.New("credential reference required in unattended operation")
	}
	if err != nil {
		return "", err
	}
	if s == "" {
		return "", errors.New("empty secret is not supported")
	}
	m.Secrets = append(m.Secrets, s)
	return s, nil
}
func ExpandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// Prepare resolves secret input before parallel work, preventing interleaved prompts.
func (m *Manager) Prepare(ctx context.Context, ids []string, sudo bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return context.Canceled
	}
	m.stopClose = context.AfterFunc(ctx, m.Close)
	m.mu.Unlock()
	m.Auth = map[string][]ssh.AuthMethod{}
	m.ProxyAuth = map[string]*proxy.Auth{}
	m.Sudo = map[string]string{}
	seen := map[string]bool{}
	var prepare func(string) error
	prepare = func(id string) error {
		if seen[id] {
			return nil
		}
		seen[id] = true
		h := m.Inventory.Hosts[id]
		if h.Connection == "console-only" {
			return nil
		}
		if h.ProxyJump != "" {
			if err := prepare(h.ProxyJump); err != nil {
				return err
			}
		}
		switch h.Auth {
		case "password":
			s, err := m.secret(h.Credential, "SSH password for "+id)
			if err != nil {
				return fmt.Errorf("host %s: %w", id, err)
			}
			m.Auth[id] = []ssh.AuthMethod{ssh.Password(s)}
		case "key":
			path, err := ExpandPath(h.Key)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("host %s: cannot read key file", id)
			}
			signer, err := ssh.ParsePrivateKey(data)
			var encrypted *ssh.PassphraseMissingError
			if errors.As(err, &encrypted) {
				s, e := m.secret(h.Credential, "Key passphrase for "+id)
				if e != nil {
					return e
				}
				signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(s))
			}
			if err != nil {
				return fmt.Errorf("host %s: cannot parse private key", id)
			}
			m.Auth[id] = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		case "agent":
			agentCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			conn, err := dialAgent(agentCtx)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return agentPreparationError(id, "SSH agent unavailable")
			}
			if err = m.register(conn); err != nil {
				return err
			}
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
			signers, err := agent.NewClient(conn).Signers()
			_ = conn.SetDeadline(time.Time{})
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				return agentPreparationError(id, "cannot read keys from SSH agent")
			}
			if len(signers) == 0 {
				return agentPreparationError(id, "SSH agent has no usable keys")
			}
			bounded := make([]ssh.Signer, 0, len(signers))
			for _, signer := range signers {
				bounded = append(bounded, timedSigner{Signer: signer, conn: conn})
			}
			m.Auth[id] = []ssh.AuthMethod{ssh.PublicKeys(bounded...)}
		}
		_, proxyCredential := m.Inventory.ProxyFor(h)
		if proxyCredential != "" {
			s, err := m.secret(proxyCredential, "Proxy credential for "+id)
			if err != nil {
				return err
			}
			user, pass, ok := strings.Cut(s, ":")
			if !ok || user == "" || pass == "" {
				return errors.New("proxy credential must be username:password")
			}
			m.Secrets = append(m.Secrets, pass)
			m.ProxyAuth[id] = &proxy.Auth{User: user, Password: pass}
		}
		if sudo && h.SudoCredential != "" {
			s, err := m.secret(h.SudoCredential, "sudo password for "+id)
			if err != nil {
				return err
			}
			if strings.ContainsAny(s, "\r\n") {
				return errors.New("sudo password cannot contain line breaks")
			}
			m.Sudo[id] = s
		}
		return nil
	}
	for _, id := range ids {
		if err := prepare(id); err != nil {
			return err
		}
	}
	return nil
}

func agentPreparationError(id, reason string) error {
	return fmt.Errorf("host %s: %s (auth=agent); for password login, run 'fleetsh edit %s --auth password'; for key-file login, run 'fleetsh edit %s --auth key --key PATH'; otherwise start SSH agent and load a key", id, reason, id, id)
}

type Client struct {
	*ssh.Client
	parents []*Client
}

func (c *Client) Close() error {
	err := c.Client.Close()
	for _, p := range c.parents {
		_ = p.Close()
	}
	return err
}
func (m *Manager) Dial(ctx context.Context, id string) (*Client, error) {
	timeout, _ := time.ParseDuration(m.Inventory.Defaults.ConnectTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.dial(ctx, id)
}
func (m *Manager) dial(ctx context.Context, id string) (*Client, error) {
	h := m.Inventory.Hosts[id]
	var conn net.Conn
	var err error
	var parent *Client
	proxyURL, _ := m.Inventory.ProxyFor(h)
	if h.ProxyJump != "" {
		parent, err = m.dial(ctx, h.ProxyJump)
		if err != nil {
			return nil, fmt.Errorf("jump host: %w", err)
		}
		conn, err = parent.DialContext(ctx, "tcp", h.Address())
	} else if proxyURL != "" {
		conn, err = dialProxy(ctx, proxyURL, h.Address(), m.ProxyAuth[id])
		if err != nil {
			err = fmt.Errorf("proxy connection failed: %w", err)
		}
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", h.Address())
	}
	if err != nil {
		if parent != nil {
			_ = parent.Close()
		}
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	type handshake struct {
		connection ssh.Conn
		channels   <-chan ssh.NewChannel
		requests   <-chan *ssh.Request
		err        error
	}
	completed := make(chan handshake, 1)
	go func() {
		connection, channels, requests, e := ssh.NewClientConn(conn, h.Address(), &ssh.ClientConfig{User: h.User, Auth: m.Auth[id], HostKeyCallback: m.Keys.Callback(ctx)})
		completed <- handshake{connection, channels, requests, e}
	}()
	var result handshake
	select {
	case result = <-completed:
	case <-ctx.Done():
		stop()
		_ = conn.Close()
		if parent != nil {
			_ = parent.Close()
		}
		return nil, ctx.Err()
	}
	sshConn, chans, reqs, err := result.connection, result.channels, result.requests, result.err
	stopped := stop()
	if err != nil || ctx.Err() != nil || !stopped {
		_ = conn.Close()
		if parent != nil {
			_ = parent.Close()
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			return nil, err
		}
		return nil, errors.New("connection canceled")
	}
	_ = conn.SetDeadline(time.Time{})
	c := &Client{Client: ssh.NewClient(sshConn, chans, reqs)}
	if parent != nil {
		c.parents = []*Client{parent}
	}
	go keepalive(c)
	return c, nil
}
func keepalive(c *Client) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	done := make(chan struct{})
	go func() { _ = c.Wait(); close(done) }()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			reply := make(chan error, 1)
			go func() { _, _, err := c.SendRequest("keepalive@openssh.com", true, nil); reply <- err }()
			timer := time.NewTimer(5 * time.Second)
			select {
			case err := <-reply:
				timer.Stop()
				if err != nil {
					_ = c.Close()
					return
				}
			case <-timer.C:
				_ = c.Close()
				return
			case <-done:
				timer.Stop()
				return
			}
		}
	}
}
