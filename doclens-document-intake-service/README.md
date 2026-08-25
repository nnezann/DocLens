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
- Expose gRPC endpoints for creation and retrieval
- Provide health checks for readiness and liveness

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `DOCUMENT_INTAKE_GRPC_ADDR` | `:9002` | gRPC listening address |
| `DOCUMENT_INTAKE_STORAGE_DIR` | `./data/documents` | Local object-store directory |
| `DOCUMENT_INTAKE_MAX_UPLOAD_BYTES` | `10485760` | 10 MiB per upload |
| `DOCUMENT_INTAKE_ALLOWED_CONTENT_TYPES` | `application/pdf,image/jpeg,image/png,image/webp` | Allowed upload types |

## Run

```bash
cd doclens-document-intake-service
go run ./cmd/document-intake
```

The service uses a local object-store adapter by default so it works in local development without requiring cloud credentials. It is structured so a production S3-compatible implementation can replace the adapter behind the same interface without changing the service contract.
