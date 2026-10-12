CREATE TABLE IF NOT EXISTS device_telemetry (
    id UUID PRIMARY KEY,
    device_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    accuracy_meters REAL NULL CHECK (accuracy_meters >= 0),
    speed_kilometers_per_hour REAL NULL CHECK (speed_kilometers_per_hour >= 0),
    bearing_degrees REAL NULL,
    provider TEXT NOT NULL DEFAULT '',
    battery_percent SMALLINT NULL CHECK (battery_percent >= 0 AND battery_percent <= 100),
    charging BOOLEAN NULL,
    manufacturer TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    android_version TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS device_telemetry_device_captured_at_idx ON device_telemetry (device_id, captured_at DESC);
