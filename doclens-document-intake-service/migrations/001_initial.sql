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

CREATE INDEX IF NOT EXISTS documents_organization_id_idx
    ON documents (organization_id, id);

CREATE TABLE IF NOT EXISTS uploads (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    checksum TEXT NOT NULL,
    storage_ref TEXT NOT NULL,
    idempotency_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
);

CREATE INDEX IF NOT EXISTS uploads_document_id_idx
    ON uploads (organization_id, document_id, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uploads_idempotency_key_idx
    ON uploads (organization_id, idempotency_key)
    WHERE idempotency_key <> '';
