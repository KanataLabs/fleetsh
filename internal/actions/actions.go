// SPDX-License-Identifier: GPL-3.0-only
package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KanataLabs/fleetsh/internal/executor"
	"github.com/KanataLabs/fleetsh/internal/transport"
)

const DetectOS = `if [ -r /etc/os-release ]; then . /etc/os-release; printf '%s\n' "$ID"; else exit 1; fi`

func UpdateCommand(osID string, dist bool) (string, error) {
	switch strings.TrimSpace(osID) {
	case "ubuntu", "debian":
		upgrade := "upgrade"
		if dist {
			upgrade = "dist-upgrade"
		}
		return "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get " + upgrade + " -y", nil
	case "rhel", "rocky", "almalinux", "fedora":
		return "dnf upgrade -y", nil
	case "centos":
		return "if command -v dnf >/dev/null 2>&1; then dnf upgrade -y; else yum update -y; fi", nil
	case "arch", "manjaro":
		return "pacman -Syu --noconfirm", nil
	default:
		return "", fmt.Errorf("unsupported Linux distribution")
	}
}

const bootProbe = "cat /proc/sys/kernel/random/boot_id"
const healthProbe = "cat /proc/sys/kernel/random/boot_id && uptime"

// Reboot only succeeds when a new boot identity and a health probe are observed.
func Reboot(ctx context.Context, m *transport.Manager, id string, options executor.Options, wait time.Duration) executor.Result {
	start := time.Now()
	probe := executor.Run(ctx, m, id, bootProbe, executor.Options{Timeout: 10 * time.Second})
	if !probe.Success {
		return probe
	}
	original := strings.TrimSpace(probe.Stdout)
	if original == "" {
		probe.Success = false
		probe.ErrorKind = "command"
		probe.Error = "cannot read boot identity"
		return probe
	}
	if options.Timeout == 0 {
		options.Timeout = 10 * time.Second
	}
	r := executor.Run(ctx, m, id, "reboot", options)
	if !r.Started || (!r.Success && r.ErrorKind != "connection" && r.ErrorKind != "timeout") {
		return r
	}
	waitCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-waitCtx.Done():
			return executor.Result{Host: id, ExitCode: -1, ErrorKind: "timeout", Error: "reboot did not produce a new boot identity and health probe before deadline", Duration: time.Since(start).Seconds()}
		case <-timer.C:
			health := executor.Run(waitCtx, m, id, healthProbe, executor.Options{Timeout: 10 * time.Second})
			if health.ErrorKind == "hostkey" {
				return health
			}
			newBoot, _, _ := strings.Cut(strings.TrimSpace(health.Stdout), "\n")
			if health.Success && newBoot != "" && newBoot != original {
				health.Stdout = "Reboot verified; new boot identity: " + newBoot + "\n"
				health.Duration = time.Since(start).Seconds()
				return health
			}
		}
	}
}
