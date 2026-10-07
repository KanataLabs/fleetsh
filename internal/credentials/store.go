// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"context"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/zalando/go-keyring"
)

type Store interface {
	Get(string) (string, error)
	Set(string, string) error
	Delete(string) error
}
type Native struct{}

var ErrNotFound = errors.New("credential not found")

func (Native) Get(ref string) (string, error) {
	secret, err := keyring.Get("fleetsh", ref)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", errors.New("system credential store unavailable or locked")
	}
	return secret, nil
}
func (Native) Set(ref, secret string) error {
	if secret == "" || len(secret) > 2560 {
		return errors.New("secret must contain 1 to 2560 bytes")
	}
	if err := keyring.Set("fleetsh", ref, secret); err != nil {
		return errors.New("cannot save to system credential store (unavailable, locked, or secret too large)")
	}
	return nil
}
func (Native) Delete(ref string) error {
	if err := keyring.Delete("fleetsh", ref); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNotFound
		}
		return errors.New("cannot delete from system credential store")
	}
	return nil
}
func ReadSecret(in io.Reader, out io.Writer, label string) (string, error) {
	return ReadSecretContext(context.Background(), in, out, label)
}
func Redact(s string, secrets []string) string {
	secrets = append([]string(nil), secrets...)
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	for _, secret := range secrets {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "******")
		}
	}
	return s
}
