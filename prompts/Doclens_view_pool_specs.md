# DocLens — Document View Separation & Document Intake Upload Pool

**Status:** Design addendum to the DocLens Unified Engineering Blueprint
**Primary service:** Document Intake (upload pool / concurrency management)
**Cross-cutting concern:** Document view separation (affects Document Intake, Document Processing, Fraud Detection, Fingerprint Service, Knowledge Service)
11
---

## Part A — Separating Views of an Uploaded Document

### 1. Problem

Multiple services need access to "the document," but they need it in different forms, for different purposes, and **none of them should have to wait for another to finish preparing its copy**:

- Fraud Detection needs the raw, untouched bytes — any transformation destroys forensic signal.
- Fingerprint Service needs its own normalized copy (perspective-corrected, lighting-normalized) — internal to itself, per the Fingerprint Service spec.
- Document Processing needs the raw bytes for OCR, and produces structured extracted fields as its own output.
- Verification frequently also needs a **known-authentic reference** — a trusted exemplar of what a genuine document of this type/issuer should look like — to cross-check the submitted document against, independent of anything derived from the submission itself.

Treating "the document" as one mutable file that different services take turns transforming creates exactly the coupling and blocking DocLens is designed to avoid (Section 2.1, 2.5 of the main spec). The fix is to make **views** an explicit, first-class concept instead of an implicit side effect of processing order.

### 2. View Types

| View | Owner | Mutable? | Consumers |
|---|---|---|---|
| `original_ref` | Document Intake | No — write-once | Fraud Detection, Document Processing, Fingerprint Service (as hashing input), audit/chain-of-custody |
| `normalized_ref` | Fingerprint Service | No — write-once, derived | Fingerprint Service internally only (per its own spec) |
| `extracted_view` (structured fields) | Document Processing | No — write-once per processing run | Rules Engine, Fraud Detection, Fingerprint Service (text-hash stage) |
| `reference_ref` (known-authentic exemplar) | Knowledge Service | No — versioned, not tied to any single submission | Fraud Detection, Rules Engine, Verification Engine |

**Core rule:** every view is a distinct, immutable, independently-addressable artifact with its own storage key or record. No service ever overwrites another service's view, and no service blocks waiting for another service's view to exist unless that view is genuinely a required input (e.g. Fraud Detection *may* optionally use `extracted_view` once available, but must be able to run its image-level checks without it).

### 3. Cross-Checking Against a Known-Authentic Reference

This is a new responsibility for **Knowledge Service**, extending its existing "Trusted reference information" role (Section 4.6 of the main spec) from structural templates (field lists, formats, checksums) to include **reference imagery/specimens**.

```
Knowledge Service
      │
      ├─ existing: templates, field formats, checksums, issuer rules (versioned)
      │
      └─ new: reference_ref per (document_type, issuer, template_version)
              — a trusted exemplar image/layout, NOT tied to any single
              submitted document
```

**API addition:**
```http
GET /knowledge/templates/{document_type}/reference
```
Returns a `reference_ref` (object-storage pointer) plus the template version it corresponds to, following the same versioning discipline already required of Knowledge Service (Section 4.6: *"A verification performed in 2027 must remain explainable using the reference data that existed when the verification occurred"*).

**How consumers use it:**
- Fraud Detection may fetch `reference_ref` alongside the submitted `original_ref` to perform layout/template-diffing (e.g., comparing field positions, security-feature placement) — this is a *comparison*, not a transformation of the submission, so it does not conflict with the "raw bytes only" rule.
- Rules Engine already consumes Knowledge Service reference data (formats, checksums); this extends the same relationship to imagery.
- The submitted document's own fingerprints (from Fingerprint Service) are a *separate* concern — duplicate/similarity detection against **other submissions**, not against the authentic reference. Do not conflate the two: "is this similar to a previously submitted document" and "does this match what an authentic document of this type looks like" are different questions with different evidence.

