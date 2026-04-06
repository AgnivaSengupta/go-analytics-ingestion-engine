CREATE TABLE IF NOT EXISTS aggregate_work_queue (
    id BIGSERIAL PRIMARY KEY,
    site_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    enqueued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    UNIQUE (site_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_aggregate_work_queue_status_enqueued
    ON aggregate_work_queue (status, enqueued_at ASC);

CREATE INDEX IF NOT EXISTS idx_aggregate_work_queue_occurred_at
    ON aggregate_work_queue (occurred_at ASC);
