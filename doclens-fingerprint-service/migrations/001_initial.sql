CREATE TABLE IF NOT EXISTS fingerprints (
    document_id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    original_ref TEXT NOT NULL,
    normalized_ref TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    tlsh TEXT,
    phash TEXT NOT NULL,
    duplicate_of TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS fingerprints_sha256_idx ON fingerprints (sha256);
CREATE INDEX IF NOT EXISTS fingerprints_tlsh_idx ON fingerprints (tlsh) WHERE tlsh IS NOT NULL;
CREATE INDEX IF NOT EXISTS fingerprints_created_at_idx ON fingerprints (created_at);
CREATE INDEX IF NOT EXISTS fingerprints_organization_idx ON fingerprints (organization_id);

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

CREATE INDEX IF NOT EXISTS fingerprint_outbox_pending_idx
    ON event_outbox (next_attempt_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE TABLE IF NOT EXISTS event_inbox (
    event_id TEXT PRIMARY KEY,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
