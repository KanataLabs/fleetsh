// SPDX-License-Identifier: GPL-3.0-only
//go:build !windows && !darwin && !linux

package credentials

import "testing"

func assertNativeMetadata(t *testing.T, ref string) {}
