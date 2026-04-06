CREATE TABLE IF NOT EXISTS agg_site_hourly (
    time_bucket TIMESTAMPTZ NOT NULL,
    site_id TEXT NOT NULL,
    events BIGINT NOT NULL DEFAULT 0,
    pageviews BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    sessions BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, time_bucket)
);

CREATE INDEX IF NOT EXISTS idx_agg_site_hourly_bucket
    ON agg_site_hourly (time_bucket DESC);

CREATE TABLE IF NOT EXISTS agg_site_daily (
    day DATE NOT NULL,
    site_id TEXT NOT NULL,
    events BIGINT NOT NULL DEFAULT 0,
    pageviews BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    sessions BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, day)
);

CREATE INDEX IF NOT EXISTS idx_agg_site_daily_day
    ON agg_site_daily (day DESC);

CREATE TABLE IF NOT EXISTS agg_page_daily (
    day DATE NOT NULL,
    site_id TEXT NOT NULL,
    page_path TEXT NOT NULL,
    page_url TEXT NOT NULL,
    events BIGINT NOT NULL DEFAULT 0,
    pageviews BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    sessions BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, day, page_path, page_url)
);

CREATE INDEX IF NOT EXISTS idx_agg_page_daily_day
    ON agg_page_daily (day DESC);

CREATE TABLE IF NOT EXISTS agg_source_daily (
    day DATE NOT NULL,
    site_id TEXT NOT NULL,
    source TEXT NOT NULL,
    medium TEXT NOT NULL,
    campaign TEXT NOT NULL,
    referrer_host TEXT NOT NULL,
    events BIGINT NOT NULL DEFAULT 0,
    pageviews BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    sessions BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, day, source, medium, campaign, referrer_host)
);

CREATE INDEX IF NOT EXISTS idx_agg_source_daily_day
    ON agg_source_daily (day DESC);

CREATE TABLE IF NOT EXISTS agg_device_daily (
    day DATE NOT NULL,
    site_id TEXT NOT NULL,
    device_type TEXT NOT NULL,
    os_name TEXT NOT NULL,
    events BIGINT NOT NULL DEFAULT 0,
    pageviews BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    sessions BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, day, device_type, os_name)
);

CREATE INDEX IF NOT EXISTS idx_agg_device_daily_day
    ON agg_device_daily (day DESC);

CREATE TABLE IF NOT EXISTS agg_geo_daily (
    day DATE NOT NULL,
    site_id TEXT NOT NULL,
    geo_country TEXT NOT NULL,
    events BIGINT NOT NULL DEFAULT 0,
    pageviews BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    sessions BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, day, geo_country)
);

CREATE INDEX IF NOT EXISTS idx_agg_geo_daily_day
    ON agg_geo_daily (day DESC);

CREATE TABLE IF NOT EXISTS reconciliation_runs (
    id BIGSERIAL PRIMARY KEY,
    job_name TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    mismatch_count BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reconciliation_runs_job_created
    ON reconciliation_runs (job_name, created_at DESC);
