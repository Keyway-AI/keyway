-- Keyway initial schema (PRD §4, §8). Full graph blobs live in JSONB for
-- provenance/flexibility; hot query columns are promoted alongside.

CREATE TABLE IF NOT EXISTS contract_versions (
    id           UUID PRIMARY KEY,
    hash         TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_baseline  BOOLEAN     NOT NULL DEFAULT FALSE,
    trigger_kind TEXT        NOT NULL DEFAULT 'manual',
    trigger_ref  TEXT,
    -- The complete ContractVersion (issuers, consumers, edges) as canonical JSON.
    data         JSONB       NOT NULL
);

-- Only one baseline row is expected; index to find it fast.
CREATE INDEX IF NOT EXISTS idx_contract_versions_baseline
    ON contract_versions (is_baseline) WHERE is_baseline;
CREATE INDEX IF NOT EXISTS idx_contract_versions_created_at
    ON contract_versions (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contract_versions_hash
    ON contract_versions (hash);

CREATE TABLE IF NOT EXISTS change_events (
    id           UUID PRIMARY KEY,
    from_version UUID        NOT NULL REFERENCES contract_versions (id),
    to_version   UUID        NOT NULL REFERENCES contract_versions (id),
    consumer_id  TEXT        NOT NULL,
    field        TEXT        NOT NULL,
    old_value    JSONB,
    new_value    JSONB,
    class        TEXT        NOT NULL,
    severity     TEXT        NOT NULL,
    confidence   DOUBLE PRECISION NOT NULL DEFAULT 0,
    evidence     JSONB,
    attribution  JSONB,
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_change_events_detected_at
    ON change_events (detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_change_events_class_severity
    ON change_events (class, severity);

CREATE TABLE IF NOT EXISTS probe_results (
    id           UUID PRIMARY KEY,
    probe_id     TEXT        NOT NULL,
    consumer_id  TEXT        NOT NULL,
    endpoint_url TEXT        NOT NULL,
    status_code  INTEGER     NOT NULL,
    latency_ms   INTEGER     NOT NULL,
    passed       BOOLEAN     NOT NULL,
    -- Truncated response body (<=512 bytes). NEVER the minted token (PRD OPEN-4).
    raw_response TEXT,
    run_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_probe_results_consumer
    ON probe_results (consumer_id, run_at DESC);
CREATE INDEX IF NOT EXISTS idx_probe_results_probe
    ON probe_results (probe_id, run_at DESC);
