package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIncidentJSONSerializesEmptyEvidenceAsArray(t *testing.T) {
	payload, err := json.Marshal(Incident{Evidence: make([]Evidence, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"evidence":[]`) {
		t.Fatalf("expected evidence array, got %s", payload)
	}
}
