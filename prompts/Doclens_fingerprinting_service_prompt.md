# DocLens — Fingerprint Service Specification & Agent Build Prompt

**Status:** Design addendum to the DocLens Unified Engineering Blueprint
**Service:** Fingerprint Service
**Domain:** AI (per Service Catalog, Section 3 of the main spec)
**Position in pipeline:** Consumes raw uploaded image in parallel with Fraud Detection — does not block on, or get blocked by, Fraud Detection

---

## 1. Purpose

The Fingerprint Service creates durable, comparable representations of a document image so DocLens can identify exact duplicates, near-duplicates (re-scans, re-photographs, re-compressions), and template-reuse patterns — without becoming the system of record for fraud or risk decisions.

It owns its own internal normalization step. Normalization exists **only inside this service** — it is not a shared upstream stage, and it must never be applied to the image before Fraud Detection or Document Processing receive it. Those two services consume the raw/original image directly from Document Intake's `DocumentUploaded` event.

---

## 2. Pipeline Placement

```
Document Intake
      │
      ▼  DocumentUploaded (raw storage_ref)
      │
      ├──► Document Processing   (OCR/extraction — raw or its own minimal preprocessing)
      │
      ├──► Fraud Detection        (raw image, untouched — forensic artifacts must survive)
      │
      └──► Fingerprint Service
              │
              ├─ Stage 1: Normalization (internal to this service only)
              │
              └─ Stage 2: Hashing (deterministic) + Stage 3: Embedding (probabilistic, optional/future)
```

**Why this ordering matters:** perspective correction, lighting normalization, and re-compression smooth out exactly the low-level artifacts Fraud Detection depends on (JPEG grid inconsistencies, noise-pattern/PRNU signals, moiré patterns, lighting-discontinuity seams from splicing). Normalizing the image before Fraud Detection sees it would destroy evidence. Fingerprint Service and Fraud Detection therefore both consume the same raw image independently and in parallel — there is no dependency in either direction between them.

---

## 3. Stage 1 — Normalization (internal only)

Goal: reduce two independent captures of the same physical document (or two scans/re-saves) to a comparable canonical form, without altering what either capture actually shows.

| Step                            | Technique options                                                                                                                                            |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Document/edge detection         | Canny edge detection + contour finding; Hough line transform as refinement                                                                                   |
| Perspective correction          | Homography estimation (4-point transform) from detected quadrilateral to flat rectangle                                                                      |
| Deskew                          | Minimum-area bounding box rotation correction, or Radon transform                                                                                            |
| Aspect-ratio correction         | Against known document-type dimensions from Knowledge Service templates, where available                                                                     |
| Lighting/exposure normalization | Grayscale conversion + CLAHE (preferred over global histogram equalization for uneven lighting); adaptive/Otsu thresholding if a binarized variant is needed |
| Noise handling                  | Light Gaussian/median blur pre-hash; optional light sharpening after, to avoid over-smoothing structure the hashes rely on                                   |

Output: a `normalized_ref` object-storage artifact, distinct from the `original_ref`/`storage_ref` already owned by Document Intake. Fingerprint Service never overwrites or mutates the original.

---

## 4. Stage 2 — Deterministic Hashing

| Purpose                       | Technique                                                                           | Notes                                                                                                                                                                                               |
| ----------------------------- | ----------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Exact duplicate               | SHA-256 / BLAKE3                                                                    | Run on raw bytes (`original_ref`), not the normalized image. Catches only byte-identical uploads. Also useful for chain-of-custody.                                                                 |
| Binary/fuzzy similarity       | TLSH (recommended default)                                                          | Robust to re-save/re-compression; weak on independent photo re-captures. Fixed-length digest, indexable.                                                                                            |
| Binary/fuzzy similarity (alt) | ssdeep, sdhash                                                                      | Alternatives if TLSH underperforms on your document formats; ssdeep needs a minimum ~50-byte input, TLSH needs ~256 bytes.                                                                          |
| Perceptual similarity         | pHash (recommended default)                                                         | DCT-based; robust to resize, mild recompression, minor crop. Run on `normalized_ref`. Compare via Hamming distance (empirically tune threshold, ~10–15 bits is a starting point).                   |
| Perceptual similarity (alt)   | dHash, aHash, wHash, Block Mean Value hash                                          | dHash cheaper/less rotation-robust; aHash useful only as a fast pre-filter; wHash sometimes more compression-robust than pHash.                                                                     |
| Structural/text-level         | MinHash / SimHash over n-grams of OCR text, with LSH banding for retrieval at scale | Requires Document Processing's extracted text as input (separate consumption of `DocumentProcessed`). Catches "same template, one field changed" forgery patterns byte/perceptual hashing can miss. |