### 4. Parallel, Non-Blocking Multi-Consumer Access

This is largely already correct in the main spec's event model and should be preserved explicitly rather than accidentally lost during implementation:

- Document Intake publishes `DocumentUploaded` to a RabbitMQ **topic exchange** (`doclens.events`, routing key `document.uploaded`).
- Document Processing, Fraud Detection, and Fingerprint Service each bind their **own queue** to that routing key. This is fanout-via-topic-exchange, not a single shared queue — each consumer gets its own copy of the event and processes independently.
- All three then perform their own independent `GET` against object storage for `original_ref`. Object storage supports concurrent reads natively; there is no lock, no turn-taking, and no service waits on another's read or transformation to complete.
- The only genuine dependency in the graph is Fraud Detection optionally consuming `FingerprintCreated` (already in the main spec, Section 4.8) — that is evidence-enrichment, not a blocking prerequisite. Fraud Detection must be able to produce a result using only `original_ref` if `FingerprintCreated` hasn't arrived yet, and should be designed to attach fingerprint evidence opportunistically/asynchronously rather than stalling.

---

## Part B — Document Intake Upload Pool / Concurrency Manager

### 5. Problem

Document Intake is the single entry point for uploads across many organizations and users simultaneously. It must not let upload traffic overwhelm its own compute, its database connection pool, or its object-storage throughput — while still accepting bursts without arbitrarily rejecting legitimate traffic.

### 6. Recommended Pattern — Direct-to-Storage Upload (primary path)

The scalable default is to get large file bytes **off the Document Intake service's own request path entirely**:

```
1. Client → POST /documents/{id}/uploads/intent
     Document Intake creates an `uploads` row (status=pending),
     generates a pre-signed S3-compatible PUT URL, returns it.

2. Client → PUT (pre-signed URL) → Object Storage directly.
     Document Intake's compute and network are not involved in the
     byte transfer at all.

3. Client → POST /documents/{id}/uploads/{upload_id}/complete
     (or an object-storage event notification, if the provider supports it)
     Document Intake performs a HEAD against the object to confirm it
     exists and matches expected size/checksum, then — in a single
     transaction — writes the final `uploads` row state and an
     `event_outbox` row, and returns success to the client.

4. Background publisher sends DocumentUploaded from the outbox, per the
   existing outbox pattern already implemented in Document Intake.
```

This removes upload *volume* (bytes) as a scaling concern for the service's own compute/memory — Document Intake only ever handles small JSON requests and metadata, regardless of how many files or how large they are. Scaling upload throughput becomes an object-storage/CDN concern, which is what S3-compatible storage is built for.

### 7. Fallback / Proxied Upload Path (when direct upload isn't available)

Some clients or deployments won't support pre-signed direct upload (e.g. constrained integrations). For that path, Document Intake must bound its own concurrency explicitly rather than accepting unlimited simultaneous uploads:

- **Bounded worker pool**: a fixed-size pool of upload-handling goroutines (Go), sized to the service's available memory/network budget — not "one goroutine per request" unbounded.
- **Admission control / backpressure**: when the pool is saturated, new upload requests get an explicit `429 Too Many Requests` (with `Retry-After`) rather than queueing indefinitely in memory or timing out silently.
- **Per-tenant rate limiting**: token-bucket limiter keyed by `organization_id`, separate from the global pool limit, so one noisy tenant cannot starve others.
- **Streaming, not buffering**: proxied uploads must stream bytes to object storage as they arrive (`io.Copy`-style streaming, multipart upload API for large files) rather than buffering the full file in memory before forwarding.
- **Separate resource budgets**: the upload-handling worker pool, the database connection pool, and the object-storage client's connection pool must each have their own independent size limits — do not let upload concurrency implicitly determine DB connection count.
- **Circuit breaker to object storage**: if object storage starts failing/timing out, the pool should trip a circuit breaker and fail fast (with retries at the client/queue level) rather than letting all pool workers hang waiting on a degraded dependency.

