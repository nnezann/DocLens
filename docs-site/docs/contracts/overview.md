---
title: Contract documentation
---

Contracts are versioned and owned by the service that exposes or publishes them.

- Public HTTP endpoints are documented with OpenAPI.
- Internal APIs are defined in versioned protobuf packages under `proto/doclens`.
- RabbitMQ messages use stable event envelopes and explicit event versions.

Buf is the planned protobuf toolchain for formatting, linting, breaking-change detection, generated references, and schema publication. Contract artifacts should be generated from source definitions in CI rather than maintained as duplicated prose.
