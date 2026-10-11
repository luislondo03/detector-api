package domain

import (
	"encoding/json"
	"errors"
	"time"
)

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

type Lifecycle string

const (
	LifecycleStarted Lifecycle = "STARTED"
	LifecycleUpdated Lifecycle = "UPDATED"
	LifecycleEnded   Lifecycle = "ENDED"
)

type Incident struct {
	EventID              string          `json:"eventId"`
	SessionID            string          `json:"sessionId"`
	Type                 string          `json:"type"`
	Lifecycle            Lifecycle       `json:"lifecycle"`
	Severity             Severity        `json:"severity"`
	StartedAt            time.Time       `json:"startedAt"`
	EndedAt              *time.Time      `json:"endedAt,omitempty"`
	DurationMilliseconds *int64          `json:"durationMs,omitempty"`
	Confidence           *float32        `json:"confidence,omitempty"`
	ObservationQuality   *float32        `json:"observationQuality,omitempty"`
	DetectorVersion      string          `json:"detectorVersion"`
	ModelVersion         string          `json:"modelVersion"`
	ThresholdProfileID   string          `json:"thresholdProfileId"`
	Details              json.RawMessage `json:"details"`
	AlertTriggered       bool            `json:"alertTriggered"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
	Evidence             []Evidence      `json:"evidence"`
}

type Evidence struct {
	ID         string    `json:"id"`
	IncidentID string    `json:"incidentId"`
	MediaType  string    `json:"mediaType"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (incident Incident) Validate() error {
	if incident.EventID == "" || incident.SessionID == "" || incident.Type == "" {
		return errors.New("eventId, sessionId and type are required")
	}
	if incident.StartedAt.IsZero() {
		return errors.New("startedAt is required")
	}
	if incident.Lifecycle != LifecycleStarted && incident.Lifecycle != LifecycleUpdated && incident.Lifecycle != LifecycleEnded {
		return errors.New("lifecycle is invalid")
	}
	if incident.Severity != SeverityInfo && incident.Severity != SeverityWarning && incident.Severity != SeverityCritical {
		return errors.New("severity is invalid")
	}
	if incident.Confidence != nil && (*incident.Confidence < 0 || *incident.Confidence > 1) {
		return errors.New("confidence must be between 0 and 1")
	}
	if incident.ObservationQuality != nil && (*incident.ObservationQuality < 0 || *incident.ObservationQuality > 1) {
		return errors.New("observationQuality must be between 0 and 1")
	}
	if len(incident.Details) > 0 && !json.Valid(incident.Details) {
		return errors.New("details must be valid JSON")
	}
	return nil
}
