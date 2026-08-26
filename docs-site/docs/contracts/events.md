---
title: Event contracts
---

DocLens initially uses RabbitMQ for reliable asynchronous jobs, acknowledgements, retries, and dead-letter queues.

Every event uses a stable envelope:

```json
{
  "event_id": "evt_123",
  "event_type": "DocumentUploaded",
  "event_version": 1,
  "occurred_at": "2026-08-25T12:00:00Z",
  "organization_id": "org_123",
  "document_id": "doc_456",
  "payload": {}
}
```

Consumers must route by `event_type` and `event_version`, and deduplicate by `event_id` or the broker message ID.

| Event | Routing key | Producer | Consumers |
| --- | --- | --- | --- |
| `DocumentUploaded` | `document.uploaded` | Document Intake | Processing, Fingerprint |
| `DocumentProcessed` | Defined by contract | Processing | Fingerprint, Rules, Fraud |
| `VerificationCompleted` | Defined by contract | Risk Assessment | Review, Reporting, Learning |

AsyncAPI definitions will be added here as event schemas are formalized.
