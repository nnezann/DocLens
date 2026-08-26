"""FastAPI health/metrics runtime for the asynchronous processing worker."""

import logging
from contextlib import asynccontextmanager
from time import perf_counter
from uuid import uuid4

from fastapi import FastAPI, Request
from fastapi.responses import Response

from config import settings
from events import ProcessingHandler, RabbitConsumer
from logging_utils import configure_logging, request_id, trace_id
from metrics import CONTENT_TYPE_LATEST, LATENCY, REQUESTS, generate_latest
from persistence import InMemoryRepository, PostgresRepository
from storage import S3ObjectStorage

configure_logging()
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    repository = PostgresRepository(settings.database_url, settings.database_schema) if settings.database_url else InMemoryRepository()
    if settings.database_url:
        await repository.connect()
    storage = S3ObjectStorage(settings) if settings.s3_bucket else None
    consumer = RabbitConsumer(ProcessingHandler(repository, storage, settings), repository, settings) if storage else None
    if consumer:
        await consumer.start()
    app.state.repository, app.state.consumer = repository, consumer
    yield
    if consumer:
        await consumer.close()
    await repository.close()


app = FastAPI(title="DocLens Document Processing Service", version="1.0.0", lifespan=lifespan)


@app.middleware("http")
async def correlation_middleware(request: Request, call_next):
    rid = request.headers.get("x-request-id", f"req_{uuid4().hex}")
    tid = request.headers.get("traceparent", request.headers.get("x-trace-id", ""))
    rid_token, tid_token = request_id.set(rid), trace_id.set(tid)
    start = perf_counter()
    try:
        response = await call_next(request)
        status = str(response.status_code)
        response.headers["X-Request-Id"] = rid
        return response
    finally:
        REQUESTS.labels(operation=request.url.path, status=locals().get("status", "500")).inc()
        LATENCY.labels(operation=request.url.path).observe(perf_counter() - start)
        request_id.reset(rid_token)
        trace_id.reset(tid_token)


@app.get("/health/live")
async def liveness():
    return {"status": "ok"}


@app.get("/health/ready")
async def readiness(request: Request):
    repository = getattr(request.app.state, "repository", None)
    consumer = getattr(request.app.state, "consumer", None)
    dependencies_configured = not settings.s3_bucket or bool(consumer and consumer.started)
    ready = bool(repository and dependencies_configured and await repository.health())
    return Response(content='{"status":"ok"}' if ready else '{"status":"not_ready"}',
                    status_code=200 if ready else 503, media_type="application/json")


@app.get("/metrics")
async def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app:app", host=settings.host, port=settings.port)