### 8. Observability for the Pool

Minimum metrics (in addition to the metrics already required in Section 11 of the main spec):

```text
intake_pool_active_workers
intake_pool_queue_depth        (if any bounded queue exists ahead of the pool)
intake_pool_rejected_total     (429s issued due to saturation)
intake_upload_bytes_streamed_total
intake_presigned_url_issued_total
intake_upload_confirm_latency_ms
```

### 9. Explicit Design Boundaries (do not silently resolve)

- Whether object-storage event notifications (S3-compatible bucket notifications) are used instead of a client-driven `.../complete` confirmation call is a deployment-specific decision — both must be supported behind the same internal interface, not hardcoded to one provider's notification mechanism.
- The exact pool size, rate-limit values, and circuit-breaker thresholds are operational tuning values, not architecture — they must be configurable (env/config), not hardcoded constants.
- Whether pre-signed URLs are the default for *all* clients or only offered above a size threshold (e.g. small metadata-only clients might still proxy) is left open — implement both paths behind the same `uploads` state machine so either can be the default without a rewrite.

---

## 10. Data Model Additions

```text
uploads
  upload_id
  document_id
  status              (pending | uploaded | confirmed | failed)
  storage_ref          (original_ref)
  upload_method        (presigned_direct | proxied)
  checksum
  content_type
  size_bytes
  created_at
  confirmed_at

knowledge_reference_specimens   (new, owned by Knowledge Service)
  document_type
  issuer
  template_version
  reference_ref
  created_at
```

---

# 11. Build Prompt — For the AI Agent Implementing This

> Copy everything below this line as the instruction set for the coding agent.

