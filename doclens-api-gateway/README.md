# DocLens API Gateway

Custom Go API gateway for DocLens. It exposes REST endpoints to external clients and forwards requests to DocLens services over gRPC.

## Implemented REST Surface

| Method | Path | Upstream gRPC service |
| --- | --- | --- |
| `GET` | `/healthz` | gateway only |
| `GET` | `/readyz` | gRPC health checks |
| `GET` | `/metrics` | gateway only |
| `POST` | `/identity/login` | IdentityService.Login |
| `POST` | `/identity/users` | IdentityService.CreateUser |
| `POST` | `/documents` | DocumentIntakeService.CreateDocument |
| `GET` | `/documents/{id}` | DocumentIntakeService.GetDocument |
| `POST` | `/verifications` | VerificationService.StartVerification |
| `GET` | `/verifications/{id}` | VerificationService.GetVerification |

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `GATEWAY_ADDR` | `:8080` | REST listen address |
| `JWT_SECRET` | empty | Required unless `GATEWAY_AUTH_DISABLED=true` |
| `GATEWAY_AUTH_DISABLED` | `false` | Local development bypass |
| `REQUEST_TIMEOUT` | `10s` | Per-request upstream timeout |
| `RATE_LIMIT_RPS` | `20` | Per-client token refill rate |
| `RATE_LIMIT_BURST` | `40` | Per-client burst |
| `PUBLIC_RATE_LIMIT_RPS` | `5` | Anonymous-route token refill rate |
| `PUBLIC_RATE_LIMIT_BURST` | `10` | Anonymous-route burst |
| `IDENTITY_GRPC_ADDR` | `localhost:9001` | Identity service gRPC endpoint |
| `DOCUMENTS_GRPC_ADDR` | `localhost:9002` | Document intake gRPC endpoint |
| `VERIFICATION_GRPC_ADDR` | `localhost:9003` | Verification service gRPC endpoint |
| `GRPC_INSECURE` | `true` | Local plaintext gRPC. TLS wiring is the next production hardening step. |

## Run

```bash
GATEWAY_AUTH_DISABLED=true go run ./cmd/gateway
```

The protobuf contracts live under `proto/doclens`. Regenerate Go bindings with:

Identity signup, Google login, email verification, and password reset routes
are treated as explicit public routes by the authentication middleware as
their upstream RPCs are added.

```bash
protoc --go_out=. --go_opt=module=github.com/doclens/api-gateway \
  --go-grpc_out=. --go-grpc_opt=module=github.com/doclens/api-gateway \
  proto/doclens/identity/v1/identity.proto \
  proto/doclens/documents/v1/documents.proto \
  proto/doclens/verification/v1/verification.proto
```
