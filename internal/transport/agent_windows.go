//go:build windows

// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"net"
	"os"
	"strings"

	"github.com/Microsoft/go-winio"
)

func dialAgent(ctx context.Context) (net.Conn, error) {
	path := os.Getenv("SSH_AUTH_SOCK")
	if path == "" {
		path = `\\.\pipe\openssh-ssh-agent`
	}
	if strings.HasPrefix(path, `\\.\pipe\`) {
		return winio.DialPipeContext(ctx, path)
	}
	return (&net.Dialer{}).DialContext(ctx, "unix", path)
}