```
You are extending the Document Intake service (Go) and Knowledge Service
(reference data owner) for DocLens. Two things to implement.

═══════════════════════════════════════════════════
PART 1 — DOCUMENT VIEW SEPARATION
═══════════════════════════════════════════════════

CONTEXT
- Multiple downstream services (Document Processing, Fraud Detection,
  Fingerprint Service) each need a specific, distinct "view" of an uploaded
  document. They must never share a mutable copy, never wait on each other,
  and never have their input silently altered by another service's needs.

WHAT TO BUILD
1. Ensure Document Intake's original_ref is treated as strictly write-once.
   No code path anywhere in the system may PUT/overwrite the object at
   original_ref after initial upload confirmation.
2. Confirm (or add, if missing) that the DocumentUploaded event fans out via
   a topic exchange with independent per-consumer queues — not a single
   shared queue — so Document Processing, Fraud Detection, and Fingerprint
   Service each get their own delivery and process fully independently and
   concurrently. Do not add any synchronous call between these three
   services, and do not make any of them wait on another's completion
   before starting its own work against original_ref.
3. In Knowledge Service, add a reference-specimen concept: a
   knowledge_reference_specimens table keyed by (document_type, issuer,
   template_version), each row pointing to a reference_ref in object
   storage. Add endpoint:
     GET /knowledge/templates/{document_type}/reference
   returning the reference_ref and template_version. This must follow the
   same versioning discipline as existing templates — never delete or
   overwrite a prior template_version's reference_ref.
4. Do NOT let Fraud Detection or any other consumer treat reference_ref
   and a submitted document's own original_ref/normalized_ref as the same
   kind of thing. reference_ref answers "what should an authentic document
   look like"; original_ref/normalized_ref answer "what was submitted."
   Keep these as clearly separate fields/parameters wherever both appear
   in code — never merge them into one generic "image" field.

RULES
- Do not add a dependency where Fraud Detection blocks on FingerprintCreated
  before producing a result — it must be able to run on original_ref alone,
  and attach fingerprint evidence opportunistically if/when it arrives.
- Do not let Document Processing, Fraud Detection, or Fingerprint Service
  write to original_ref under any circumstance.

═══════════════════════════════════════════════════
PART 2 — DOCUMENT INTAKE UPLOAD POOL / CONCURRENCY MANAGER
═══════════════════════════════════════════════════

CONTEXT
- Document Intake must handle uploads from many organizations and users
  concurrently without unbounded resource growth, while not needlessly
  rejecting legitimate burst traffic.

WHAT TO BUILD (in order)
1. Primary path — direct-to-storage upload:
   a. POST /documents/{id}/uploads/intent — create an uploads row
      (status=pending), generate a pre-signed PUT URL against the
      configured object-storage backend (R2/S3/local adapter — reuse the
      existing object-storage abstraction, do not add a second one).
      Return the URL and upload_id to the client.
   b. Client uploads directly to storage — no Document Intake code is on
      this path.
   c. POST /documents/{id}/uploads/{upload_id}/complete — verify the
      object exists via HEAD, verify size/checksum against what the client
      declared at intent time, then in a single DB transaction: update the
      uploads row to confirmed and write the event_outbox row (reuse the
      existing outbox pattern already implemented for DocumentUploaded).
   d. Also implement an object-storage-event-notification listener as an
      alternative confirmation trigger, behind the same internal interface,
      so either mechanism can confirm an upload without duplicating logic.

2. Fallback path — proxied upload (for clients that can't do pre-signed
   direct upload):
   a. Implement a bounded worker pool (fixed-size goroutine pool via a
      semaphore or worker-pool library) for handling proxied upload
      requests. Size must be configurable, not hardcoded.
   b. When the pool is saturated, return 429 with a Retry-After header —
      do not queue requests unboundedly in memory and do not let them hang
      past their context timeout.
   c. Add a per-organization token-bucket rate limiter, independent of the
      global pool limit.
   d. Stream request bytes directly to object storage (multipart upload
      API for large files) — never buffer a full file into memory before
      forwarding.
   e. Give the upload worker pool, the DB connection pool, and the
      object-storage client pool independent, separately configurable size
      limits. Do not let one implicitly bound another.
   f. Wrap object-storage calls in a circuit breaker; on repeated failure,
      fail fast rather than letting pool workers block on a degraded
      dependency.

3. Emit these metrics at minimum:
   intake_pool_active_workers, intake_pool_queue_depth,
   intake_pool_rejected_total, intake_upload_bytes_streamed_total,
   intake_presigned_url_issued_total, intake_upload_confirm_latency_ms

RULES
- Every new external call (object storage, DB) must have an explicit
  timeout.
- Do not hardcode pool size, rate-limit values, or circuit-breaker
  thresholds — these must be configuration, not constants in code.
- Do not remove or bypass the existing outbox-pattern event publishing —
  extend it, don't replace it.
- Structured JSON logs for every state transition in the uploads state
  machine (pending → uploaded → confirmed / failed), including
  organization_id and request_id, but never log document bytes or full
  checksums of sensitive content beyond what's needed for verification.

DELIVERABLE
- Updated Document Intake service with both upload paths behind the same
  uploads state machine.
- Updated Knowledge Service with the reference-specimen table and endpoint.
- Unit tests for: pool saturation/backpressure behavior, checksum mismatch
  handling on upload confirmation, rate-limiter per-tenant isolation.
- Integration tests for: full presigned-upload-to-confirmed flow, full
  proxied-upload-to-confirmed flow, and a concurrent-upload load test
  hitting the pool's configured limit to confirm 429s are returned rather
  than unbounded queueing or crashes.
```

---

## 12. Summary for Human Readers

Two related fixes, one document: first, stop treating "the uploaded document" as a single mutable file — split it into explicit, immutable views (raw, normalized, extracted, and a new authentic-reference view from Knowledge Service) so every consumer gets exactly what it needs without waiting on or corrupting what another consumer needs. Second, make Document Intake itself scale to many concurrent uploads by getting file bytes off its own request path via pre-signed direct-to-storage uploads, with a bounded, rate-limited, circuit-broken fallback path for clients that can't use that pattern.
# DocLens — Document View Separation & Document Intake Upload Pool

