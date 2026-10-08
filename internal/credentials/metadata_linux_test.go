// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	ss "github.com/zalando/go-keyring/secret_service"
	"strings"
	"testing"
)

func assertNativeMetadata(t *testing.T, ref string) {
	t.Helper()
	service, err := ss.NewSecretService()
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.SearchItems(service.GetLoginCollection(), map[string]string{"username": ref, "service": Service})
	if err != nil || len(items) != 1 {
		t.Fatal("missing or duplicate credential item")
	}
	label, err := service.Object("org.freedesktop.secrets", items[0]).GetProperty("org.freedesktop.Secret.Item.Label")
	if err != nil {
		t.Fatal(err)
	}
	value, ok := label.Value().(string)
	if !ok || !strings.Contains(value, Service+":"+ref) || !strings.Contains(value, CreatedBy) {
		t.Fatal("credential label or creator note missing")
	}
}
