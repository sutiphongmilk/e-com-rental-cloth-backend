-- +goose Up
-- Enable btree_gist so rental booking tables can enforce database-level non-overlapping
-- date ranges: EXCLUDE USING gist (garment_unit_id WITH =, booked_period WITH &&)
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Opaque server-side sessions (only SHA-256 hash of token is stored; raw token lives in HttpOnly cookie)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS sessions;
