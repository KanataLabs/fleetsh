// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPromptRejectsEchoedInput(t *testing.T) {
	if _, err := ReadSecret(strings.NewReader("sensitive\n"), io.Discard, "Secret"); err == nil {
		t.Fatal("non-terminal input accepted")
	}
	if got := Redact("token=private and password=private", []string{"private"}); strings.Contains(got, "private") {
		t.Fatal("redaction failed")
	}
}
func TestNativeRoundTrip(t *testing.T) {
	if os.Getenv("FLEETSH_TEST_KEYRING") != "1" {
		t.Skip("native keyring integration requires an unlocked OS store")
	}
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	ref := "fleetsh-test-" + hex.EncodeToString(data)
	store := Native{}
	if err := store.Set(ref, "fixture-secret"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(ref) })
	if value, err := store.Get(ref); err != nil || value != "fixture-secret" {
		t.Fatalf("native get failed: %v", err)
	}
	if err := store.Set(ref, "replacement-fixture"); err != nil {
		t.Fatal(err)
	}
	if value, err := store.Get(ref); err != nil || value != "replacement-fixture" {
		t.Fatalf("native replacement failed: %v", err)
	}
	if err := store.Delete(ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("removed credential still exists: %v", err)
	}
}
