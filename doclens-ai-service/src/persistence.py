"""Service-owned PostgreSQL persistence and an in-memory test implementation."""

import json
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from typing import Any


class PostgresRepository:
    def __init__(self, database_url: str, schema: str) -> None:
        self.database_url = database_url
        self.schema = schema
        self.connected = False

    async def connect(self) -> None:
        import psycopg
        async with await psycopg.AsyncConnection.connect(
            self.database_url, connect_timeout=10,
            options="-c statement_timeout=30000",
        ) as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(f'CREATE SCHEMA IF NOT EXISTS "{self.schema}"')
        self.connected = True

    async def close(self) -> None:
        self.connected = False

    @asynccontextmanager
    async def _connection(self):
        import psycopg
        from psycopg.rows import dict_row
        conn = await psycopg.AsyncConnection.connect(
            self.database_url, connect_timeout=10, row_factory=dict_row,
            options="-c statement_timeout=30000",
        )
        try:
            yield conn
        finally:
            await conn.close()

    async def health(self) -> bool:
        if not self.connected:
            return False
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute("SELECT 1")
                return bool(await cursor.fetchone())

    async def claim_event(self, event_id: str) -> bool:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                    f'INSERT INTO "{self.schema}".consumed_events (event_id) VALUES (%s) '
                    "ON CONFLICT (event_id) DO NOTHING RETURNING event_id", (event_id,),
                )
                return await cursor.fetchone() is not None

    async def start_job(self, event_id: str, organization_id: str, document_id: str) -> str:
        job_id = f"job_{event_id}"
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                f'INSERT INTO "{self.schema}".processing_jobs '
                "(id, event_id, organization_id, document_id, status, attempt_count, created_at, updated_at) "
                "VALUES (%s,%s,%s,%s,'processing',1,NOW(),NOW()) "
                "ON CONFLICT (event_id) DO UPDATE SET status='processing', attempt_count="
                "processing_jobs.attempt_count+1, updated_at=NOW()", (job_id, event_id, organization_id, document_id))
        return job_id

    async def release_event(self, event_id: str) -> None:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                    f'DELETE FROM "{self.schema}".consumed_events WHERE event_id=%s', (event_id,)
                )

    async def complete_job(self, job_id: str, result: dict[str, Any], event: dict[str, Any]) -> None:
        async with self._connection() as conn:
            async with conn.transaction():
                async with conn.cursor() as cursor:
                    await cursor.execute(
                        f'UPDATE "{self.schema}".processing_jobs SET status=%s, result_ref=%s, '
                        "last_error=NULL, updated_at=NOW() WHERE id=%s",
                        (result.get("status", "processed"), result.get("result_ref"), job_id),
                        )
                    await cursor.execute(
                    f'INSERT INTO "{self.schema}".processing_results '
                    "(job_id, document_id, organization_id, result, created_at) VALUES (%s,%s,%s,%s,NOW()) "
                    "ON CONFLICT (job_id) DO UPDATE SET result=%s",
                    (job_id, result["document_id"], result["organization_id"], json.dumps(result), json.dumps(result)),
                )
                    await cursor.execute(
                    f'INSERT INTO "{self.schema}".event_outbox '
                    "(event_id, routing_key, payload, attempts, available_at, created_at) "
                    "VALUES (%s,%s,%s,0,NOW(),NOW()) ON CONFLICT (event_id) DO NOTHING",
                    (event["event_id"], "document.processed", json.dumps(event)),
                )

    async def fail_job(self, job_id: str, error: str) -> None:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                    f'UPDATE "{self.schema}".processing_jobs SET status=\'failed\', last_error=%s, updated_at=NOW() WHERE id=%s',
                    (error[:1000], job_id),
                )

    async def add_outbox(self, event: dict[str, Any], routing_key: str) -> None:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                f'INSERT INTO "{self.schema}".event_outbox '
                "(event_id, routing_key, payload, attempts, available_at, created_at) "
                "VALUES (%s,%s,%s,0,NOW(),NOW()) ON CONFLICT (event_id) DO NOTHING",
                (event["event_id"], routing_key, json.dumps(event)),
            )

    async def pending_outbox(self, limit: int = 50) -> list[dict[str, Any]]:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                f'SELECT event_id, routing_key, payload, attempts FROM "{self.schema}".event_outbox '
                "WHERE published_at IS NULL AND dead_lettered_at IS NULL AND available_at <= NOW() "
                "ORDER BY created_at LIMIT %s", (limit,))
                return await cursor.fetchall()

    async def mark_outbox_published(self, event_id: str) -> None:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                    f'UPDATE "{self.schema}".event_outbox SET published_at=NOW() WHERE event_id=%s', (event_id,)
                )

    async def mark_outbox_retry(self, event_id: str, attempts: int, delay: float, dead_letter: bool) -> None:
        async with self._connection() as conn:
            async with conn.cursor() as cursor:
                await cursor.execute(
                f'UPDATE "{self.schema}".event_outbox SET attempts=%s, '
                "available_at=NOW()+(%s * INTERVAL '1 second'), "
                "dead_lettered_at=CASE WHEN %s THEN NOW() ELSE dead_lettered_at END WHERE event_id=%s",
                (attempts, delay, dead_letter, event_id),
            )


class InMemoryRepository:
    def __init__(self) -> None:
        self.events: set[str] = set()
        self.jobs: dict[str, dict[str, Any]] = {}
        self.outbox: list[dict[str, Any]] = []

    async def health(self) -> bool:
        return True

    async def close(self) -> None:
        return None

    async def claim_event(self, event_id: str) -> bool:
        if event_id in self.events:
            return False
        self.events.add(event_id)
        return True

    async def release_event(self, event_id: str) -> None:
        self.events.discard(event_id)

    async def start_job(self, event_id: str, organization_id: str, document_id: str) -> str:
        job_id = f"job_{event_id}"
        self.jobs[job_id] = {
            "event_id": event_id, "organization_id": organization_id,
            "document_id": document_id, "status": "processing", "attempt_count": 1,
        }
        return job_id

    async def complete_job(self, job_id: str, result: dict[str, Any], event: dict[str, Any]) -> None:
        self.jobs[job_id].update({"status": result.get("status", "processed"), "result": result})
        await self.add_outbox(event, "document.processed")

    async def fail_job(self, job_id: str, error: str) -> None:
        self.jobs[job_id].update({"status": "failed", "last_error": error[:1000]})

    async def add_outbox(self, event: dict[str, Any], routing_key: str) -> None:
        self.outbox.append({"event_id": event["event_id"], "routing_key": routing_key, "payload": event})

    async def pending_outbox(self, limit: int = 50) -> list[dict[str, Any]]:
        return self.outbox[:limit]

    async def mark_outbox_published(self, event_id: str) -> None:
        self.outbox[:] = [item for item in self.outbox if item["event_id"] != event_id]

    async def mark_outbox_retry(self, event_id: str, attempts: int, delay: float, dead_letter: bool) -> None:
        for item in self.outbox:
            if item["event_id"] == event_id:
                item["attempts"] = attempts
                if dead_letter:
                    self.outbox.remove(item)
                return
