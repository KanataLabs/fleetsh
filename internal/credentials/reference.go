// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

const (
	Service         = "fleetsh"
	CreatedBy       = "Created by fleetsh (KanataLabs). Managed VPS credential."
	ReferencePrefix = "fleetsh-ssh-"
)

// NewReference gives each saved host password its own entry, including across inventories.
func NewReference(alias string) (string, error) {
	data := make([]byte, 8)
	if _, err := rand.Read(data); err != nil {
		return "", errors.New("cannot generate credential reference")
	}
	maxAlias := 128 - len(ReferencePrefix) - 1 - hex.EncodedLen(len(data))
	if len(alias) > maxAlias {
		alias = alias[:maxAlias]
	}
	return ReferencePrefix + alias + "-" + hex.EncodeToString(data), nil
}

func ValidateSecret(secret string) error {
	if secret == "" || len(secret) > 2560 {
		return errors.New("secret must contain 1 to 2560 bytes")
	}
	return nil
}
