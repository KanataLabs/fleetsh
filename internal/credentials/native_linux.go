// SPDX-License-Identifier: GPL-3.0-only
package credentials

import ss "github.com/zalando/go-keyring/secret_service"

func setNative(ref, secret string) error {
	service, err := ss.NewSecretService()
	if err != nil {
		return err
	}
	session, err := service.OpenSession()
	if err != nil {
		return err
	}
	defer service.Close(session)
	collection := service.GetLoginCollection()
	if err := service.Unlock(collection.Path()); err != nil {
		return err
	}
	// Preserve the matching attributes used by go-keyring and older entries.
	return service.CreateItem(collection, Service+":"+ref+" — "+CreatedBy,
		map[string]string{"username": ref, "service": Service}, ss.NewSecret(session.Path(), secret))
}
