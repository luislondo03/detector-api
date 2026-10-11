package domain

import (
	"testing"
	"time"
)

func TestIncidentValidateRejectsOutOfRangeConfidence(t *testing.T) {
	confidence := float32(1.1)
	incident := Incident{
		EventID: "f5c1a2f1-8d2e-4b10-a3f0-1fd970a005ee", SessionID: "session-1", Type: "POSSIBLE_PHONE_USE",
		Lifecycle: LifecycleStarted, Severity: SeverityWarning, StartedAt: time.Now(), Confidence: &confidence,
	}
	if err := incident.Validate(); err == nil {
		t.Fatal("expected invalid confidence to be rejected")
	}
}
