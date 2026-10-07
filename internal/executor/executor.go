// SPDX-License-Identifier: GPL-3.0-only
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/KanataLabs/fleetsh/internal/credentials"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/transport"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Result struct {
	Host      string  `json:"host"`
	Success   bool    `json:"success"`
	Skipped   bool    `json:"skipped"`
	ExitCode  int     `json:"exit_code"`
	Stdout    string  `json:"stdout"`
	Stderr    string  `json:"stderr"`
	Error     string  `json:"error,omitempty"`
	ErrorKind string  `json:"error_kind,omitempty"`
	Duration  float64 `json:"duration_seconds"`
	Truncated bool    `json:"truncated,omitempty"`
	Started   bool    `json:"-"`
}
type Summary struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}
type Report struct {
	Results []Result `json:"results"`
	Summary Summary  `json:"summary"`
}

func Summarize(rows []Result) Report {
	s := Summary{}
	for _, r := range rows {
		if r.Skipped {
			s.Skipped++
		} else if r.Success {
			s.Success++
		} else {
			s.Failed++
		}
	}
	return Report{rows, s}
}
func ExitCode(rows []Result) int {
	code := 0
	for _, r := range rows {
		if r.Skipped || r.Success {
			continue
		}
		if r.ErrorKind == "connection" || r.ErrorKind == "authentication" || r.ErrorKind == "hostkey" || r.ErrorKind == "proxy" || r.ErrorKind == "connection_timeout" {
			if code == 0 {
				code = 3
			}
		} else {
			return 1
		}
	}
	return code
}
func Batch(ctx context.Context, inv *inventory.Inventory, ids []string, parallel int, run func(context.Context, string) Result) []Result {
	rows := make([]Result, len(ids))
	for i, id := range ids {
		rows[i] = Result{Host: id, ExitCode: -1, Error: "canceled before dispatch", ErrorKind: "canceled"}
		if inv.Hosts[id].Connection == "console-only" {
			rows[i] = Result{Host: id, Skipped: true, ExitCode: -1}
		}
	}
	if parallel < 1 {
		parallel = 1
	}
	if parallel > len(ids) {
		parallel = len(ids)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				id := ids[i]
				if inv.Hosts[id].Connection == "console-only" {
					rows[i] = Result{Host: id, Skipped: true, ExitCode: -1}
					continue
				}
				rows[i] = run(ctx, id)
			}
		}()
	}
feed:
	for i := range ids {
		if rows[i].Skipped {
			continue
		}
		select {
		case jobs <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()
	return rows
}

type Options struct {
	Timeout time.Duration
	Sudo    bool
}
type cappedBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

const OutputLimit = 4 * 1024 * 1024

func (b *cappedBuffer) Len() int       { return b.buf.Len() }
func (b *cappedBuffer) String() string { return b.buf.String() }
func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := OutputLimit - b.Len()
	if len(p) > remaining {
		b.truncated = true
		p = p[:remaining]
	}
	_, err := b.buf.Write(p)
	return n, err
}
func Quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func WrapSudo(command, password string) (string, io.Reader) {
	if password != "" {
		return "sudo -S -p '' -- sh -c " + Quote(command), strings.NewReader(password + "\n")
	}
	return "sudo -n -- sh -c " + Quote(command), nil
}
func Classify(err error) string {
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "connection_timeout"
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return "connection_timeout"
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "host key") || strings.Contains(s, "knownhosts"):
		return "hostkey"
	case strings.Contains(s, "authenticate"):
		return "authentication"
	case strings.Contains(s, "proxy"):
		return "proxy"
	default:
		return "connection"
	}
}
func Run(ctx context.Context, m *transport.Manager, id, command string, options Options) Result {
	return run(ctx, m, id, command, options, nil)
}

