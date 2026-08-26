---
title: System context
---

DocLens separates external access, domain ownership, asynchronous processing, and AI-derived evidence.

```mermaid
flowchart LR
  Client[Clients] --> Gateway[API Gateway]
  Gateway --> Identity[Identity]
  Gateway --> Intake[Document Intake]
  Intake --> Rabbit[RabbitMQ]
  Rabbit --> Processing[Document Processing]
  Rabbit --> Fingerprint[Fingerprint]
  Processing --> Verification[Verification pipeline]
  Fingerprint --> Verification
  Verification --> Review[Review]
  Verification --> Reporting[Reporting]
```

AI services produce evidence, scores, and classifications. They do not become the system of record for final business outcomes unless the specification explicitly assigns that responsibility.