**Status:** Design addendum to the DocLens Unified Engineering Blueprint
**Primary service:** Document Intake (upload pool / concurrency management)
**Cross-cutting concern:** Document view separation (affects Document Intake, Document Processing, Fraud Detection, Fingerprint Service, Knowledge Service)

---

## Part A — Separating Views of an Uploaded Document

### 1. Problem

Multiple services need access to "the document," but they need it in different forms, for different purposes, and **none of them should have to wait for another to finish preparing its copy**:

- Fraud Detection needs the raw, untouched bytes — any transformation destroys forensic signal.
- Fingerprint Service needs its own normalized copy (perspective-corrected, lighting-normalized) — internal to itself, per the Fingerprint Service spec.
- Document Processing needs the raw bytes for OCR, and produces structured extracted fields as its own output.
- Verification frequently also needs a **known-authentic reference** — a trusted exemplar of what a genuine document of this type/issuer should look like — to cross-check the submitted document against, independent of anything derived from the submission itself.

Treating "the document" as one mutable file that different services take turns transforming creates exactly the coupling and blocking DocLens is designed to avoid (Section 2.1, 2.5 of the main spec). The fix is to make **views** an explicit, first-class concept instead of an implicit side effect of processing order.

### 2. View Types

| View | Owner | Mutable? | Consumers |
|---|---|---|---|
| `original_ref` | Document Intake | No — write-once | Fraud Detection, Document Processing, Fingerprint Service (as hashing input), audit/chain-of-custody |
| `normalized_ref` | Fingerprint Service | No — write-once, derived | Fingerprint Service internally only (per its own spec) |
| `extracted_view` (structured fields) | Document Processing | No — write-once per processing run | Rules Engine, Fraud Detection, Fingerprint Service (text-hash stage) |
| `reference_ref` (known-authentic exemplar) | Knowledge Service | No — versioned, not tied to any single submission | Fraud Detection, Rules Engine, Verification Engine |

**Core rule:** every view is a distinct, immutable, independently-addressable artifact with its own storage key or record. No service ever overwrites another service's view, and no service blocks waiting for another service's view to exist unless that view is genuinely a required input (e.g. Fraud Detection *may* optionally use `extracted_view` once available, but must be able to run its image-level checks without it).

### 3. Cross-Checking Against a Known-Authentic Reference

This is a new responsibility for **Knowledge Service**, extending its existing "Trusted reference information" role (Section 4.6 of the main spec) from structural templates (field lists, formats, checksums) to include **reference imagery/specimens**.

```
Knowledge Service
      │
      ├─ existing: templates, field formats, checksums, issuer rules (versioned)
      │
      └─ new: reference_ref per (document_type, issuer, template_version)
              — a trusted exemplar image/layout, NOT tied to any single
              submitted document
```

**API addition:**
```http
GET /knowledge/templates/{document_type}/reference
```
Returns a `reference_ref` (object-storage pointer) plus the template version it corresponds to, following the same versioning discipline already required of Knowledge Service (Section 4.6: *"A verification performed in 2027 must remain explainable using the reference data that existed when the verification occurred"*).

**How consumers use it:**
- Fraud Detection may fetch `reference_ref` alongside the submitted `original_ref` to perform layout/template-diffing (e.g., comparing field positions, security-feature placement) — this is a *comparison*, not a transformation of the submission, so it does not conflict with the "raw bytes only" rule.
- Rules Engine already consumes Knowledge Service reference data (formats, checksums); this extends the same relationship to imagery.
- The submitted document's own fingerprints (from Fingerprint Service) are a *separate* concern — duplicate/similarity detection against **other submissions**, not against the authentic reference. Do not conflate the two: "is this similar to a previously submitted document" and "does this match what an authentic document of this type looks like" are different questions with different evidence.

### 4. Parallel, Non-Blocking Multi-Consumer Access

