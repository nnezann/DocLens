from __future__ import annotations

from datetime import datetime, timezone
import logging
from typing import Any, Optional

from .config import Settings
from .events import fingerprint_event, processed_uploads
from .hashing import normalize_image, phash_bits, sha256_hex, tlsh_digest
from .models import DuplicateMatch, Fingerprint
from .repository import PostgresRepository
from .storage import ObjectStore


class FingerprintProcessor:
    def __init__(self, store: ObjectStore, repository: PostgresRepository,
                 settings: Optional[Settings] = None, logger: Optional[logging.Logger] = None):
        self.store = store
        self.repository = repository
        self.settings = settings or Settings()
        self.logger = logger or logging.getLogger("doclens.fingerprint")

    def process_event(self, event: dict[str, Any]) -> list[Fingerprint]:
        organization_id = event.get("organization_id", "")
        results = []
        for index, upload in enumerate(processed_uploads(event)):
            document_id = event.get("document_id") or upload.get("document_id", "")
            if not document_id or not upload.get("storage_ref"):
                raise ValueError("DocumentProcessed is missing document_id or storage_ref")
            results.append(self.process_upload(
                document_id=document_id,
                organization_id=organization_id,
                original_ref=upload["storage_ref"],
                event_id=(f"{event['event_id']}:{upload.get('id', index)}"
                          if event.get("event_id") else None),
                request_id=event.get("request_id", ""),
                trace_id=event.get("trace_id", event.get("traceparent", "")),
            ))
        return results

    def process_upload(self, document_id: str, organization_id: str, original_ref: str,
                       event_id: Optional[str] = None, request_id: str = "",
                       trace_id: str = "") -> Fingerprint:
        raw = self.store.get(original_ref)
        normalized = normalize_image(raw)
        normalized_ref = f"fingerprints/{organization_id or 'unknown'}/{document_id}/normalized.png"
        self.store.put(normalized_ref, normalized)
        current = Fingerprint(
            document_id=document_id, original_ref=original_ref, normalized_ref=normalized_ref,
            sha256=sha256_hex(raw), tlsh=tlsh_digest(raw, self.settings.tls_min_bytes, self.logger),
            phash=phash_bits(normalized), duplicate_of=None, created_at=datetime.now(timezone.utc),
        )
        candidates = self.repository.find_candidates(current, self.settings.max_candidates)
        qualifying = self.qualifying_matches(candidates)
        exact = next((c for c in qualifying if c.exact_match), None)
        chosen = exact or (qualifying[0] if qualifying else None)
        if chosen:
            current = Fingerprint(**{**current.__dict__, "duplicate_of": chosen.document_id})
        output_event_id, payload = fingerprint_event(current, organization_id, [
            {"document_id": c.document_id, "tlsh_distance": c.tlsh_distance,
             "phash_distance": c.phash_distance, "exact_match": c.exact_match}
            for c in qualifying
        ], request_id=request_id, trace_id=trace_id)
        # The consumed event ID is the idempotency key; the produced event gets its own ID.
        self.repository.save_with_outbox(
            current, payload, organization_id, output_event_id, inbox_event_id=event_id
        )
        self.logger.info("fingerprint_created", extra={"document_id": document_id, "has_tlsh": bool(current.tlsh)})
        return current

    def lookup(self, document_id: str, limit: int = 100) -> tuple[Optional[Fingerprint], list[DuplicateMatch]]:
        current = self.repository.get(document_id)
        if not current:
            return None, []
        return current, self.qualifying_matches(self.repository.find_candidates(current, limit))

    def qualifying_matches(self, candidates: list[DuplicateMatch]) -> list[DuplicateMatch]:
        return [
            c for c in candidates
            if c.exact_match or
            (c.tlsh_distance is not None and c.tlsh_distance <= self.settings.tls_hamming_threshold) or
            (c.phash_distance is not None and c.phash_distance <= self.settings.phash_hamming_threshold)
        ]
