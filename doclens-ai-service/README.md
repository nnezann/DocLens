# DocLens Document Processing Service

Asynchronous Python/FastAPI worker for `DocumentUploaded` messages. It keeps
the original object-storage bytes untouched, performs deterministic metadata
and image validation, and optionally exposes an extension point for OCR/model
inference. Inference is disabled by default because the OCR provider, model,
document types, risk thresholds, cloud provider, and service authentication
are unresolved architecture decisions.

## Runtime

```bash
cd doclens-ai-service
pip install -r requirements.txt
PYTHONPATH=src python -m app
```

The worker consumes RabbitMQ routing key `document.uploaded`, writes result
JSON to S3-compatible storage, persists jobs/results/outbox records in the
`doclens_processing` PostgreSQL schema, and publishes version-1
`DocumentProcessed` events with routing key `document.processed`.

Required production settings are `DATABASE_URL`, `RABBITMQ_URL`, `S3_BUCKET`,
`S3_ENDPOINT` (when not using a provider default), `S3_ACCESS_KEY_ID`, and
`S3_SECRET_ACCESS_KEY`. All timeouts, queue names, retry count/backoff, input
limit, schema, and inference flags are configurable through environment
variables. No local paths or model URLs are assumed.

Apply `migrations/001_processing.sql` before starting a PostgreSQL-backed
worker. The outbox is published durably by the RabbitMQ publisher; failed
messages are retried with bounded exponential backoff and then rejected to
the configured dead-letter queue. Duplicate `event_id` delivery is ignored.

Redis is intentionally not used by this service: PostgreSQL is authoritative
for jobs/results, and RabbitMQ provides delivery, retry, and dead-letter state.
The only HTTP API is operational; document business operations remain
asynchronous event processing rather than the legacy synchronous gRPC server.

Health endpoints:

* `GET /health/live` — process liveness.
* `GET /health/ready` — PostgreSQL readiness (in-memory fallback is used only
  when `DATABASE_URL` is absent).
* `GET /metrics` — Prometheus exposition format.

## Boundaries and technology rationale

This service owns processing jobs/results only and never reads another
service's database. Python/FastAPI matches the repository's AI-service
decision and keeps deterministic processing close to the existing Python
pipeline while allowing future provider adapters. RabbitMQ provides durable
acknowledged work queues and DLQs; PostgreSQL provides transactional service
state; S3-compatible storage preserves raw input outside PostgreSQL.
