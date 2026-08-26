---
title: Document Intake Service
---

The Go gRPC Document Intake Service owns logical documents, physical uploads, and processing-job state. File bytes belong in S3-compatible object storage; PostgreSQL owns authoritative metadata.

It publishes `DocumentUploaded` through the RabbitMQ outbox flow and must not perform OCR, extraction, fingerprinting, fraud analysis, or final verification.

See [`doclens-document-intake-service/README.md`](https://github.com/nnezann/DocLens/blob/main/doclens-document-intake-service/README.md) for the RPC surface, configuration, and upload flow.
