---
title: Identity Service
---

The Go gRPC Identity Service owns users, organizations, roles, permissions, sessions, and refresh tokens. It issues JWTs consumed by the API Gateway.

The current local implementation supports user creation, login, and gRPC health checks. See [`doclens-identity-service/README.md`](https://github.com/nnezann/DocLens/blob/main/doclens-identity-service/README.md) for configuration and local startup.
