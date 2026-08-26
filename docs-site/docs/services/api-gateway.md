---
title: API Gateway
---

The Go API Gateway is DocLens's unified external entry point. It exposes REST endpoints and forwards requests to internal services over gRPC.

Responsibilities include authentication and authorization enforcement, rate limiting, validation, request IDs, structured logging, and metrics.

See the implementation README for the current endpoint surface and configuration: [`doclens-api-gateway/README.md`](https://github.com/nnezann/DocLens/blob/main/doclens-api-gateway/README.md).
