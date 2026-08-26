---
title: Communication architecture
---

| Boundary | Technology | Use |
| --- | --- | --- |
| Client → Gateway | REST/HTTPS | Public API and integrations |
| Gateway → services | gRPC | Typed, low-latency internal calls |
| Service → service | gRPC | Synchronous operations |
| Service → service | RabbitMQ | Asynchronous workflows and jobs |
| Background processing | RabbitMQ + workers | OCR, embeddings, fraud analysis, and reports |

Expensive processing must not block the external HTTP request lifecycle. Every network operation requires an explicit timeout.