Each hash type is stored and compared independently — do not collapse to a single score. Treat agreement across multiple deterministic signals as stronger evidence than any single hash matching.

---

## 5. Stage 3 — Probabilistic Layer (optional, future upgrade path)

- CNN/ViT visual embedding (e.g. ResNet/EfficientNet feature extractor or CLIP-style encoder) computed on `normalized_ref`, stored in pgvector, compared via cosine similarity.
- Adds recall specifically on the hardest case — independent photo-vs-photo captures under real-world lighting/angle/blur noise — where deterministic perceptual hashing degrades.
- Must carry `embedding_model_version` so re-embedding is possible without ambiguity when the model changes (matches the data model already defined in the main DocLens spec).
- Treated strictly as additional evidence, not a verdict — same rule as every other AI output in DocLens (Section 2.5 of the main spec).

This stage is explicitly **not required for V1**. It should be designed for later without being built now, per the project's "introduce specialized/probabilistic infrastructure only when justified" principle (ADR-002 pattern applied to this service).

---

## 6. Data Model

```text
fingerprints
  document_id
  original_ref
  normalized_ref
  sha256
  tlsh
  phash
  dhash                  (optional)
  text_minhash            (optional, requires OCR output)
  embedding                (optional, future — pgvector)
  embedding_model_version  (optional, future)
  duplicate_of             (nullable document_id)
  created_at
```

## 7. Events

**Consumes:**

```text
DocumentUploaded    (raw image — triggers normalization + hashing)
DocumentProcessed    (optional — only if text_minhash is implemented)
```

**Produces:**

```text
FingerprintCreated
```

```json
{
  "event_id": "evt_125",
  "event_type": "FingerprintCreated",
  "event_version": 1,
  "document_id": "doc_456",
  "hashes": {
    "sha256": "...",
    "tlsh": "...",
    "phash": "..."
  },
  "duplicate_of": null,
  "timestamp": "..."
}
```

Consumers of `FingerprintCreated` remain unchanged from the main spec: Fraud Detection.

---

## 8. Explicit Design Boundaries (do not silently resolve)

- Whether embeddings (Stage 3) are built in V1 or deferred to Phase 2/3 — **deferred by default**; do not build without an explicit decision.
- Hamming-distance thresholds for pHash/TLSH matches are **not fixed in this document** — they must be empirically tuned against real document samples before being treated as production thresholds. Do not hardcode an arbitrary threshold and call it final.
- Whether `duplicate_of` triggers any automatic action (e.g. auto-flag) is a Risk Assessment decision, not a Fingerprint Service decision. This service only reports evidence.

---

# 9. Build Prompt — For the AI Agent Implementing This Service

> Copy everything below this line as the instruction set for the coding agent building the Fingerprint Service.

