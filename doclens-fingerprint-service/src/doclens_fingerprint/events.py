from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
import uuid


DOCUMENT_UPLOADED = "document.uploaded"
FINGERPRINT_CREATED = "fingerprint.created"


def uploaded_items(event: dict[str, Any]) -> list[dict[str, Any]]:
    payload = event.get("payload", event)
    uploads = payload.get("uploads", [])
    if not uploads and payload.get("storage_ref"):
        uploads = [payload]
    if isinstance(uploads, dict):
        uploads = [uploads]
    return uploads


def fingerprint_event(fingerprint: Any, organization_id: str, distances: list[dict[str, Any]],
                      request_id: str = "", trace_id: str = "") -> tuple[str, dict[str, Any]]:
    event_id = str(uuid.uuid4())
    occurred_at = datetime.now(timezone.utc).isoformat()
    envelope = {
        "event_id": event_id,
        "event_type": "FingerprintCreated",
        "event_version": 1,
        "occurred_at": occurred_at,
        "organization_id": organization_id,
        "document_id": fingerprint.document_id,
        "payload": {
            "hashes": {"sha256": fingerprint.sha256, "tlsh": fingerprint.tlsh, "phash": fingerprint.phash},
            "original_ref": fingerprint.original_ref,
            "normalized_ref": fingerprint.normalized_ref,
            "duplicate_of": fingerprint.duplicate_of,
            "distances": distances,
        },
    }
    if request_id:
        envelope["request_id"] = request_id
    if trace_id:
        envelope["trace_id"] = trace_id
    return event_id, envelope
