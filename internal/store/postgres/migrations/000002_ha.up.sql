-- HA infrastructure (architecture review W5/#6, security-audit future work).
-- These tables let multiple replicas share coordination state:
--   * keyway_idempotency  — durable idempotent-write replay across replicas
--   * keyway_operated_keys — canary/operated signing key material (encrypted),
--                            shared and durable instead of per-process memory
-- Scheduler leadership uses a Postgres session-level advisory lock and needs no
-- table.

CREATE TABLE IF NOT EXISTS keyway_idempotency (
    key          TEXT        PRIMARY KEY,
    status       INTEGER     NOT NULL,
    body         BYTEA       NOT NULL,
    content_type TEXT        NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL
);

-- Sweep expired rows efficiently.
CREATE INDEX IF NOT EXISTS idx_keyway_idempotency_expires
    ON keyway_idempotency (expires_at);

CREATE TABLE IF NOT EXISTS keyway_operated_keys (
    issuer     TEXT        PRIMARY KEY,
    -- AES-256-GCM ciphertext of the issuer's PersistedKey set. Private key
    -- material is never stored in plaintext; the root key comes from a secret
    -- manager (env / mounted file / command).
    data       BYTEA       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
