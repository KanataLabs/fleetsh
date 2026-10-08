// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"os/exec"
	"strings"
	"testing"
)

func assertNativeMetadata(t *testing.T, ref string) {
	t.Helper()
	out, err := exec.Command("/usr/bin/security", "find-generic-password", "-s", Service, "-a", ref).CombinedOutput()
	if err != nil {
		t.Fatal("cannot inspect keychain metadata")
	}
	if !strings.Contains(string(out), Service+":"+ref) || !strings.Contains(string(out), CreatedBy) {
		t.Fatal("keychain label or creator comment missing")
	}
}
