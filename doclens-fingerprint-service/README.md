# DocLens Fingerprint Service

The Fingerprint Service is a deterministic Python AI service. It consumes the
Document Intake `DocumentUploaded` event from RabbitMQ, reads the raw image
from S3-compatible storage, creates a private normalized PNG, and stores
independent SHA-256, TLSH, and pHash evidence in PostgreSQL. The original
object is never changed. V1 deliberately does not contain embeddings or a
vector database.

## Technology choices

- **Python**: image processing is CPU-oriented and matches the repository's
  existing AI-service stack.
- **OpenCV/Pillow**: deterministic edge/perspective correction, CLAHE,
  denoising, and DCT pHash without learned models.
- **FastAPI**: small operational HTTP surface for liveness/readiness.
- **gRPC**: documented, strongly typed internal `GetFingerprint` and
  `FindDuplicates` APIs, plus standard gRPC health checking.
- **RabbitMQ**: asynchronous `document.uploaded` trigger and
  `fingerprint.created` outbox publication.
- **PostgreSQL**: service-owned fingerprints, inbox deduplication, and
  transactional outbox.
- **boto3**: S3-compatible object-store abstraction (AWS S3, MinIO, or R2).

## Run

```bash
cd doclens-fingerprint-service
python -m venv .venv && . .venv/bin/activate
pip install -r requirements.txt
PYTHONPATH=src python -m doclens_fingerprint
```

Required production settings are `DATABASE_URL`, `S3_BUCKET`, and the S3
credentials/endpoint. `RABBITMQ_URL` enables consumption and outbox publishing;
the exchange defaults to `doclens.events`. The service listens on gRPC
`:9004` and HTTP `0.0.0.0:8084` by default.

The local Docker Compose stack builds the service from this directory and
provides PostgreSQL, RabbitMQ, and RustFS through the shared `doclens` network.
For a container-only run:

```bash
docker build -t doclens-fingerprint:local .
docker run --rm -p 8084:8084 -p 9004:9004 \
  -e DATABASE_URL=postgresql://doclens:doclens@host.docker.internal:15432/doclens \
  doclens-fingerprint:local
```

## Internal gRPC API

The contract is `proto/doclens/fingerprint/v1/fingerprint.proto`:

- `GetFingerprint(document_id)` returns all persisted deterministic hashes and
  references.
- `FindDuplicates(document_id, limit)` returns exact matches and near-match
  candidates with separate `tlsh_distance` and `phash_distance` fields.
- The standard `grpc.health.v1.Health` service supports liveness/readiness
  probes.

Useful configuration:

| Variable | Default | Meaning |
| --- | --- | --- |
| `FINGERPRINT_GRPC_ADDR` | `:9004` | Internal gRPC listener |
| `FINGERPRINT_HTTP_ADDR` | `0.0.0.0:8084` | Health HTTP listener |
| `FINGERPRINT_TLSH_HAMMING_THRESHOLD` | `64` | Candidate threshold |
| `FINGERPRINT_PHASH_HAMMING_THRESHOLD` | `12` | Candidate threshold |
| `FINGERPRINT_TLSH_MIN_BYTES` | `256` | Skip TLSH below this size |
| `FINGERPRINT_EXTERNAL_TIMEOUT_SECONDS` | `10` | S3, RabbitMQ, and DB operation timeout |

The TLSH and pHash thresholds are **empirical tuning placeholders**, not
production decisions. They must be calibrated against representative
document-pair data. Each distance is reported independently; this service
does not issue fraud or risk verdicts.

## Events

The consumer accepts the Document Intake envelope:

```json
{"event_id":"evt_1","event_type":"DocumentUploaded","event_version":1,
 "occurred_at":"...","organization_id":"org_1","document_id":"doc_1",
 "payload":{"uploads":[{"storage_ref":"org_1/doc_1/page.jpg"}]}}
```

It publishes `FingerprintCreated` with the same envelope conventions and
routing key `fingerprint.created`. The event is written in the same
transaction as the fingerprint row and retried asynchronously. `event_id`
deduplicates consumed events and output events carry their own IDs.

## Tests and protobuf generation

```bash
PYTHONPATH=src python -m unittest discover -s tests -v
make proto
```

The checked-in bindings under `src/doclens/fingerprint/v1/` are generated
from `proto/doclens/fingerprint/v1/fingerprint.proto`; `make proto` is the
reproducible generation approach for future contract changes.

Health endpoints are `/health/live` and `/health/ready`; the gRPC health
service reports both the empty service name and
`doclens.fingerprint.v1.FingerprintService` as serving.
