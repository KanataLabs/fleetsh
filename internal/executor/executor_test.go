// SPDX-License-Identifier: GPL-3.0-only
package executor

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
)

func TestSSHResultsTimeoutAndSudo(t *testing.T) {
	server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
		switch {
		case command == "fail":
			return testutil.Reply{Out: "output", Err: "failure", Code: 7}
		case command == "hang":
			<-ctx.Done()
			return testutil.Reply{}
		case command == "secret":
			return testutil.Reply{Out: testutil.Password}
		case strings.HasPrefix(command, "sudo -S"):
			data, _ := io.ReadAll(in)
			if string(data) != "sudo-secret\n" || strings.Contains(command, "sudo-secret") {
				return testutil.Reply{Code: 9}
			}
			return testutil.Reply{Out: "sudo ok"}
		default:
			return testutil.Reply{Out: "ready"}
		}
	})
	h := server.Host()
	h.SudoCredential = "sudo"
	m := testutil.Manager(t, map[string]inventory.Host{"a": h}, server)
	if err := m.Prepare(context.Background(), []string{"a"}, true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		command string
		options Options
		success bool
		exit    int
		kind    string
	}{
		{"uptime", Options{}, true, 0, ""}, {"fail", Options{}, false, 7, "command"}, {"hang", Options{Timeout: 40 * time.Millisecond}, false, -1, "timeout"},
		{"id", Options{Sudo: true}, true, 0, ""},
	} {
		r := Run(context.Background(), m, "a", tc.command, tc.options)
		if r.Success != tc.success || r.ExitCode != tc.exit || r.ErrorKind != tc.kind {
			t.Fatalf("%s: %+v", tc.command, r)
		}
	}
	r := Run(context.Background(), m, "a", "secret", Options{})
	if r.Stdout != "******" {
		t.Fatal("known secret not redacted")
	}
}
func TestBatchBoundsOrderAndSkips(t *testing.T) {
	inv := &inventory.Inventory{Hosts: map[string]inventory.Host{}}
	ids := []string{"a", "b", "c", "d", "console"}
	for _, id := range ids {
		inv.Hosts[id] = inventory.Host{Connection: "ssh"}
	}
	inv.Hosts["console"] = inventory.Host{Connection: "console-only"}
	var active, max atomic.Int32
	rows := Batch(context.Background(), inv, ids, 2, func(ctx context.Context, id string) Result {
		now := active.Add(1)
		for old := max.Load(); now > old; old = max.Load() {
			if max.CompareAndSwap(old, now) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		active.Add(-1)
		return Result{Host: id, Success: true, ExitCode: 0}
	})
	if max.Load() != 2 || !rows[4].Skipped {
		t.Fatalf("bound/skip failed: %d %+v", max.Load(), rows)
	}
	for i, r := range rows {
		if r.Host != ids[i] {
			t.Fatal("output ordering changed")
		}
	}
	if ExitCode(rows) != 0 || Summarize(rows).Summary.Skipped != 1 {
		t.Fatal("incorrect summary")
	}
	if ExitCode([]Result{{ErrorKind: "authentication"}, {ErrorKind: "command"}}) != 1 {
		t.Fatal("mixed failure precedence")
	}
}
func TestOutputCapAppliesToCopy(t *testing.T) {
	var b cappedBuffer
	_, err := io.Copy(&b, io.LimitReader(strings.NewReader(strings.Repeat("x", OutputLimit+4096)), int64(OutputLimit+4096)))
	if err != nil || b.Len() != OutputLimit || !b.truncated {
		t.Fatalf("output limit bypassed: %d %v %v", b.Len(), b.truncated, err)
	}
}
func TestShellQuotingAndSudoSecrets(t *testing.T) {
	command := "printf '%s' \"a'b\""
	wrapped, input := WrapSudo(command, "private")
	if strings.Contains(wrapped, "private") {
		t.Fatal("sudo password entered command arguments")
	}
	data, _ := io.ReadAll(input)
	if string(data) != "private\n" {
		t.Fatal("password was not sent through input")
	}
	if !strings.Contains(wrapped, "'\"'\"'") {
		t.Fatal("single quote was not escaped")
	}
}
