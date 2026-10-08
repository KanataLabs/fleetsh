// SPDX-License-Identifier: GPL-3.0-only
package credentials

import "github.com/danieljoos/wincred"

func setNative(ref, secret string) error {
	cred := wincred.NewGenericCredential(Service + ":" + ref)
	cred.UserName = ref
	cred.Comment = CreatedBy
	cred.CredentialBlob = []byte(secret)
	return cred.Write()
}