// Probe returns unredacted machine output for action decisions. Do not print the raw value.
func Probe(ctx context.Context, m *transport.Manager, id, command string, options Options) (string, Result) {
	var raw string
	r := run(ctx, m, id, command, options, &raw)
	return raw, r
}
func run(ctx context.Context, m *transport.Manager, id, command string, options Options, raw *string) (r Result) {
	started := time.Now()
	r = Result{Host: id, ExitCode: -1}
	defer func() { r.Duration = time.Since(started).Seconds(); r.Error = credentials.Redact(r.Error, m.Secrets) }()
	c, err := m.Dial(ctx, id)
	if err != nil {
		r.Error = err.Error()
		r.ErrorKind = Classify(err)
		return
	}
	defer c.Close()
	timeout := options.Timeout
	if timeout == 0 {
		timeout, _ = time.ParseDuration(m.Inventory.Defaults.CommandTimeout)
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stop := context.AfterFunc(runCtx, func() { _ = c.Close() })
	defer stop()
	session, err := c.NewSession()
	if err != nil {
		r.Error = err.Error()
		r.ErrorKind = "connection"
		return
	}
	defer session.Close()
	var stdout, stderr cappedBuffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	var input io.Reader
	var inputPipe io.WriteCloser
	if options.Sudo {
		command, input = WrapSudo(command, m.Sudo[id])
		if input != nil {
			inputPipe, err = session.StdinPipe()
			if err != nil {
				r.Error = err.Error()
				r.ErrorKind = "connection"
				return
			}
		}
	}
	err = session.Start(command)
	if err == nil {
		r.Started = true
		if inputPipe != nil {
			// The remote exit status is authoritative even if it exits before reading stdin.
			_, _ = io.Copy(inputPipe, input)
			_ = inputPipe.Close()
		}
		err = session.Wait()
	}
	if raw != nil {
		*raw = stdout.String()
	}
	r.Stdout = credentials.Redact(stdout.String(), m.Secrets)
	r.Stderr = credentials.Redact(stderr.String(), m.Secrets)
	r.Truncated = stdout.truncated || stderr.truncated
	if runCtx.Err() != nil {
		r.Error = runCtx.Err().Error()
		r.ErrorKind = "timeout"
		return
	}
	if err != nil {
		r.Error = err.Error()
		var exit *ssh.ExitError
		if errors.As(err, &exit) {
			r.ExitCode = exit.ExitStatus()
			r.ErrorKind = "command"
		} else {
			r.ErrorKind = "connection"
		}
		return
	}
	r.ExitCode = 0
	r.Success = true
	return
}
func Interactive(ctx context.Context, m *transport.Manager, id string, in io.Reader, out, errOut io.Writer) error {
	c, err := m.Dial(ctx, id)
	if err != nil {
		return err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = in
	session.Stdout = out
	session.Stderr = errOut
	var resizeStop context.CancelFunc
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		width, height, err := term.GetSize(int(f.Fd()))
		if err != nil {
			width = 80
			height = 24
		}
		terminal := os.Getenv("TERM")
		if terminal == "" {
			terminal = "xterm-256color"
		}
		if err = session.RequestPty(terminal, height, width, ssh.TerminalModes{ssh.ECHO: 1}); err != nil {
			return fmt.Errorf("request PTY: %w", err)
		}
		state, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return err
		}
		defer term.Restore(int(f.Fd()), state)
		resizeCtx, cancel := context.WithCancel(ctx)
		resizeStop = cancel
		go func() {
			timer := time.NewTicker(time.Second)
			defer timer.Stop()
			for {
				select {
				case <-resizeCtx.Done():
					return
				case <-timer.C:
					w, h, e := term.GetSize(int(f.Fd()))
					if e == nil && (w != width || h != height) {
						_ = session.WindowChange(h, w)
						width = w
						height = h
					}
				}
			}
		}()
	}
	if resizeStop != nil {
		defer resizeStop()
	}
	if err = session.Shell(); err != nil {
		return err
	}
	return session.Wait()
}
