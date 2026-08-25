# DocLens — Document Intake Service
## Build Prompt / Functional and Technical Specification

This prompt is for the agent implementing the Document Intake Service. It
extends `Doclens_Engineering_Specification.md`; that document is the source of
truth for architecture, ownership, communication, security, and events.

Before writing code, read these files in order:

1. `Doclens_Engineering_Specification.md`
2. This build prompt
3. `AGENTS.md`

Do not change service boundaries, API contracts, event names, or data ownership
without explicit human approval.

## 1. Service purpose

Document Intake accepts documents through the API Gateway, stores file bytes in
S3-compatible object storage, owns document lifecycle metadata, and publishes
an event for asynchronous processing. It must not perform OCR, extraction,
fingerprinting, fraud analysis, or final verification.

The service owns these resources exclusively:

- `documents`: one logical document that moves through verification.
- `uploads`: one physical file received for a logical document.
- `processing_jobs`: asynchronous processing state.

No other service may query these tables directly.

## 2. Communication and deployment

- External clients use REST/HTTPS through the API Gateway.
- The Gateway calls this service using internal gRPC.
- Asynchronous processing uses RabbitMQ.
- PostgreSQL is the authoritative metadata store.
- Redis may be used for temporary state, locks, rate limiting, or caching; it
  must not become the source of truth.
- Actual file bytes must be stored in S3-compatible object storage, never in
  PostgreSQL.
- Every network operation must have an explicit timeout.
- Provide health, readiness, liveness, and graceful shutdown behavior.

The service should be independently containerizable and expose a versioned
internal protobuf package under `proto/doclens/documents/v1`.

## 3. API contract

Implement these operations through the Gateway-facing gRPC contract:

### Create logical document

```http
POST /documents
```

Creates a logical document for the authenticated organization. The request
must include the document type and appropriate metadata. The organization ID
must come from authenticated Gateway identity metadata, not an arbitrary
client-supplied tenant value.

Return a stable document ID and initial lifecycle status.

### Get document

```http
GET /documents/{id}
```

Return document metadata only when the document belongs to the authenticated
organization. Missing documents and documents belonging to another
organization must not leak existence.

### Upload physical file

```http
POST /documents/{id}/uploads
```

Validate content type, filename, size, and document ownership. Store the bytes
in object storage under a service-generated, tenant-scoped key. Persist only
metadata and `storage_ref` in PostgreSQL. Support multiple uploads for one
logical document, such as front and back images.

Do not trust a client-provided storage path. Do not log file contents, tokens,
or sensitive extracted data.

### Get document status

```http
GET /documents/{id}/status
```

Return the logical document status and processing-job status. Status values
must be explicit and documented; at minimum support the lifecycle needed for
created, processing, processed, failed, and completed/error states without
pretending asynchronous processing has completed.

## 4. Data model

Use PostgreSQL migrations and enforce tenant scoping with `organization_id`.
The exact schema may follow normal relational conventions, but it must retain
these fields:

### documents

- `id`
- `organization_id`
- `type`
- `status`
- `created_at`
- `updated_at`

### uploads

- `id`
- `document_id`
- `organization_id`
- `filename`
- `content_type`
- `size_bytes`
- `checksum` where available
- `storage_ref`
- `created_at`

### processing_jobs

- `id`
- `document_id`
- `organization_id`
- `status`
- `attempt_count`
- `last_error` without secrets or file contents
- timestamps

Add foreign keys, useful indexes, and constraints that prevent cross-tenant
references. A logical document may have multiple uploads. A physical upload
must belong to exactly one logical document.

## 5. DocumentUploaded event

Publish an event after the metadata transaction and object-storage write have
completed successfully. Use RabbitMQ routing key:

```text
document.uploaded
```

Use the stable event envelope defined in the engineering specification:

```json
{
  "event_id": "evt_123",
  "event_type": "DocumentUploaded",
  "event_version": 1,
  "occurred_at": "2026-08-25T12:00:00Z",
  "organization_id": "org_123",
  "document_id": "doc_456",
  "payload": {
    "type": "certificate",
    "upload_ids": ["upl_789"]
  }
}
```

Consumers must be able to identify the document, organization, upload
metadata, and storage reference without reading this service's database.

Use an outbox or an equivalent durable publication mechanism so a successful
upload is not silently lost if RabbitMQ is temporarily unavailable. Publishing
must be idempotent or safely retryable. Consumers will receive duplicates, so
include `event_id` and document the deduplication behavior.

## 6. UserCreated consumption

