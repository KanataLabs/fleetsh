// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"github.com/KanataLabs/fleetsh/internal/inventory"
	"strings"
	"testing"
)

func TestReferencesFitSchemaAndSeparateRepeatedAliases(t *testing.T) {
	alias := strings.Repeat("a", 128)
	first, err := NewReference(alias)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewReference(alias)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !strings.HasPrefix(first, ReferencePrefix) || !inventory.ValidName(first) || !inventory.ValidName(second) {
		t.Fatal("reference collision, missing prefix, or oversized reference")
	}
}
