CREATE TABLE IF NOT EXISTS analytics_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100),
    post_id VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    user_id VARCHAR(100),
    author_id VARCHAR(100),
    referrer TEXT,
    user_agent TEXT,
    ip_address INET,
    geo_country VARCHAR(16),
    geo_region VARCHAR(100),
    device_type VARCHAR(20),
    os_name VARCHAR(50),
    scroll_depth_percent INT,
    time_spent_sec INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lookup
    ON analytics_events (post_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_author_lookup
    ON analytics_events (author_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_event_id
    ON analytics_events (event_id);

CREATE TABLE IF NOT EXISTS hourly_stats (
    time_bucket TIMESTAMPTZ NOT NULL,
    post_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    views BIGINT DEFAULT 0,
    visitors BIGINT DEFAULT 0,
    time_spent_sec BIGINT DEFAULT 0,
    PRIMARY KEY (post_id, time_bucket)
);

CREATE INDEX IF NOT EXISTS idx_hourly_time
    ON hourly_stats (post_id, time_bucket DESC);

CREATE TABLE IF NOT EXISTS daily_stats (
    day DATE NOT NULL,
    post_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    views BIGINT DEFAULT 0,
    visitors BIGINT DEFAULT 0,
    time_spent_sec BIGINT DEFAULT 0,
    PRIMARY KEY (post_id, day)
);

CREATE INDEX IF NOT EXISTS idx_daily_time
    ON daily_stats (post_id, day DESC);

CREATE TABLE IF NOT EXISTS monthly_stats (
    month DATE NOT NULL,
    post_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    views BIGINT DEFAULT 0,
    visitors BIGINT DEFAULT 0,
    time_spent_sec BIGINT DEFAULT 0,
    PRIMARY KEY (post_id, month)
);

CREATE INDEX IF NOT EXISTS idx_monthly_time
    ON monthly_stats (post_id, month DESC);

CREATE TABLE IF NOT EXISTS yearly_stats (
    year DATE NOT NULL,
    post_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    views BIGINT DEFAULT 0,
    visitors BIGINT DEFAULT 0,
    time_spent_sec BIGINT DEFAULT 0,
    PRIMARY KEY (post_id, year)
);

CREATE INDEX IF NOT EXISTS idx_yearly_time
    ON yearly_stats (post_id, year DESC);
