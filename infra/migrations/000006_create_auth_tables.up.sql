CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sites (
    id          TEXT PRIMARY KEY,       -- e.g. 'site_abc'
    name        TEXT NOT NULL,
    user_id    TEXT NOT NULL,          -- links to your users table
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          TEXT PRIMARY KEY,       -- opaque key ID shown in the dashboard
    site_id     TEXT NOT NULL REFERENCES sites(id),
    key_hash    TEXT NOT NULL UNIQUE,   -- SHA-256(raw_key), never store plaintext
    key_type    TEXT NOT NULL DEFAULT 'public', -- 'public' now; 'secret' later
    label       TEXT,                   -- e.g. "Production tracker"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ             -- NULL means active
);

CREATE INDEX ON api_keys (key_hash) WHERE revoked_at IS NULL;
