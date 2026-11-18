-- Keyway Cloud multi-tenant schema. Every row is tenant-scoped by owner_id
-- (projects) or transitively through project_id (analyses); the Store methods
-- always filter accordingly so no query crosses a tenant boundary.

CREATE TABLE IF NOT EXISTS cloud_users (
    id         TEXT PRIMARY KEY,                       -- provider:providerID, e.g. "gh:42"
    login      TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    email      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cloud_projects (
    id                 TEXT PRIMARY KEY,
    owner_id           TEXT NOT NULL,
    name               TEXT NOT NULL DEFAULT '',
    source             JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    latest_analysis_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS cloud_projects_owner_idx
    ON cloud_projects (owner_id, created_at DESC);

CREATE TABLE IF NOT EXISTS cloud_analyses (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL REFERENCES cloud_projects (id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    trigger_kind   TEXT NOT NULL DEFAULT 'manual',
    trigger_ref    TEXT NOT NULL DEFAULT '',
    hash           TEXT NOT NULL DEFAULT '',
    is_baseline    BOOLEAN NOT NULL DEFAULT false,
    consumer_count INTEGER NOT NULL DEFAULT 0,
    issuer_count   INTEGER NOT NULL DEFAULT 0,
    change_count   INTEGER NOT NULL DEFAULT 0,
    data           JSONB NOT NULL                       -- the full cloud.Analysis
);
CREATE INDEX IF NOT EXISTS cloud_analyses_project_idx
    ON cloud_analyses (project_id, created_at DESC);
