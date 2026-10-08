// SPDX-License-Identifier: GPL-3.0-only
// Package forwarding runs explicit TCP tunnels over an already verified SSH connection.
package forwarding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/KanataLabs/fleetsh/internal/inventory"
	"golang.org/x/crypto/ssh"
)

type Binding struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Listen      string `json:"listen"`
	Destination string `json:"destination,omitempty"`
}
type listener struct {
	spec inventory.Forward
	net.Listener
}

type Session struct {
	Bindings  []Binding
	ctx       context.Context
	cancel    context.CancelFunc
	client    *ssh.Client
	listeners []listener
	timeout   time.Duration
	report    func(error)
	slots     chan struct{}
	mu        sync.Mutex
	active    map[net.Conn]bool
	err       error
	once      sync.Once
	wg        sync.WaitGroup
	done      chan struct{}
}

// Start owns client after successful startup. A failed setup also closes client,
// rolling back all listeners. Profile addresses have been validated by inventory.
func Start(ctx context.Context, client *ssh.Client, specs []inventory.Forward, timeout time.Duration, report func(error)) (*Session, error) {
	if len(specs) == 0 || timeout <= 0 {
		return nil, errors.New("port forwards and a positive connection timeout are required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	s := &Session{ctx: runCtx, cancel: cancel, client: client, timeout: timeout, report: report, slots: make(chan struct{}, 128), active: map[net.Conn]bool{}, done: make(chan struct{})}
	setupCtx, setupCancel := context.WithTimeout(ctx, timeout)
	stopSetup := context.AfterFunc(setupCtx, func() { _ = client.Close() })
	defer func() { stopSetup(); setupCancel() }()
	for _, spec := range specs {
		var ln net.Listener
		var err error
		if err = setupCtx.Err(); err == nil {
			switch spec.Type {
			case "local", "dynamic":
				ln, err = (&net.ListenConfig{}).Listen(setupCtx, "tcp", spec.Listen)
			case "remote":
				ln, err = client.Listen("tcp", spec.Listen)
			default:
				err = errors.New("unsupported port forward type")
			}
		}
		if err != nil {
			s.shutdown(err)
			<-s.done
			return nil, fmt.Errorf("forward %s: cannot listen: %w", spec.Name, err)
		}
		s.listeners = append(s.listeners, listener{spec, ln})
		s.Bindings = append(s.Bindings, Binding{spec.Name, spec.Type, ln.Addr().String(), spec.Destination})
	}
	if !stopSetup() || setupCtx.Err() != nil {
		err := setupCtx.Err()
		if err == nil {
			err = context.Canceled
		}
		s.shutdown(err)
		<-s.done
		return nil, err
	}
	s.wg.Add(len(s.listeners))
	for _, ln := range s.listeners {
		go s.accept(ln)
	}
	context.AfterFunc(runCtx, func() { s.shutdown(runCtx.Err()) })
	go func() {
		err := client.Wait()
		if runCtx.Err() != nil {
			return
		}
		if err == nil {
			err = io.EOF
		}
		s.shutdown(fmt.Errorf("SSH connection closed: %w", err))
	}()
	return s, nil
}

func (s *Session) shutdown(err error) {
	s.once.Do(func() {
		s.mu.Lock()
		s.err = err
		s.cancel()
		_ = s.client.Close()
		for _, ln := range s.listeners {
			_ = ln.Close()
		}
		for conn := range s.active {
			_ = conn.Close()
		}
		s.mu.Unlock()
		go func() { s.wg.Wait(); close(s.done) }()
	})
}

func (s *Session) Close() { s.shutdown(nil); <-s.done }
func (s *Session) Wait() error {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}
func (s *Session) track(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx.Err() != nil {
		_ = conn.Close()
		return false
	}
	s.active[conn] = true
	return true
}
func (s *Session) release(conn net.Conn) {
	_ = conn.Close()
	s.mu.Lock()
	delete(s.active, conn)
	s.mu.Unlock()
}
func (s *Session) accept(ln listener) {
	defer s.wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.ctx.Err() == nil {
				s.shutdown(fmt.Errorf("forward %s: listener stopped: %w", ln.spec.Name, err))
			}
			return
		}
		select {
		case s.slots <- struct{}{}:
		default:
			_ = conn.Close()
			continue
		}
		if !s.track(conn) {
			<-s.slots
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { <-s.slots }()
			defer s.release(conn)
			if err := s.serve(conn, ln.spec); err != nil && s.ctx.Err() == nil && s.report != nil {
				s.report(fmt.Errorf("forward %s: %w", ln.spec.Name, err))
			}
		}()
	}
}
func (s *Session) serve(conn net.Conn, spec inventory.Forward) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()
	destination := spec.Destination
	if spec.Type == "dynamic" {
		_ = conn.SetDeadline(time.Now().Add(s.timeout))
		var err error
		destination, err = socksDestination(conn)
		if err != nil {
			return err
		}
	}
	var target net.Conn
	var err error
	if spec.Type == "remote" {
		target, err = (&net.Dialer{}).DialContext(ctx, "tcp", destination)
	} else {
		target, err = s.client.DialContext(ctx, "tcp", destination)
	}
	if err != nil {
		if spec.Type == "dynamic" {
			_ = socksReply(conn, 5)
		}
		return fmt.Errorf("destination connection failed: %w", err)
	}
	if !s.track(target) {
		return context.Canceled
	}
	defer s.release(target)
	if spec.Type == "dynamic" {
		if err := socksReply(conn, 0); err != nil {
			return err
		}
		_ = conn.SetDeadline(time.Time{})
	}
	relay(conn, target)
	return nil
}
func relay(a, b net.Conn) {
	done := make(chan struct{})
	copyTo := func(dst, src net.Conn) {
		_, err := io.Copy(dst, src)
		if closer, ok := dst.(interface{ CloseWrite() error }); ok && err == nil {
			_ = closer.CloseWrite()
		} else {
			_ = dst.Close()
			_ = src.Close()
		}
	}
	go func() { copyTo(b, a); close(done) }()
	copyTo(a, b)
	<-done
}
