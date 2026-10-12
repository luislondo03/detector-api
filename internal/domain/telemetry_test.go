package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDeviceTelemetryValidateRejectsInvalidCoordinates(t *testing.T) {
	telemetry := DeviceTelemetry{
		DeviceID: "installation-1", SessionID: "session-1", CapturedAt: time.Now(), Latitude: 91, Longitude: -74,
	}
	if err := telemetry.Validate(); err == nil || !strings.Contains(err.Error(), "latitude") {
		t.Fatalf("expected coordinate validation error, got %v", err)
	}
}

func TestDeviceTelemetryValidateAcceptsFleetPoint(t *testing.T) {
	telemetry := DeviceTelemetry{
		DeviceID: "installation-1", SessionID: "session-1", CapturedAt: time.Now(), Latitude: 4.60971, Longitude: -74.08175,
	}
	if err := telemetry.Validate(); err != nil {
		t.Fatalf("expected valid telemetry, got %v", err)
	}
}