This is largely already correct in the main spec's event model and should be preserved explicitly rather than accidentally lost during implementation:

- Document Intake publishes `DocumentUploaded` to a RabbitMQ **topic exchange** (`doclens.events`, routing key `document.uploaded`).
- Document Processing, Fraud Detection, and Fingerprint Service each bind their **own queue** to that routing key. This is fanout-via-topic-exchange, not a single shared queue — each consumer gets its own copy of the event and processes independently.
- All three then perform their own independent `GET` against object storage for `original_ref`. Object storage supports concurrent reads natively; there is no lock, no turn-taking, and no service waits on another's read or transformation to complete.
- The only genuine dependency in the graph is Fraud Detection optionally consuming `FingerprintCreated` (already in the main spec, Section 4.8) — that is evidence-enrichment, not a blocking prerequisite. Fraud Detection must be able to produce a result using only `original_ref` if `FingerprintCreated` hasn't arrived yet, and should be designed to attach fingerprint evidence opportunistically/asynchronously rather than stalling.

---

## Part B — Document Intake Upload Pool / Concurrency Manager

### 5. Problem

Document Intake is the single entry point for uploads across many organizations and users simultaneously. It must not let upload traffic overwhelm its own compute, its database connection pool, or its object-storage throughput — while still accepting bursts without arbitrarily rejecting legitimate traffic.

### 6. Recommended Pattern — Direct-to-Storage Upload (primary path)

The scalable default is to get large file bytes **off the Document Intake service's own request path entirely**:

```
1. Client → POST /documents/{id}/uploads/intent
     Document Intake creates an `uploads` row (status=pending),
     generates a pre-signed S3-compatible PUT URL, returns it.

2. Client → PUT (pre-signed URL) → Object Storage directly.
     Document Intake's compute and network are not involved in the
     byte transfer at all.

3. Client → POST /documents/{id}/uploads/{upload_id}/complete
     (or an object-storage event notification, if the provider supports it)
     Document Intake performs a HEAD against the object to confirm it
     exists and matches expected size/checksum, then — in a single
     transaction — writes the final `uploads` row state and an
     `event_outbox` row, and returns success to the client.

4. Background publisher sends DocumentUploaded from the outbox, per the
   existing outbox pattern already implemented in Document Intake.
```

This removes upload *volume* (bytes) as a scaling concern for the service's own compute/memory — Document Intake only ever handles small JSON requests and metadata, regardless of how many files or how large they are. Scaling upload throughput becomes an object-storage/CDN concern, which is what S3-compatible storage is built for.

### 7. Fallback / Proxied Upload Path (when direct upload isn't available)

Some clients or deployments won't support pre-signed direct upload (e.g. constrained integrations). For that path, Document Intake must bound its own concurrency explicitly rather than accepting unlimited simultaneous uploads:

- **Bounded worker pool**: a fixed-size pool of upload-handling goroutines (Go), sized to the service's available memory/network budget — not "one goroutine per request" unbounded.
- **Admission control / backpressure**: when the pool is saturated, new upload requests get an explicit `429 Too Many Requests` (with `Retry-After`) rather than queueing indefinitely in memory or timing out silently.
- **Per-tenant rate limiting**: token-bucket limiter keyed by `organization_id`, separate from the global pool limit, so one noisy tenant cannot starve others.
- **Streaming, not buffering**: proxied uploads must stream bytes to object storage as they arrive (`io.Copy`-style streaming, multipart upload API for large files) rather than buffering the full file in memory before forwarding.
- **Separate resource budgets**: the upload-handling worker pool, the database connection pool, and the object-storage client's connection pool must each have their own independent size limits — do not let upload concurrency implicitly determine DB connection count.
- **Circuit breaker to object storage**: if object storage starts failing/timing out, the pool should trip a circuit breaker and fail fast (with retries at the client/queue level) rather than letting all pool workers hang waiting on a degraded dependency.

