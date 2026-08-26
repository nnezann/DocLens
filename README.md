# DocLens

DocLens is an AI-powered document verification platform built as a set of
independently deployable services. The repository currently contains the
platform foundation: a Go API Gateway, a Go Identity Service, and a Go
Document Intake Service.

The architecture uses REST/HTTPS for external clients, gRPC for internal
service calls, RabbitMQ for asynchronous events, PostgreSQL for authoritative
Document Intake metadata, and S3-compatible object storage for document bytes.

## Current repository stage

### API Gateway

`doclens-api-gateway/` exposes the external REST API and provides:

- JWT authentication enforcement
- Per-client rate limiting
- Request timeouts and request IDs
- Structured logging and Prometheus metrics
- Gateway, dependency readiness, and health endpoints
- gRPC forwarding to Identity, Document Intake, and Verification contracts

Implemented routes include:

| Method | Path | Upstream |
| --- | --- | --- |
| `GET` | `/healthz` | Gateway |
| `GET` | `/readyz` | Dependency health |
| `GET` | `/metrics` | Gateway |
| `POST` | `/identity/login` | Identity |
| `POST` | `/identity/users` | Identity |
| `POST` | `/documents` | Document Intake |
| `GET` | `/documents/{id}` | Document Intake |
| `POST` | `/documents/{id}/uploads` | Document Intake |
| `GET` | `/documents/{id}/status` | Document Intake |
| `POST` | `/verifications` | Verification contract |
| `GET` | `/verifications/{id}` | Verification contract |

### Identity Service

`doclens-identity-service/` provides organization-scoped user creation and
login over gRPC. It issues gateway-compatible JWT access tokens and refresh
tokens, and exposes a gRPC health check.

The current implementation uses an in-memory store and an optional seeded
development administrator. PostgreSQL/Redis persistence is not implemented
yet.

### Document Intake Service

`doclens-document-intake-service/` owns:

- Logical document records
- Physical upload metadata
- Processing-job status

It provides gRPC operations to create, retrieve, upload, and inspect document
status. It validates tenant scope, filenames, MIME types, and upload size.
Bytes are written to local storage by default or to Cloudflare R2 through the
S3-compatible adapter.

When PostgreSQL is configured, document metadata, upload metadata, processing
state, and `DocumentUploaded` outbox records are persisted transactionally.
The RabbitMQ publisher retries pending outbox records using the
`document.uploaded` routing key. Without PostgreSQL, the service uses an
in-memory metadata store for local development.

## Repository layout

```text
doclens-api-gateway/              REST gateway and gateway protobuf contracts
doclens-identity-service/         Identity gRPC service
doclens-document-intake-service/  Document Intake gRPC service and migrations
Doclens_Engineering_Specification.md
DocLens_Document_Intake_Build_Prompt.md
```

Each service has its own README with configuration, usage, and technology
rationale.

## Running locally

Start the Identity Service:

```bash
cd doclens-identity-service
JWT_SECRET=dev-secret go run ./cmd/identity
```

Start the Document Intake Service:

```bash
cd doclens-document-intake-service
go run ./cmd/document-intake
```

Start the API Gateway in a third terminal:

```bash
cd doclens-api-gateway
JWT_SECRET=dev-secret go run ./cmd/gateway
```

The gateway defaults to `http://localhost:8080`, Identity to gRPC
`localhost:9001`, and Document Intake to gRPC `localhost:9002`. The gateway
also dials the configured Verification endpoint; verification routes require a
running compatible service when used.

For a quick local login using the seeded identity:

```bash
curl -s http://localhost:8080/identity/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@doclens.local","password":"doclens-dev"}'
```

Set `GATEWAY_AUTH_DISABLED=true` only for local development when JWT validation
is intentionally bypassed. See each service README for all configuration
variables, PostgreSQL/RabbitMQ setup, and R2 configuration.

### Docker Compose local stack

Run the complete local integration environment with PostgreSQL, RabbitMQ,
RustFS, Identity, Document Intake, the API Gateway, and a verification stub:

```bash
cp .env.local.example .env
docker compose -f docker-compose.local.yml up --build
```

The gateway is available at `http://localhost:8080`, RustFS S3 API at
`http://localhost:9000` with its console at `http://localhost:9001`, RabbitMQ
management at `http://localhost:15672`, and Document Intake metrics at
`http://localhost:9092/metrics`.

This Compose stack is for local testing only. Production must use managed or
production-operated PostgreSQL, RabbitMQ, and S3-compatible storage with
unique secrets, TLS, backups, monitoring, resource limits, and deployment
secret management. The verification container is explicitly a local stub and
must not be deployed as a production service.

## Event contract

`DocumentUploaded` is published with routing key `document.uploaded` using the
versioned envelope defined in `Doclens_Engineering_Specification.md`:

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

Consumers must deduplicate events by `event_id` or the AMQP message ID.

## Planned services

The broader architecture defines additional services that are not implemented
in this repository stage:

- Document Processing and OCR
- Fingerprinting and similarity search
- Knowledge and Rules services
- Verification Engine
- Fraud Detection and Risk Assessment
- Review, Reporting, and Learning services

These services will consume and produce versioned contracts without bypassing
service ownership boundaries.

## Development

Run the existing Go tests for the implemented services:

```bash
cd doclens-api-gateway && go test ./...
cd ../doclens-identity-service && go test ./...
cd ../doclens-document-intake-service && go test ./...
```

The engineering specification and service-specific build prompt are the
authoritative sources for service boundaries, data ownership, API contracts,
and event contracts.
