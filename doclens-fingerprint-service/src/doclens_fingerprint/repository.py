from __future__ import annotations

from contextlib import contextmanager
from datetime import datetime, timezone
import json
from typing import Any, Iterator, Optional
import uuid

import psycopg
from psycopg.rows import dict_row

from .models import DuplicateMatch, Fingerprint
from .hashing import hamming_distance, tlsh_distance


class PostgresRepository:
    def __init__(self, database_url: str, connect_timeout: float = 10.0):
        self.database_url = database_url
        self.connect_timeout = connect_timeout

    @contextmanager
    def connection(self) -> Iterator[psycopg.Connection]:
        conn = psycopg.connect(
            self.database_url, connect_timeout=self.connect_timeout, row_factory=dict_row,
            options=f"-c statement_timeout={int(self.connect_timeout * 1000)}",
        )
        try:
            yield conn
        finally:
            conn.close()

    def migrate(self, migration_sql: str) -> None:
        with self.connection() as conn:
            with conn.cursor() as cur:
                cur.execute(migration_sql)
            conn.commit()

    def ping(self) -> None:
        with self.connection() as conn, conn.cursor() as cur:
            cur.execute("SELECT 1")
            cur.fetchone()

    def get(self, document_id: str) -> Optional[Fingerprint]:
        with self.connection() as conn, conn.cursor() as cur:
            cur.execute("SELECT * FROM fingerprints WHERE document_id = %s", (document_id,))
            row = cur.fetchone()
        return self._model(row) if row else None

    def find_candidates(self, current: Fingerprint, limit: int = 1000) -> list[DuplicateMatch]:
        with self.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """SELECT document_id, sha256, tlsh, phash, duplicate_of
                   FROM fingerprints WHERE document_id <> %s
                   ORDER BY created_at DESC LIMIT %s""",
                (current.document_id, limit),
            )
            rows = cur.fetchall()
        matches = []
        for row in rows:
            matches.append(DuplicateMatch(
                document_id=row["document_id"],
                tlsh_distance=tlsh_distance(current.tlsh, row["tlsh"]),
                phash_distance=hamming_distance(current.phash, row["phash"]),
                exact_match=current.sha256 == row["sha256"],
                duplicate_of=row["duplicate_of"],
            ))
        return matches

    def save_with_outbox(self, fingerprint: Fingerprint, event: dict[str, Any],
                         organization_id: str, event_id: Optional[str] = None,
                         inbox_event_id: Optional[str] = None) -> None:
        event_id = event_id or str(uuid.uuid4())
        created = fingerprint.created_at.astimezone(timezone.utc)
        with self.connection() as conn:
            with conn.cursor() as cur:
                if inbox_event_id:
                    cur.execute("INSERT INTO event_inbox (event_id) VALUES (%s) ON CONFLICT DO NOTHING",
                                (inbox_event_id,))
                if inbox_event_id and cur.rowcount == 0:
                    conn.rollback()
                    return
                cur.execute(
                    """INSERT INTO fingerprints
                       (document_id, organization_id, original_ref, normalized_ref, sha256, tlsh, phash, duplicate_of, created_at)
                       VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)
                       ON CONFLICT (document_id) DO UPDATE SET
                         organization_id=EXCLUDED.organization_id,
                         original_ref=EXCLUDED.original_ref, normalized_ref=EXCLUDED.normalized_ref,
                         sha256=EXCLUDED.sha256, tlsh=EXCLUDED.tlsh, phash=EXCLUDED.phash,
                         duplicate_of=EXCLUDED.duplicate_of""",
                    (fingerprint.document_id, organization_id, fingerprint.original_ref, fingerprint.normalized_ref,
                     fingerprint.sha256, fingerprint.tlsh, fingerprint.phash,
                     fingerprint.duplicate_of, created),
                )
                cur.execute(
                    """INSERT INTO event_outbox
                       (id,event_type,event_version,routing_key,organization_id,document_id,payload,
                        status,next_attempt_at,created_at)
                       VALUES (%s,'FingerprintCreated',1,'fingerprint.created',%s,%s,%s,'pending',%s,%s)
                       ON CONFLICT (id) DO NOTHING""",
                    (event_id, organization_id, fingerprint.document_id, json.dumps(event), created, created),
                )
            conn.commit()

    def claim_outbox(self, limit: int = 50) -> list[dict[str, Any]]:
        with self.connection() as conn, conn.cursor() as cur:
            cur.execute(
                """WITH claimed AS (
                     SELECT id FROM event_outbox
                     WHERE status IN ('pending','failed') AND next_attempt_at <= NOW()
                     ORDER BY next_attempt_at, created_at FOR UPDATE SKIP LOCKED LIMIT %s
                   )
                   UPDATE event_outbox e SET status='publishing', attempt_count=e.attempt_count+1
                   FROM claimed WHERE e.id=claimed.id
                   RETURNING e.id,e.routing_key,e.payload,e.attempt_count""",
                (limit,),
            )
            rows = cur.fetchall()
            conn.commit()
        return list(rows)

    def mark_outbox_published(self, event_id: str) -> None:
        with self.connection() as conn, conn.cursor() as cur:
            cur.execute("UPDATE event_outbox SET status='published',published_at=NOW(),last_error='' WHERE id=%s",
                        (event_id,))
            conn.commit()

    def mark_outbox_failed(self, event_id: str, error: str, next_attempt_at: datetime) -> None:
        with self.connection() as conn, conn.cursor() as cur:
            cur.execute(
                "UPDATE event_outbox SET status='failed',last_error=LEFT(%s,1000),next_attempt_at=%s WHERE id=%s",
                (error, next_attempt_at, event_id),
            )
            conn.commit()

    @staticmethod
    def _model(row: dict[str, Any]) -> Fingerprint:
        return Fingerprint(
            document_id=row["document_id"], original_ref=row["original_ref"],
            normalized_ref=row["normalized_ref"], sha256=row["sha256"], tlsh=row["tlsh"],
            phash=row["phash"], duplicate_of=row["duplicate_of"], created_at=row["created_at"],
        )
