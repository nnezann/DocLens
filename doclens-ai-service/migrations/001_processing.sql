CREATE SCHEMA IF NOT EXISTS doclens_processing;

CREATE TABLE IF NOT EXISTS doclens_processing.consumed_events (
    event_id TEXT PRIMARY KEY,
    consumed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS doclens_processing.processing_jobs (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE,
    organization_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('processing', 'processed', 'failed')),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    result_ref TEXT,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS doclens_processing.processing_results (
    job_id TEXT PRIMARY KEY REFERENCES doclens_processing.processing_jobs(id),
    document_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    result JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS doclens_processing.event_outbox (
    event_id TEXT PRIMARY KEY,
    routing_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    dead_lettered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS processing_jobs_document_idx
    ON doclens_processing.processing_jobs (organization_id, document_id);
