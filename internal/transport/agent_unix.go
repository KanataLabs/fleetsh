//go:build !windows

// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"errors"
	"net"
	"os"
)

func dialAgent(ctx context.Context) (net.Conn, error) {
	path := os.Getenv("SSH_AUTH_SOCK")
	if path == "" {
		return nil, errors.New("SSH_AUTH_SOCK is not set")
	}
	return (&net.Dialer{}).DialContext(ctx, "unix", path)
}
