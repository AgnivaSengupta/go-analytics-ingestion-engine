CREATE TABLE IF NOT EXISTS visitors (
    id BIGSERIAL PRIMARY KEY,
    site_id TEXT NOT NULL,
    visitor_id TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    first_referrer TEXT,
    last_referrer TEXT,
    first_page_url TEXT,
    last_page_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, visitor_id)
);

CREATE INDEX IF NOT EXISTS idx_visitors_site_last_seen
    ON visitors (site_id, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS sessions (
    id BIGSERIAL PRIMARY KEY,
    site_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    visitor_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    landing_page_url TEXT,
    landing_page_path TEXT,
    landing_referrer TEXT,
    device_type TEXT,
    os_name TEXT,
    geo_country TEXT,
    source TEXT,
    medium TEXT,
    campaign TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_site_started_at
    ON sessions (site_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_site_visitor_started_at
    ON sessions (site_id, visitor_id, started_at DESC);

CREATE TABLE IF NOT EXISTS dead_letter_events (
    id BIGSERIAL PRIMARY KEY,
    site_id TEXT,
    event_id TEXT,
    payload JSONB NOT NULL,
    error_reason TEXT NOT NULL,
    attempt_count INT NOT NULL DEFAULT 1,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dead_letter_failed_at
    ON dead_letter_events (failed_at DESC);
