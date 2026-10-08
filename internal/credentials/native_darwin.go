// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

func setNative(ref, secret string) error {
	// Match go-keyring's encoding so existing Get/Delete remain compatible.
	encoded := "go-keyring-base64:" + base64.StdEncoding.EncodeToString([]byte(secret))
	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -l %s -j %s -w %s\n",
		quoteKeychain(Service), quoteKeychain(ref), quoteKeychain(Service+":"+ref),
		quoteKeychain(CreatedBy), quoteKeychain(encoded))
	cmd := exec.Command("/usr/bin/security", "-i")
	cmd.Stdin = strings.NewReader(command)
	// Keep the secret in stdin, and discard utility diagnostics that could echo it.
	return cmd.Run()
}

func quoteKeychain(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
