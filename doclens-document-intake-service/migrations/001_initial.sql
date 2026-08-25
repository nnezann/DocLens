CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    type TEXT NOT NULL,
    filename TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('created', 'processing', 'processed', 'completed', 'failed', 'error')),
    processing_job_status TEXT NOT NULL DEFAULT 'pending',
    storage_ref TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS documents_organization_id_id_key
    ON documents (organization_id, id);

CREATE INDEX IF NOT EXISTS documents_organization_id_idx
    ON documents (organization_id, id);

CREATE TABLE IF NOT EXISTS uploads (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    checksum TEXT NOT NULL,
    storage_ref TEXT NOT NULL,
    idempotency_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (organization_id, document_id)
        REFERENCES documents (organization_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS uploads_document_id_idx
    ON uploads (organization_id, document_id, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uploads_idempotency_key_idx
    ON uploads (organization_id, idempotency_key)
    WHERE idempotency_key <> '';

CREATE TABLE IF NOT EXISTS event_outbox (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    event_version INTEGER NOT NULL,
    routing_key TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'publishing', 'published', 'failed')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS event_outbox_pending_idx
    ON event_outbox (next_attempt_at, created_at)
    WHERE status IN ('pending', 'failed');
