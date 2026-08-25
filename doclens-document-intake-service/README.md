# DocLens Document Intake Service

Go gRPC service for ingesting uploaded documents and persisting their metadata in a tenant-scoped store. It is the first service in the verification pipeline and owns the `documents` lifecycle boundary described in `Doclens_Engineering_Specification.md`.

## Why Go

This service is implemented in Go because the repository already uses Go for the API Gateway and Identity Service, and Go is a strong fit for highly concurrent network services with gRPC, structured logging, and simple deployment. The document intake path is primarily I/O-bound and request-driven rather than CPU-heavy, so the implementation does not justify a Rust rewrite at this stage.

Rust would only be a worthwhile follow-up if we observe sustained, high-volume ingestion bottlenecks at the process boundary after the service has been benchmarked under realistic multi-tenant traffic. Until then, the service architecture remains simple, maintainable, and consistent with the rest of DocLens.

## Service responsibilities

- Accept a logical document request from the API Gateway
- Validate tenant ownership, content type, filename, and size
- Write file bytes to a pluggable object-store adapter
- Persist metadata in a tenant-scoped in-memory store
- Expose gRPC endpoints for creation, upload, retrieval, and processing status
- Provide health checks for readiness and liveness

## RPC surface

- `CreateDocument`: creates a logical document record
- `GetDocument`: fetches a document by tenant and ID
- `UploadDocument`: stores physical bytes and returns upload metadata/checksum
- `CreateUploadIntent`: creates a pending upload and returns a pre-signed PUT URL
- `CompleteUpload`: verifies the direct-to-storage object and confirms the upload
- `GetDocumentStatus`: returns the current lifecycle and processing status

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `DOCUMENT_INTAKE_GRPC_ADDR` | `:9002` | gRPC listening address |
| `DOCUMENT_INTAKE_METRICS_ADDR` | `:9092` | Prometheus metrics HTTP address |
| `DOCUMENT_INTAKE_STORAGE_DIR` | `./data/documents` | Local object-store directory |
| `DOCUMENT_INTAKE_MAX_UPLOAD_BYTES` | `10485760` | 10 MiB per upload |
| `DOCUMENT_INTAKE_ALLOWED_CONTENT_TYPES` | `application/pdf,image/jpeg,image/png,image/webp` | Allowed upload types |
| `DATABASE_URL` | empty | PostgreSQL connection URL; empty uses in-memory metadata for local development |
| `R2_ENDPOINT` | empty | Cloudflare R2 S3-compatible endpoint |
| `R2_ACCESS_KEY_ID` | empty | R2 API token access key |
| `R2_SECRET_ACCESS_KEY` | empty | R2 API token secret |
| `R2_BUCKET` | empty | R2 bucket name |
| `RABBITMQ_URL` | empty | RabbitMQ connection URL; requires `DATABASE_URL` |
| `RABBITMQ_EXCHANGE` | `doclens.events` | Durable topic exchange for outbox events |
| `DOCUMENT_INTAKE_UPLOAD_WORKERS` | `8` | Maximum concurrent proxied uploads |
| `DOCUMENT_INTAKE_TENANT_UPLOAD_RATE` | `5` | Per-organization proxied-upload tokens per second |
| `DOCUMENT_INTAKE_TENANT_UPLOAD_BURST` | `10` | Per-organization token bucket capacity |
| `DOCUMENT_INTAKE_STORAGE_FAILURE_THRESHOLD` | `5` | Object-storage failures before circuit opening |
| `DOCUMENT_INTAKE_STORAGE_CIRCUIT_OPEN` | `30s` | Circuit-breaker open duration |

## Run

```bash
cd doclens-document-intake-service
go run ./cmd/document-intake
```

The service uses a local object-store adapter by default. Configure all R2 variables to use Cloudflare R2. Configure `DATABASE_URL` to use PostgreSQL; otherwise the in-memory metadata store is used only for local development.

When both `DATABASE_URL` and `RABBITMQ_URL` are configured, uploads write a durable `DocumentUploaded` outbox record in the same PostgreSQL transaction as their metadata. A background publisher retries pending records and publishes them with routing key `document.uploaded` and persistent delivery mode. Consumers should deduplicate using the envelope `event_id`/AMQP `message_id`.

For the primary upload path, call `CreateUploadIntent`, upload bytes to the returned
URL, then call `CompleteUpload`. Intents remain `pending` until a storage `HEAD`
matches the declared size/checksum; confirmation and outbox insertion are
transactional. `UploadDocument` is the bounded proxied fallback and returns
`ResourceExhausted` with `retry-after` metadata when its pool or tenant limiter
rejects admission. `original_ref`/`storage_ref` is write-once per upload and is
never transformed by this service. Provider notification adapters can call
`ConfirmUploadNotification` with the same upload identity and verification
metadata, so notification-driven and client-driven confirmation share one path.

## Authorization

The gRPC service requires authenticated Gateway metadata on document operations:
`x-organization-id` must match the request tenant and `x-roles` must contain an
allowed role. Document reads/status checks require a document-read permission;
creation and uploads require their corresponding document permissions. The
current allowed organization roles are `platform_admin`, `org_admin`, `reviewer`,
`analyst`, `user`, and the existing `member` compatibility role.
