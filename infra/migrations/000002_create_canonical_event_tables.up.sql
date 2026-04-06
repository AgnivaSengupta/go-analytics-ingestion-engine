CREATE TABLE IF NOT EXISTS raw_events (
    id BIGSERIAL PRIMARY KEY,
    site_id TEXT NOT NULL,
    event_id TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    source_type TEXT NOT NULL,
    api_key_id TEXT,
    request_ip INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_raw_events_site_received_at
    ON raw_events (site_id, received_at DESC);

CREATE INDEX IF NOT EXISTS idx_raw_events_event_id
    ON raw_events (event_id);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL,
    site_id TEXT NOT NULL,
    visitor_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    page_url TEXT NOT NULL,
    page_path TEXT NOT NULL,
    referrer TEXT,
    user_agent TEXT,
    ip_address INET,
    schema_version INT NOT NULL DEFAULT 1,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    context JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_events_site_occurred_at
    ON events (site_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_events_site_session_occurred_at
    ON events (site_id, session_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_events_site_visitor_occurred_at
    ON events (site_id, visitor_id, occurred_at DESC);
