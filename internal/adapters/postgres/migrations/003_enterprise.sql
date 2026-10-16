CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY, name TEXT NOT NULL UNIQUE, timezone TEXT NOT NULL DEFAULT 'America/Bogota', created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS vehicles (
    id UUID PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), plate TEXT NOT NULL, internal_code TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'ACTIVE', created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(company_id, plate)
);
CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), label TEXT NOT NULL DEFAULT '', hardware TEXT NOT NULL DEFAULT '', app_version TEXT NOT NULL DEFAULT '', model_version TEXT NOT NULL DEFAULT '', credential_hash TEXT NULL, enrolled_at TIMESTAMPTZ NULL, revoked_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS device_vehicle_assignments (
    id UUID PRIMARY KEY, device_id UUID NOT NULL REFERENCES devices(id), vehicle_id UUID NOT NULL REFERENCES vehicles(id), starts_at TIMESTAMPTZ NOT NULL, ends_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS active_device_vehicle_assignment_idx ON device_vehicle_assignments(device_id) WHERE ends_at IS NULL;
CREATE TABLE IF NOT EXISTS enrollment_codes (
    id UUID PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), vehicle_id UUID NOT NULL REFERENCES vehicles(id), device_id UUID NOT NULL REFERENCES devices(id), code_hash TEXT NOT NULL UNIQUE, expires_at TIMESTAMPTZ NOT NULL, used_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS monitoring_sessions (
    session_id TEXT PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), device_id UUID NOT NULL REFERENCES devices(id), vehicle_id UUID NOT NULL REFERENCES vehicles(id), started_at TIMESTAMPTZ NOT NULL, ended_at TIMESTAMPTZ NULL, close_reason TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS detection_events (
    event_id TEXT PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), device_id UUID NOT NULL REFERENCES devices(id), vehicle_id UUID NOT NULL REFERENCES vehicles(id), session_id TEXT NOT NULL, revision INTEGER NOT NULL, type TEXT NOT NULL, lifecycle TEXT NOT NULL, severity TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, started_at TIMESTAMPTZ NOT NULL, ended_at TIMESTAMPTZ NULL, duration_ms BIGINT NULL, confidence REAL NULL, observation_quality REAL NULL, detector_version TEXT NOT NULL DEFAULT '', model_version TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS coverage_intervals (
    id UUID PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), device_id UUID NOT NULL REFERENCES devices(id), vehicle_id UUID NOT NULL REFERENCES vehicles(id), session_id TEXT NOT NULL, started_at TIMESTAMPTZ NOT NULL, ended_at TIMESTAMPTZ NOT NULL, status TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS device_heartbeats (
    id UUID PRIMARY KEY, company_id UUID NOT NULL REFERENCES companies(id), device_id UUID NOT NULL REFERENCES devices(id), vehicle_id UUID NOT NULL REFERENCES vehicles(id), occurred_at TIMESTAMPTZ NOT NULL, operation_status TEXT NOT NULL, quality_status TEXT NOT NULL, camera_status TEXT NOT NULL, app_version TEXT NOT NULL DEFAULT '', model_version TEXT NOT NULL DEFAULT '', pending_messages INTEGER NOT NULL DEFAULT 0, storage_available_bytes BIGINT NULL, received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS received_device_messages (
    device_id UUID NOT NULL REFERENCES devices(id), message_id TEXT NOT NULL, received_at TIMESTAMPTZ NOT NULL DEFAULT now(), PRIMARY KEY(device_id, message_id)
);
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY, company_id UUID NULL REFERENCES companies(id), device_id UUID NULL REFERENCES devices(id), action TEXT NOT NULL, resource_type TEXT NOT NULL, resource_id TEXT NOT NULL, details JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE device_telemetry ADD COLUMN IF NOT EXISTS company_id UUID NULL REFERENCES companies(id);
ALTER TABLE device_telemetry ADD COLUMN IF NOT EXISTS vehicle_id UUID NULL REFERENCES vehicles(id);
ALTER TABLE device_telemetry ADD COLUMN IF NOT EXISTS enterprise_device_id UUID NULL REFERENCES devices(id);
CREATE INDEX IF NOT EXISTS device_telemetry_enterprise_device_captured_idx ON device_telemetry(enterprise_device_id, captured_at DESC);
CREATE INDEX IF NOT EXISTS detection_events_started_at_idx ON detection_events(started_at DESC);
CREATE INDEX IF NOT EXISTS detection_events_vehicle_idx ON detection_events(vehicle_id);
