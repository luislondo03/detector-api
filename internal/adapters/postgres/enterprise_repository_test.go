package postgres

import (
	"strings"
	"testing"
)

func TestFleetListQueryNormalizesNullableJoinedStrings(t *testing.T) {
	for _, expected := range []string{
		"COALESCE(d.id::text, '')",
		"COALESCE(d.label, '')",
		"COALESCE(h.operation_status, '')",
	} {
		if !strings.Contains(fleetListQuery, expected) {
			t.Fatalf("fleet list query must normalize optional field %q", expected)
		}
	}
}
