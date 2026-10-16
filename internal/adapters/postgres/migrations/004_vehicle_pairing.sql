-- Durable vehicle pairing codes let the dashboard show a stable QR without
-- creating a one-time enrollment every time a terminal needs to be linked.
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS pairing_code TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS vehicles_pairing_code_uidx ON vehicles (pairing_code) WHERE pairing_code IS NOT NULL;
