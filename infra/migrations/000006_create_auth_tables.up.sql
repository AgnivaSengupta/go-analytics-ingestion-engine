CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),       
    name        TEXT NOT NULL,
    user_id     UUID NOT NULL REFERENCES users(id),         
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),       
    name        TEXT NOT NULL,
    site_id     UUID NOT NULL REFERENCES sites(id),
    user_id     UUID NOT NULL REFERENCES users(id),
    key_hash    TEXT NOT NULL UNIQUE,   
    key_type    TEXT NOT NULL DEFAULT 'public',             
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ             
);

CREATE INDEX ON api_keys (key_hash) WHERE revoked_at IS NULL;
CREATE INDEX ON api_keys (user_id, site_id) WHERE revoked_at IS NULL;
