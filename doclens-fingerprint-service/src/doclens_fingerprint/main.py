from __future__ import annotations

import asyncio
from pathlib import Path
import threading

import uvicorn

from .config import Settings
from .health import create_health_app
from .logging import configure_logging
from .rabbit import consume_document_processed, publish_outbox
from .repository import PostgresRepository
from .grpc_server import create_grpc_server
from .storage import S3ObjectStore
from .service import FingerprintProcessor


def main() -> None:
    settings = Settings()
    logger = configure_logging()
    if not settings.database_url:
        raise RuntimeError("DATABASE_URL is required for the fingerprint service")
    repository = PostgresRepository(settings.database_url, settings.external_timeout_seconds)
    migration = Path(__file__).parents[2] / "migrations" / "001_initial.sql"
    repository.migrate(migration.read_text())
    store = S3ObjectStore(settings.s3_endpoint, settings.s3_access_key, settings.s3_secret_key,
                          settings.s3_bucket, settings.s3_region, settings.external_timeout_seconds)
    processor = FingerprintProcessor(store, repository, settings, logger)
    grpc_server = create_grpc_server(processor, settings.grpc_addr)
    grpc_server.start()
    logger.info("grpc_started", extra={"address": settings.grpc_addr})

    async def workers() -> None:
        jobs = [consume_document_processed(processor, settings, logger),
                publish_outbox(repository, settings, logger)]
        await asyncio.gather(*jobs)

    worker_thread = threading.Thread(target=lambda: asyncio.run(workers()), daemon=True)
    worker_thread.start()
    host, port = settings.http_addr.rsplit(":", 1)
    uvicorn.run(create_health_app(repository.ping, settings.external_timeout_seconds),
                host=host, port=int(port), log_config=None)


if __name__ == "__main__":
    main()
