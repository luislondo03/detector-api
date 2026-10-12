package domain

import (
	"errors"
	"time"
)

// DeviceTelemetry is a point-in-time fleet location sent by an opted-in active session.
// DeviceID is an app-installation identifier, never a hardware identifier.
type DeviceTelemetry struct {
	ID                     string     `json:"id"`
	DeviceID               string     `json:"deviceId"`
	SessionID              string     `json:"sessionId"`
	CapturedAt             time.Time  `json:"capturedAt"`
	Latitude               float64    `json:"latitude"`
	Longitude              float64    `json:"longitude"`
	AccuracyMeters         *float32   `json:"accuracyMeters,omitempty"`
	SpeedKilometersPerHour *float32   `json:"speedKilometersPerHour,omitempty"`
	BearingDegrees         *float32   `json:"bearingDegrees,omitempty"`
	Provider               string     `json:"provider,omitempty"`
	BatteryPercent         *int       `json:"batteryPercent,omitempty"`
	Charging               *bool      `json:"charging,omitempty"`
	Manufacturer           string     `json:"manufacturer,omitempty"`
	Model                  string     `json:"model,omitempty"`
	AndroidVersion         string     `json:"androidVersion,omitempty"`
	AppVersion             string     `json:"appVersion,omitempty"`
	ReceivedAt             *time.Time `json:"receivedAt,omitempty"`
}

func (telemetry DeviceTelemetry) Validate() error {
	if telemetry.DeviceID == "" || telemetry.SessionID == "" {
		return errors.New("deviceId and sessionId are required")
	}
	if telemetry.CapturedAt.IsZero() {
		return errors.New("capturedAt is required")
	}
	if telemetry.Latitude < -90 || telemetry.Latitude > 90 || telemetry.Longitude < -180 || telemetry.Longitude > 180 {
		return errors.New("latitude or longitude is invalid")
	}
	if telemetry.AccuracyMeters != nil && *telemetry.AccuracyMeters < 0 {
		return errors.New("accuracyMeters must be non-negative")
	}
	if telemetry.SpeedKilometersPerHour != nil && *telemetry.SpeedKilometersPerHour < 0 {
		return errors.New("speedKilometersPerHour must be non-negative")
	}
	if telemetry.BatteryPercent != nil && (*telemetry.BatteryPercent < 0 || *telemetry.BatteryPercent > 100) {
		return errors.New("batteryPercent must be between 0 and 100")
	}
	return nil
}
