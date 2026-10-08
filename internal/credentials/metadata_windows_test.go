// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"github.com/danieljoos/wincred"
	"testing"
)

func assertNativeMetadata(t *testing.T, ref string) {
	t.Helper()
	cred, err := wincred.GetGenericCredential(Service + ":" + ref)
	if err != nil {
		t.Fatal(err)
	}
	if cred.TargetName != Service+":"+ref || cred.Comment != CreatedBy || cred.UserName != ref {
		t.Fatal("credential prefix or creator comment missing")
	}
}
