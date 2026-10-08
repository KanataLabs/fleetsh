// SPDX-License-Identifier: GPL-3.0-only
//go:build !windows && !darwin && !linux

package credentials

import "github.com/zalando/go-keyring"

func setNative(ref, secret string) error {
	return keyring.Set(Service, ref, secret)
}
