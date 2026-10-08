// SPDX-License-Identifier: GPL-3.0-only
package actions

import (
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStatsAgainstLinuxProcAndUtilities(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the resource snapshot is executed on Linux VPSs")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "sh", "-c", Stats).CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot failed: %v %s", err, output)
	}
	text := string(output)
	for _, pattern := range []string{"(?m)^USER: .+", "(?m)^CPU: [0-9]+\\.[0-9]% busy", "(?m)^MEMORY: [0-9]+ / [0-9]+ MiB", "(?m)^SWAP: [0-9]+ / [0-9]+ MiB", "(?m)^DISK:"} {
		if !regexp.MustCompile(pattern).MatchString(text) {
			t.Fatalf("missing %q in %s", pattern, text)
		}
	}
	if !strings.Contains(text, "Filesystem") {
		t.Fatal("df output missing")
	}
}
