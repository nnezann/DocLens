---
id: overview
title: DocLens documentation
sidebar_label: Overview
slug: /
---

DocLens is an AI-powered document verification platform built as independently deployable microservices.

## Documentation layers

- **Architecture** describes service boundaries, ownership, communication, and data flow.
- **Service guides** explain each service's responsibility, APIs, configuration, and local operation.
- **Contracts** document REST, versioned protobuf/gRPC, and RabbitMQ event interfaces.
- **Operations** covers development, deployment, reliability, security, and observability.
- **ADRs** record decisions that affect multiple services.

The engineering source of truth is [`Doclens_Engineering_Specification.md`](https://github.com/nnezann/DocLens/blob/main/Doclens_Engineering_Specification.md). Service-specific build prompts remain in the [`prompts/`](https://github.com/nnezann/DocLens/tree/main/prompts) directory.

## Current platform shape

```text
External clients
      |
      v
 REST/HTTPS API Gateway
      |
      v
 Internal gRPC services
      |
      +--> PostgreSQL per service boundary
      +--> S3-compatible object storage
      +--> RabbitMQ events and workers
```
