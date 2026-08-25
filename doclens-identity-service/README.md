# DocLens Identity Service

Go gRPC identity service for DocLens. It owns user authentication and emits gateway-compatible JWTs for the DocLens API Gateway.

Passwords are stored using Argon2id with a per-password random salt; plaintext
passwords are never persisted or logged.

## Implemented Surface

| RPC | Purpose |
| --- | --- |
| `CreateUser` | Creates an organization-scoped user with roles. |
| `Login` | Verifies credentials and returns an access token plus refresh token. |
| `grpc.health.v1.Health/Check` | Readiness check consumed by the API Gateway. |

This first slice uses an in-memory store so the gateway can be exercised locally. The service boundary is already shaped around users and refresh tokens, so PostgreSQL and Redis can be added behind `internal/store.Store` without changing the API.

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `IDENTITY_GRPC_ADDR` | `:9001` | gRPC listen address. |
| `JWT_SECRET` | empty | Required. Must match the gateway's `JWT_SECRET`. |
| `ACCESS_TOKEN_TTL` | `15m` | JWT lifetime. |
| `REFRESH_TOKEN_TTL` | `720h` | Refresh token lifetime. |
| `IDENTITY_DEV_SEED` | `true` | Seeds a local admin user on boot. |
| `IDENTITY_DEV_ORG_ID` | `dev-org` | Seed user organization. |
| `IDENTITY_DEV_EMAIL` | `admin@doclens.local` | Seed user email. |
| `IDENTITY_DEV_PASSWORD` | `doclens-dev` | Seed user password. |

## Run With The Gateway

Terminal 1:

```bash
JWT_SECRET=dev-secret go run ./cmd/identity
```

Terminal 2, from `/home/nezn/doclens-api-gateway`:

```bash
JWT_SECRET=dev-secret GATEWAY_AUTH_DISABLED=true go run ./cmd/gateway
```

Then log in through the gateway:

```bash
curl -s http://localhost:8080/identity/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@doclens.local","password":"doclens-dev"}'
```
