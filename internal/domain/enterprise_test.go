package domain

import (
	"testing"
	"time"
)

func TestDeviceEventRequiresMonotonicRevisionAndMessageIdentity(t *testing.T) {
	event := DeviceEvent{EventID: "event-1", SessionID: "session-1", Type: "NODDING_OFF", Lifecycle: LifecycleStarted, Severity: SeverityCritical, OccurredAt: time.Now(), StartedAt: time.Now()}
	if err := event.Validate(); err == nil { t.Fatal("expected message identity validation error") }
	event.MessageID, event.BootID, event.EventRevision = "message-1", "boot-1", 1
	if err := event.Validate(); err != nil { t.Fatalf("expected valid enterprise event, got %v", err) }
}

func TestTelemetryMessageRejectsInvalidCoordinates(t *testing.T) {
	value := TelemetryMessage{SyncMessage: SyncMessage{MessageID: "message-1", BootID: "boot-1"}, SessionID: "session-1", OccurredAt: time.Now(), Latitude: 91, Longitude: 0}
	if err := value.Validate(); err == nil { t.Fatal("expected invalid coordinate error") }
}
