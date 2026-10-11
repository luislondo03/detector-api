CREATE TABLE IF NOT EXISTS incidents (
    event_id UUID PRIMARY KEY,
    session_id TEXT NOT NULL,
    type TEXT NOT NULL,
    lifecycle TEXT NOT NULL,
    severity TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NULL,
    duration_ms BIGINT NULL,
    confidence REAL NULL CHECK (confidence >= 0 AND confidence <= 1),
    observation_quality REAL NULL CHECK (observation_quality >= 0 AND observation_quality <= 1),
    detector_version TEXT NOT NULL DEFAULT '',
    model_version TEXT NOT NULL DEFAULT '',
    threshold_profile_id TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    alert_triggered BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS incidents_started_at_idx ON incidents (started_at DESC);
CREATE INDEX IF NOT EXISTS incidents_type_idx ON incidents (type);
CREATE TABLE IF NOT EXISTS incident_evidence (
    id UUID PRIMARY KEY,
    incident_id UUID NOT NULL REFERENCES incidents(event_id) ON DELETE CASCADE,
    media_type TEXT NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS incident_evidence_incident_id_idx ON incident_evidence (incident_id);
