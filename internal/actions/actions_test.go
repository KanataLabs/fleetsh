// SPDX-License-Identifier: GPL-3.0-only
package actions

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"github.com/KanataLabs/fleetsh/internal/testutil"
)

func TestUpdateCommands(t *testing.T) {
	command, err := UpdateCommand("ubuntu\n", true)
	if err != nil || !strings.Contains(command, "dist-upgrade") {
		t.Fatal("Debian dist upgrade missing")
	}
	for _, id := range []string{"debian", "rhel", "rocky", "almalinux", "fedora", "centos", "arch", "manjaro"} {
		if _, err := UpdateCommand(id, false); err != nil {
			t.Fatal(id, err)
		}
	}
	if _, err := UpdateCommand("unknown", false); err == nil {
		t.Fatal("unsupported OS accepted")
	}
}
func TestRebootNeedsNewIdentity(t *testing.T) {
	for _, changes := range []bool{true, false} {
		t.Run(fmt.Sprint(changes), func(t *testing.T) {
			var boot atomic.Int32
			boot.Store(1)
			server := testutil.StartSSH(t, func(ctx context.Context, command string, in io.Reader) testutil.Reply {
				switch command {
				case bootProbe:
					return testutil.Reply{Out: fmt.Sprintf("boot-%d\n", boot.Load())}
				case "reboot":
					if changes {
						boot.Store(2)
					}
					return testutil.Reply{Disconnect: true}
				case healthProbe:
					return testutil.Reply{Out: fmt.Sprintf("boot-%d\nuptime\n", boot.Load())}
				default:
					return testutil.Reply{Code: 1}
				}
			})
			m := testutil.Manager(t, map[string]inventory.Host{"a": server.Host()}, server)
			if err := m.Prepare(context.Background(), []string{"a"}, false); err != nil {
				t.Fatal(err)
			}
			wait := 5 * time.Second
			if !changes {
				wait = 150 * time.Millisecond
			}
			result := Reboot(context.Background(), m, "a", executor.Options{}, wait)
			if result.Success != changes {
				t.Fatalf("incorrect reboot result: %+v", result)
			}
		})
	}
}
