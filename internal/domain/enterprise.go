package domain

import (
	"errors"
	"time"
)

type Company struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"createdAt"`
}

type Vehicle struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"companyId"`
	Plate        string    `json:"plate"`
	InternalCode string    `json:"internalCode"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Device struct {
	ID           string     `json:"id"`
	CompanyID    string     `json:"companyId"`
	Label        string     `json:"label"`
	Hardware     string     `json:"hardware"`
	AppVersion   string     `json:"appVersion"`
	ModelVersion string     `json:"modelVersion"`
	EnrolledAt   *time.Time `json:"enrolledAt,omitempty"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
}

type EnrollmentCode struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"companyId"`
	VehicleID string    `json:"vehicleId"`
	DeviceID  string    `json:"deviceId"`
	Code      string    `json:"code,omitempty"`
	QRPayload string    `json:"qrPayload,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type DevicePrincipal struct {
	DeviceID  string
	CompanyID string
	VehicleID string
}

type EnrollmentResult struct {
	DeviceID              string `json:"deviceId"`
	CompanyID             string `json:"companyId"`
	VehicleID             string `json:"vehicleId"`
	VehiclePlate          string `json:"vehiclePlate"`
	DeviceToken           string `json:"deviceToken"`
	TelemetryEverySeconds int    `json:"telemetryEverySeconds"`
}

type SyncMessage struct {
	MessageID string `json:"messageId"`
	BootID    string `json:"bootId"`
}

type DeviceEvent struct {
	SyncMessage
	EventID              string     `json:"eventId"`
	EventRevision        int        `json:"eventRevision"`
	SessionID            string     `json:"sessionId"`
	Type                 string     `json:"type"`
	Lifecycle            Lifecycle  `json:"lifecycle"`
	Severity             Severity   `json:"severity"`
	OccurredAt           time.Time  `json:"occurredAt"`
	StartedAt            time.Time  `json:"startedAt"`
	EndedAt              *time.Time `json:"endedAt,omitempty"`
	DurationMilliseconds *int64     `json:"durationMs,omitempty"`
	Confidence           *float32   `json:"confidence,omitempty"`
	ObservationQuality   *float32   `json:"observationQuality,omitempty"`
	DetectorVersion      string     `json:"detectorVersion"`
	ModelVersion         string     `json:"modelVersion"`
}

type MonitoringSession struct {
	SyncMessage
	SessionID   string     `json:"sessionId"`
	StartedAt   time.Time  `json:"startedAt"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	CloseReason string     `json:"closeReason,omitempty"`
	OccurredAt  time.Time  `json:"occurredAt"`
}

type CoverageInterval struct {
	SyncMessage
	SessionID  string    `json:"sessionId"`
	StartedAt  time.Time `json:"startedAt"`
	EndedAt    time.Time `json:"endedAt"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurredAt"`
}

type DeviceHeartbeat struct {
	SyncMessage
	OccurredAt       time.Time `json:"occurredAt"`
	OperationStatus  string    `json:"operationStatus"`
	QualityStatus    string    `json:"qualityStatus"`
	CameraStatus     string    `json:"cameraStatus"`
	AppVersion       string    `json:"appVersion"`
	ModelVersion     string    `json:"modelVersion"`
	PendingMessages  int       `json:"pendingMessages"`
	StorageAvailable *int64    `json:"storageAvailableBytes,omitempty"`
}

type TelemetryMessage struct {
	SyncMessage
	SessionID              string    `json:"sessionId"`
	OccurredAt             time.Time `json:"occurredAt"`
	Latitude               float64   `json:"latitude"`
	Longitude              float64   `json:"longitude"`
	AccuracyMeters         *float32  `json:"accuracyMeters,omitempty"`
	SpeedKilometersPerHour *float32  `json:"speedKilometersPerHour,omitempty"`
	BearingDegrees         *float32  `json:"bearingDegrees,omitempty"`
	Provider               string    `json:"provider,omitempty"`
	BatteryPercent         *int      `json:"batteryPercent,omitempty"`
	Charging               *bool     `json:"charging,omitempty"`
	Manufacturer           string    `json:"manufacturer,omitempty"`
	Model                  string    `json:"model,omitempty"`
	AndroidVersion         string    `json:"androidVersion,omitempty"`
	AppVersion             string    `json:"appVersion,omitempty"`
}

type BatchAck struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Code      string `json:"code,omitempty"`
}

func (message SyncMessage) Validate() error {
	if message.MessageID == "" || message.BootID == "" {
		return errors.New("messageId and bootId are required")
	}
	return nil
}

func (event DeviceEvent) Validate() error {
	if err := event.SyncMessage.Validate(); err != nil {
		return err
	}
	if event.EventID == "" || event.SessionID == "" || event.Type == "" || event.OccurredAt.IsZero() || event.StartedAt.IsZero() {
		return errors.New("eventId, sessionId, type, occurredAt and startedAt are required")
	}
	if event.EventRevision < 1 {
		return errors.New("eventRevision must be positive")
	}
	if event.Lifecycle != LifecycleStarted && event.Lifecycle != LifecycleUpdated && event.Lifecycle != LifecycleEnded {
		return errors.New("lifecycle is invalid")
	}
	if event.Severity != SeverityInfo && event.Severity != SeverityWarning && event.Severity != SeverityCritical {
		return errors.New("severity is invalid")
	}
	return nil
}

func (telemetry TelemetryMessage) Validate() error {
	if err := telemetry.SyncMessage.Validate(); err != nil {
		return err
	}
	if telemetry.SessionID == "" || telemetry.OccurredAt.IsZero() {
		return errors.New("sessionId and occurredAt are required")
	}
	if telemetry.Latitude < -90 || telemetry.Latitude > 90 || telemetry.Longitude < -180 || telemetry.Longitude > 180 {
		return errors.New("latitude or longitude is invalid")
	}
	return nil
}
