CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS urls (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short      VARCHAR(10) NOT NULL UNIQUE,
    original   TEXT NOT NULL,
    clicks     BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_urls_short ON urls(short);
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at DESC);

CREATE TABLE IF NOT EXISTS clicks (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_id   UUID NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    ip         INET,
    user_agent TEXT,
    referer    TEXT,
    country    VARCHAR(2),
    timestamp  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clicks_short_id ON clicks(short_id);
CREATE INDEX IF NOT EXISTS idx_clicks_timestamp ON clicks(timestamp DESC);