```
You are implementing the Fingerprint Service for DocLens, a document-verification
platform. This service is domain "AI" but its Stage 1 and Stage 2 work must be
fully deterministic and explainable — no learned model weights in V1.

CONTEXT
- This service consumes DocumentUploaded events from RabbitMQ (routing key
  document.uploaded) containing a storage_ref to the raw uploaded image in
  S3-compatible object storage.
- It must NOT modify or overwrite the original file at storage_ref.
- It runs in parallel with Fraud Detection and Document Processing — it has no
  dependency on their output, and nothing depends on it blocking. Do not add
  a synchronous call to either service.

WHAT TO BUILD (in order)
1. A normalization step, internal to this service only:
   a. Detect document edges (Canny + contour finding, Hough line refinement
      as fallback) and correct perspective via homography to a flat rectangle.
   b. Deskew the result.
   c. Normalize lighting via CLAHE on a grayscale conversion.
   d. Apply light denoising (small-kernel Gaussian or median blur) — do not
      over-smooth; the hashing stage depends on preserved structure.
   e. Write the result to a new normalized_ref key in object storage. Never
      overwrite original_ref.

2. Deterministic hashing, computed and stored per document:
   a. SHA-256 of the raw bytes at original_ref (exact-duplicate detection).
   b. TLSH of the raw bytes at original_ref (fuzzy/binary similarity —
      re-saves, re-compression). If TLSH's minimum input-size requirement
      (~256 bytes) is not met, log and skip rather than erroring the pipeline.
   c. pHash of the image at normalized_ref (perceptual similarity — the
      primary signal for re-scans/re-photographs). Store as a bit-string
      suitable for Hamming-distance comparison.
   d. Do NOT implement a probabilistic embedding model in this pass unless
      explicitly instructed to in a follow-up task. If asked to add one
      later, treat it as an additive Stage 3 — do not remove or replace
      the deterministic hashes.

3. Persist results to the fingerprints table (document_id, original_ref,
   normalized_ref, sha256, tlsh, phash, duplicate_of, created_at) in this
   service's own PostgreSQL schema. Do not query any other service's tables
   directly.

4. Duplicate/similarity lookup:
   a. Exact duplicate: query for matching sha256.
   b. Near-duplicate candidates: retrieve prior fingerprints and compute
      Hamming distance on tlsh and phash independently. Do NOT collapse
      these into one combined score inside this service — report both
      distances as separate fields. Downstream consumers (Fraud Detection,
      Risk Assessment) decide how to weight agreement between them.
   c. Leave the actual match threshold as a configurable value (env var or
      config table), not a hardcoded constant. Do not invent a "final"
      threshold — flag in code comments that it requires empirical tuning
      against real sample data before being trusted in production.

5. Publish a FingerprintCreated event (same outbox-pattern approach already
   used by Document Intake: write the event in the same transaction as the
   fingerprints row, publish asynchronously with retry/backoff, include
   event_id for consumer-side deduplication) with the hash values and
   duplicate_of (nullable) in the payload.

RULES YOU MUST FOLLOW
- Do not access another service's database directly.
- Do not introduce a vector database or embedding model in this pass —
  pgvector/embeddings are an explicitly deferred future stage.
- Do not apply normalization to any image consumed by Fraud Detection or
  Document Processing — normalization is scoped to this service's internal
  pipeline only, operating on its own copy of the image.
- Do not make this service a source of truth for duplicate/fraud verdicts —
  it reports hash evidence and distances only.
- Every consumer-facing event must be idempotent-safe (use event_id).
- Every external call (object storage, RabbitMQ, Postgres) must have an
  explicit timeout.
- Emit structured JSON logs; do not log raw document bytes or full hash
  collision details that could leak content.
- If you hit an open decision not resolved in this spec (e.g. exact
  Hamming-distance threshold, whether to add dHash as a secondary
  perceptual hash, embedding model choice), do not silently choose —
  implement the extension point but leave the decision configurable and
  flag it clearly in a comment or README, not buried in code.

DELIVERABLE
A runnable service (Python, matching DocLens's AI-service stack) exposing:
- Internal gRPC or event-only interface — confirm against Document Intake's
  existing gRPC pattern.
- Health check endpoint (readiness + liveness).
- Structured logs with request_id/trace_id propagation.
- Unit tests for the hashing functions (known-input/known-output pairs).
- Integration tests for the normalization pipeline against at least 3 sample
  image pairs: (a) same file re-uploaded, (b) same page re-scanned, (c) same
  physical page photographed twice under different lighting.
```

---

## 10. Summary for Human Readers

In plain terms: this service takes a raw uploaded image, makes its own private "cleaned up" copy purely for comparison purposes, computes several independent similarity fingerprints on both the original and the cleaned copy, and reports those as evidence — never a verdict. It runs alongside Fraud Detection, not before or after it, so neither service's input is ever altered by the other's needs.