### 8. Observability for the Pool

Minimum metrics (in addition to the metrics already required in Section 11 of the main spec):

```text
intake_pool_active_workers
intake_pool_queue_depth        (if any bounded queue exists ahead of the pool)
intake_pool_rejected_total     (429s issued due to saturation)
intake_upload_bytes_streamed_total
intake_presigned_url_issued_total
intake_upload_confirm_latency_ms
```

### 9. Explicit Design Boundaries (do not silently resolve)

- Whether object-storage event notifications (S3-compatible bucket notifications) are used instead of a client-driven `.../complete` confirmation call is a deployment-specific decision — both must be supported behind the same internal interface, not hardcoded to one provider's notification mechanism.
- The exact pool size, rate-limit values, and circuit-breaker thresholds are operational tuning values, not architecture — they must be configurable (env/config), not hardcoded constants.
- Whether pre-signed URLs are the default for *all* clients or only offered above a size threshold (e.g. small metadata-only clients might still proxy) is left open — implement both paths behind the same `uploads` state machine so either can be the default without a rewrite.

---

## 10. Data Model Additions

```text
uploads
  upload_id
  document_id
  status              (pending | uploaded | confirmed | failed)
  storage_ref          (original_ref)
  upload_method        (presigned_direct | proxied)
  checksum
  content_type
  size_bytes
  created_at
  confirmed_at

knowledge_reference_specimens   (new, owned by Knowledge Service)
  document_type
  issuer
  template_version
  reference_ref
  created_at
```

---

# 11. Build Prompt — For the AI Agent Implementing This

> Copy everything below this line as the instruction set for the coding agent.

