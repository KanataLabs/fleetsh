// SPDX-License-Identifier: GPL-3.0-only
// Package testutil provides local-only SSH fixtures; it is not linked into fleetsh.
package testutil

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/transport"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const Password = "fixture-password"

type Reply struct {
	Out, Err   string
	Code       uint32
	Disconnect bool
}
type Handler func(context.Context, string, io.Reader) Reply
type Server struct {
	Address        string
	Key, ClientKey ssh.Signer
	Private        ed25519.PrivateKey
	listener       net.Listener
	mu             sync.Mutex
	conns          map[net.Conn]bool
	wg             sync.WaitGroup
}

func StartSSH(t *testing.T, handler Handler) *Server {
	t.Helper()
	_, serverPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverKey, err := ssh.NewSignerFromKey(serverPrivate)
	if err != nil {
		t.Fatal(err)
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientKey, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{Address: listener.Addr().String(), Key: serverKey, ClientKey: clientKey, Private: private, listener: listener, conns: map[net.Conn]bool{}}
	config := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if string(password) != Password {
				return nil, errors.New("incorrect password")
			}
			return nil, nil
		},
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if string(key.Marshal()) != string(clientKey.PublicKey().Marshal()) {
				return nil, errors.New("incorrect key")
			}
			return nil, nil
		},
	}
	config.AddHostKey(serverKey)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.conns[conn] = true
			s.mu.Unlock()
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer conn.Close()
				defer func() { s.mu.Lock(); delete(s.conns, conn); s.mu.Unlock() }()
				sshConn, channels, requests, err := ssh.NewServerConn(conn, config)
				if err != nil {
					return
				}
				defer sshConn.Close()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				go func() { _ = sshConn.Wait(); cancel() }()
				s.wg.Add(1)
				go s.forwardingRequests(ctx, sshConn, requests)
				for channel := range channels {
					switch channel.ChannelType() {
					case "session":
						ch, reqs, err := channel.Accept()
						if err != nil {
							continue
						}
						s.wg.Add(1)
						go func() {
							defer s.wg.Done()
							defer ch.Close()
							for req := range reqs {
								if req.Type == "pty-req" || req.Type == "window-change" {
									_ = req.Reply(true, nil)
									continue
								}
								if req.Type != "exec" && req.Type != "shell" {
									_ = req.Reply(false, nil)
									continue
								}
								command := "shell"
								if req.Type == "exec" {
									var payload struct{ Command string }
									if ssh.Unmarshal(req.Payload, &payload) != nil {
										_ = req.Reply(false, nil)
										return
									}
									command = payload.Command
								}
								_ = req.Reply(true, nil)
								result := Reply{Out: "ok\n"}
								if handler != nil {
									result = handler(ctx, command, ch)
								}
								if result.Disconnect {
									_ = sshConn.Close()
									return
								}
								_, _ = io.WriteString(ch, result.Out)
								_, _ = io.WriteString(ch.Stderr(), result.Err)
								_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{result.Code}))
								return
							}
						}()
					case "direct-tcpip":
						var request struct {
							Host       string
							Port       uint32
							Origin     string
							OriginPort uint32
						}
						if ssh.Unmarshal(channel.ExtraData(), &request) != nil || (request.Host != "127.0.0.1" && request.Host != "::1" && request.Host != "localhost") {
							_ = channel.Reject(ssh.Prohibited, "local targets only")
							continue
						}
						target, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(request.Host, strconv.Itoa(int(request.Port))))
						if err != nil {
							_ = channel.Reject(ssh.ConnectionFailed, "target unavailable")
							continue
						}
						ch, reqs, err := channel.Accept()
						if err != nil {
							target.Close()
							continue
						}
						go ssh.DiscardRequests(reqs)
						s.wg.Add(1)
						go func() {
							defer s.wg.Done()
							defer ch.Close()
							defer target.Close()
							stop := context.AfterFunc(ctx, func() { target.Close() })
							defer stop()
							done := make(chan struct{})
							go func() {
								_, _ = io.Copy(target, ch)
								if tcp, ok := target.(*net.TCPConn); ok {
									_ = tcp.CloseWrite()
								}
								close(done)
							}()
							_, _ = io.Copy(ch, target)
							_ = ch.CloseWrite()
							<-done
						}()
					default:
						_ = channel.Reject(ssh.UnknownChannelType, "unsupported")
					}
				}
			}()
		}
	}()
	t.Cleanup(func() {
		listener.Close()
		s.mu.Lock()
		for c := range s.conns {
			c.Close()
		}
		s.mu.Unlock()
		s.wg.Wait()
	})
	return s
}
func (s *Server) Host() inventory.Host {
	host, port, _ := net.SplitHostPort(s.Address)
	p, _ := strconv.Atoi(port)
	return inventory.Host{Host: host, Port: p, User: "fixture", Auth: "password", Credential: "login"}
}

type Store map[string]string

func (s Store) Get(ref string) (string, error) {
	v, ok := s[ref]
	if !ok {
		return "", credentials.ErrNotFound
	}
	return v, nil
}
func (s Store) Set(ref, value string) error { s[ref] = value; return nil }
func (s Store) Delete(ref string) error {
	if _, ok := s[ref]; !ok {
		return credentials.ErrNotFound
	}
	delete(s, ref)
	return nil
}
func Manager(t *testing.T, hosts map[string]inventory.Host, servers ...*Server) *transport.Manager {
	t.Helper()
	inv := &inventory.Inventory{Hosts: hosts, Defaults: inventory.Defaults{ConnectTimeout: "2s", CommandTimeout: "2s", Parallel: 2}}
	if err := inv.Validate(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "known_hosts")
	text := ""
	for _, s := range servers {
		text += knownhosts.Line([]string{knownhosts.Normalize(s.Address)}, s.Key.PublicKey()) + "\n"
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	m := &transport.Manager{Inventory: inv, Keys: &transport.HostKeys{Path: path}, Store: Store{"login": Password, "proxy": "proxy-user:proxy-pass", "sudo": "sudo-secret"}}
	t.Cleanup(m.Close)
	return m
}
func KeyFile(t *testing.T, s *Server, password string) string {
	t.Helper()
	var block *pem.Block
	var err error
	if password == "" {
		block, err = ssh.MarshalPrivateKey(s.Private, "fixture")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(s.Private, "fixture", []byte(password))
	}
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "key")
	if err = os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
