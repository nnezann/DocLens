# DocLens documentation site

This directory contains the deployable DocLens documentation site. The
published site is generated under `build/`; its OpenAPI contract is available
at `build/openapi/openapi.yaml`.

## Try the microservices with Docker Compose

Run these commands from the repository root:

```bash
cp .env.local.example .env
docker compose -f docker-compose.local.yml up --build
```

The local stack starts PostgreSQL, RabbitMQ, RustFS, Identity, Document
Intake, the API Gateway, and a Verification stub. The service endpoints are:

| Service | Endpoint |
| --- | --- |
| API Gateway | `http://localhost:8080` |
| Document Intake metrics | `http://localhost:9092/metrics` |
| PostgreSQL | `localhost:15432` |
| RabbitMQ AMQP | `localhost:5672` |
| RabbitMQ management | `http://localhost:15672` |
| RustFS S3 API | `http://localhost:9000` |
| RustFS console | `http://localhost:9001` |

Check container state and gateway readiness:

```bash
docker compose -f docker-compose.local.yml ps
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

The seeded development administrator is
`admin@doclens.local` / `doclens-dev`. Obtain a token and create a document:

```bash
export DOC_LENS_TOKEN="$(curl -s http://localhost:8080/identity/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@doclens.local","password":"doclens-dev"}' |
  jq -r .access_token)"

curl -s http://localhost:8080/documents \
  -H "Authorization: Bearer $DOC_LENS_TOKEN" \
  -H 'content-type: application/json' \
  -d '{"type":"certificate","filename":"certificate.pdf",
       "content_type":"application/pdf","content_base64":"JVBERi0="}'
```

The document request demonstrates the gateway-to-Document Intake gRPC path.
Document metadata is persisted in PostgreSQL, bytes are written to RustFS,
and the `document.uploaded` event is published through RabbitMQ. The
Verification service is a local stub and intentionally returns an
unimplemented response.

Stop the stack with:

```bash
docker compose -f docker-compose.local.yml down
```

Add `-v` only when the local PostgreSQL, RabbitMQ, and RustFS volumes should
also be removed.

## OpenAPI documentation

The public REST contract is documented in
[`build/openapi/openapi.yaml`](build/openapi/openapi.yaml), an OpenAPI 3.1
document. It describes the gateway routes, request and response schemas, and
authentication requirements. The rendered documentation is available from the
site's **Contracts → REST API** page.

Keep the OpenAPI document synchronized with the gateway handlers when REST
routes or payloads change. Internal service-to-service calls use the versioned
gRPC contracts documented in the service pages and are not exposed as public
REST endpoints.