```
You are extending the Document Intake service (Go) and Knowledge Service
(reference data owner) for DocLens. Two things to implement.

═══════════════════════════════════════════════════
PART 1 — DOCUMENT VIEW SEPARATION
═══════════════════════════════════════════════════

CONTEXT
- Multiple downstream services (Document Processing, Fraud Detection,
  Fingerprint Service) each need a specific, distinct "view" of an uploaded
  document. They must never share a mutable copy, never wait on each other,
  and never have their input silently altered by another service's needs.

WHAT TO BUILD
1. Ensure Document Intake's original_ref is treated as strictly write-once.
   No code path anywhere in the system may PUT/overwrite the object at
   original_ref after initial upload confirmation.
2. Confirm (or add, if missing) that the DocumentUploaded event fans out via
   a topic exchange with independent per-consumer queues — not a single
   shared queue — so Document Processing, Fraud Detection, and Fingerprint
   Service each get their own delivery and process fully independently and
   concurrently. Do not add any synchronous call between these three
   services, and do not make any of them wait on another's completion
   before starting its own work against original_ref.
3. In Knowledge Service, add a reference-specimen concept: a
   knowledge_reference_specimens table keyed by (document_type, issuer,
   template_version), each row pointing to a reference_ref in object
   storage. Add endpoint:
     GET /knowledge/templates/{document_type}/reference
   returning the reference_ref and template_version. This must follow the
   same versioning discipline as existing templates — never delete or
   overwrite a prior template_version's reference_ref.
4. Do NOT let Fraud Detection or any other consumer treat reference_ref
   and a submitted document's own original_ref/normalized_ref as the same
   kind of thing. reference_ref answers "what should an authentic document
   look like"; original_ref/normalized_ref answer "what was submitted."
   Keep these as clearly separate fields/parameters wherever both appear
   in code — never merge them into one generic "image" field.

RULES
- Do not add a dependency where Fraud Detection blocks on FingerprintCreated
  before producing a result — it must be able to run on original_ref alone,
  and attach fingerprint evidence opportunistically if/when it arrives.
- Do not let Document Processing, Fraud Detection, or Fingerprint Service
  write to original_ref under any circumstance.

═══════════════════════════════════════════════════
PART 2 — DOCUMENT INTAKE UPLOAD POOL / CONCURRENCY MANAGER
═══════════════════════════════════════════════════

CONTEXT
- Document Intake must handle uploads from many organizations and users
  concurrently without unbounded resource growth, while not needlessly
  rejecting legitimate burst traffic.

WHAT TO BUILD (in order)
1. Primary path — direct-to-storage upload:
   a. POST /documents/{id}/uploads/intent — create an uploads row
      (status=pending), generate a pre-signed PUT URL against the
      configured object-storage backend (R2/S3/local adapter — reuse the
      existing object-storage abstraction, do not add a second one).
      Return the URL and upload_id to the client.
   b. Client uploads directly to storage — no Document Intake code is on
      this path.
   c. POST /documents/{id}/uploads/{upload_id}/complete — verify the
      object exists via HEAD, verify size/checksum against what the client
      declared at intent time, then in a single DB transaction: update the
      uploads row to confirmed and write the event_outbox row (reuse the
      existing outbox pattern already implemented for DocumentUploaded).
   d. Also implement an object-storage-event-notification listener as an
      alternative confirmation trigger, behind the same internal interface,
      so either mechanism can confirm an upload without duplicating logic.

2. Fallback path — proxied upload (for clients that can't do pre-signed
   direct upload):
   a. Implement a bounded worker pool (fixed-size goroutine pool via a
      semaphore or worker-pool library) for handling proxied upload
      requests. Size must be configurable, not hardcoded.
   b. When the pool is saturated, return 429 with a Retry-After header —
      do not queue requests unboundedly in memory and do not let them hang
      past their context timeout.
   c. Add a per-organization token-bucket rate limiter, independent of the
      global pool limit.
   d. Stream request bytes directly to object storage (multipart upload
      API for large files) — never buffer a full file into memory before
      forwarding.
   e. Give the upload worker pool, the DB connection pool, and the
      object-storage client pool independent, separately configurable size
      limits. Do not let one implicitly bound another.
   f. Wrap object-storage calls in a circuit breaker; on repeated failure,
      fail fast rather than letting pool workers block on a degraded
      dependency.

3. Emit these metrics at minimum:
   intake_pool_active_workers, intake_pool_queue_depth,
   intake_pool_rejected_total, intake_upload_bytes_streamed_total,
   intake_presigned_url_issued_total, intake_upload_confirm_latency_ms

RULES
- Every new external call (object storage, DB) must have an explicit
  timeout.
- Do not hardcode pool size, rate-limit values, or circuit-breaker
  thresholds — these must be configuration, not constants in code.
- Do not remove or bypass the existing outbox-pattern event publishing —
  extend it, don't replace it.
- Structured JSON logs for every state transition in the uploads state
  machine (pending → uploaded → confirmed / failed), including
  organization_id and request_id, but never log document bytes or full
  checksums of sensitive content beyond what's needed for verification.

DELIVERABLE
- Updated Document Intake service with both upload paths behind the same
  uploads state machine.
- Updated Knowledge Service with the reference-specimen table and endpoint.
- Unit tests for: pool saturation/backpressure behavior, checksum mismatch
  handling on upload confirmation, rate-limiter per-tenant isolation.
- Integration tests for: full presigned-upload-to-confirmed flow, full
  proxied-upload-to-confirmed flow, and a concurrent-upload load test
  hitting the pool's configured limit to confirm 429s are returned rather
  than unbounded queueing or crashes.
```

---

## 12. Summary for Human Readers

Two related fixes, one document: first, stop treating "the uploaded document" as a single mutable file — split it into explicit, immutable views (raw, normalized, extracted, and a new authentic-reference view from Knowledge Service) so every consumer gets exactly what it needs without waiting on or corrupting what another consumer needs. Second, make Document Intake itself scale to many concurrent uploads by getting file bytes off its own request path via pre-signed direct-to-storage uploads, with a bounded, rate-limited, circuit-broken fallback path for clients that can't use that pattern.