The ownership matrix lists `UserCreated` as an input. Implement only the
minimal integration required by the confirmed contract. Do not create or
duplicate Identity tables. If the service needs local authorization metadata,
maintain a clearly bounded projection and document its consistency behavior.

If the current repository does not yet provide a RabbitMQ contract or
`UserCreated` consumer foundation, create the contract abstraction and a
disabled/configurable adapter rather than inventing payload fields or silently
using another broker.

## 7. Security and tenant isolation

- Trust organization/user identity only from authenticated Gateway metadata.
- Reject unauthenticated requests before business logic.
- Enforce `organization_id` on every document, upload, and processing-job query.
- Prevent path traversal and unsafe object keys.
- Enforce configured request and upload-size limits.
- Validate allowed MIME types and file metadata.
- Never store passwords, access tokens, or raw document bytes in logs.
- Return the repository's consistent error shape with `request_id`.
- Propagate `X-Request-Id`/trace context into gRPC calls and events.
- Use TLS for production internal gRPC and RabbitMQ connections when the
  repository's selected service-auth configuration is available.

## 8. Reliability

Implement:

- idempotent upload handling where an idempotency key is supplied,
- explicit database/object-storage failure handling,
- bounded retries with configurable exponential backoff,
- RabbitMQ acknowledgement only after durable processing,
- dead-letter handling for repeatedly failed asynchronous work,
- graceful shutdown,
- readiness that reflects required dependencies.

Do not add broad catches or success-shaped fallbacks. Surface failures with
safe, actionable errors.

## 9. Observability

Emit structured JSON logs containing at least:

- timestamp,
- service,
- operation,
- request ID,
- trace ID when available,
- organization ID where safe,
- resource ID,
- duration,
- status.

Provide Prometheus-compatible metrics for request count, latency, errors,
upload bytes/count, storage failures, event publication failures, queue/retry
counts, and processing-job status.

Use OpenTelemetry instrumentation consistent with the repository stack.

## 10. Testing requirements

Add tests for:

- document creation,
- tenant isolation,
- upload metadata validation,
- object-storage key generation,
- multiple uploads per document,
- missing document behavior,
- status retrieval,
- database transaction failure,
- object-storage failure,
- durable/retryable event publication,
- event envelope and routing key,
- idempotent duplicate delivery,
- request ID and error propagation.

Add integration/contract tests for the gRPC service and Gateway route. Use
existing repository test tooling; do not introduce a second test framework.

## 11. Documentation

Create or update the service README with:

- service responsibilities and boundaries,
- REST-to-gRPC API mapping,
- protobuf generation commands,
- PostgreSQL schema/migrations,
- Redis usage,
- object-storage abstraction and configuration,
- RabbitMQ event contract,
- local development instructions,
- health/readiness behavior,
- security and tenant-isolation model,
- language/technology analysis and rationale.

## 12. Required implementation discipline

Use a feature branch cut from the backend domain branch. Keep commits atomic,
include the required Copilot co-author trailer, and do not commit secrets.

Before implementation, stop and ask the repository owner if any of these
choices are needed and not already decided elsewhere:

1. The concrete implementation language for Document Intake. The blueprint
   explicitly reserves Python for AI/ML services and does not otherwise pin
   this service's language; Go is a reasonable fit for the existing Gateway,
   Identity, and gRPC stack but must be documented rather than silently
   assumed.
2. The concrete S3-compatible provider for local/production deployment. The
   architecture intentionally supports AWS S3, MinIO, and other providers;
   keep the provider behind an abstraction.
3. The production multi-tenant isolation strategy. The MVP should enforce
   application-level `organization_id` scoping without blocking a future
   schema/database-per-tenant migration.
4. The service-to-service authentication mechanism, because the blueprint
   leaves mTLS, service-account JWTs, workload identity, and network policy
   open.

Do not substitute Kafka, a dedicated vector database, a second primary
database, or direct database access by another service.

## 13. Acceptance criteria

The service is complete when:

- authenticated clients can create and retrieve only their organization's
  document metadata;
- physical files are stored outside PostgreSQL and referenced by `storage_ref`;
- one logical document supports multiple physical uploads;
- invalid metadata and oversized/unsupported uploads are rejected;
- document status is queryable without exposing another tenant's data;
- `DocumentUploaded` uses the specified envelope and RabbitMQ routing key;
- an event cannot be silently lost after a successful upload;
- duplicate event delivery is safe;
- health/readiness, structured logs, metrics, tracing, and request IDs exist;
- gRPC contracts, migrations, unit tests, integration tests, and README
  documentation are included;
- no OCR, AI, verification, or unrelated domain logic is added.
