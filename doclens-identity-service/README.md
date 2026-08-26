# DocLens Identity Service

Go gRPC identity service for DocLens. It owns user authentication and emits gateway-compatible JWTs for the DocLens API Gateway.

## Implemented Surface

| RPC | Purpose |
| --- | --- |
| `CreateUser` | Creates an organization-scoped user with roles. |
| `Login` | Verifies credentials and returns an access token plus refresh token. |
| `grpc.health.v1.Health/Check` | Readiness check consumed by the API Gateway. |

Users and refresh tokens are persisted in PostgreSQL when `DATABASE_URL` is configured. The service applies the checked-in migration at startup. If `DATABASE_URL` is absent, an in-memory store is used only when `IDENTITY_ENV=development` (the default); non-development environments fail fast.

Refresh tokens are opaque to clients and stored as SHA-256 hashes in
PostgreSQL. Database writes are transactional and all database operations use
bounded contexts.

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `IDENTITY_GRPC_ADDR` | `:9001` | gRPC listen address. |
| `JWT_SECRET` | empty | Required. Must match the gateway's `JWT_SECRET`. |
| `IDENTITY_ENV` | `development` | Only development may run without `DATABASE_URL`. |
| `DATABASE_URL` | empty | PostgreSQL connection URL, for example `postgres://doclens:doclens@localhost:15432/doclens?sslmode=disable`. |
| `DATABASE_TIMEOUT` | `5s` | Timeout applied to PostgreSQL operations and startup migration. |
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

For a durable local run, start PostgreSQL and configure the service:

```bash
docker run --name doclens-postgres --rm \
  -e POSTGRES_DB=doclens -e POSTGRES_USER=doclens -e POSTGRES_PASSWORD=doclens \
  -p 15432:5432 postgres:16-alpine
DATABASE_URL='postgres://doclens:doclens@localhost:15432/doclens?sslmode=disable' \
JWT_SECRET=dev-secret go run ./cmd/identity
```

The service image can be built with the repository `Dockerfile` and works with
Compose by setting `DATABASE_URL` to the PostgreSQL service hostname (for
example `postgres://doclens:doclens@postgres:5432/doclens?sslmode=disable`).
PostgreSQL is the source of truth; Redis is not currently a dependency and is
deferred as a temporary acceleration layer for sessions, rate limits, and code
caching.

This change persists the users and refresh-token records used by the existing
`CreateUser` and `Login` RPCs. Refresh-token exchange/revocation, federation,
external providers, service authentication, and organization-management APIs
remain out of scope because they are not part of the current gRPC contract.

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
