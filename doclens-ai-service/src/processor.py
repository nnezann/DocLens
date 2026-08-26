"""Deterministic document processing with optional, explicitly disabled inference."""

from hashlib import sha256
import io
import json
from datetime import datetime, timezone
from typing import Any

from config import Settings


class DeterministicProcessor:
    def __init__(self, settings: Settings) -> None:
        self.settings = settings

    def process(self, raw: bytes, filename: str, content_type: str) -> dict[str, Any]:
        if len(raw) > self.settings.max_input_bytes:
            raise ValueError("input exceeds configured processing size limit")
        result: dict[str, Any] = {
            "filename": filename,
            "content_type": content_type,
            "size_bytes": len(raw),
            "sha256": sha256(raw).hexdigest(),
            "processed_at": datetime.now(timezone.utc).isoformat(),
            "deterministic": {"valid": True},
            "model_inference": {"enabled": False, "provider": self.settings.ocr_provider},
        }
        if content_type.startswith("image/"):
            result["deterministic"].update(self._image_metadata(raw))
        elif content_type == "application/pdf":
            result["deterministic"]["format"] = "pdf"
        else:
            result["deterministic"]["format"] = "binary"

        if self.settings.enable_model_inference:
            result["model_inference"] = self._optional_ocr(raw, filename)
        return result

    @staticmethod
    def _image_metadata(raw: bytes) -> dict[str, Any]:
        try:
            from PIL import Image
            from ela_check import run_ela_bytes
            with Image.open(io.BytesIO(raw)) as image:
                image.verify()
            with Image.open(io.BytesIO(raw)) as image:
                metadata = {
                    "format": image.format,
                    "width": image.width,
                    "height": image.height,
                    "mode": image.mode,
                }
            metadata["ela"] = run_ela_bytes(raw)
            return metadata
        except Exception as exc:
            return {"valid": False, "validation_error": str(exc)[:200]}

    @staticmethod
    def _optional_ocr(raw: bytes, filename: str) -> dict[str, Any]:
        # Provider selection is intentionally unresolved; no provider is enabled by default.
        if not filename:
            return {"enabled": True, "status": "skipped", "reason": "missing filename"}
        return {"enabled": True, "status": "skipped", "reason": "OCR provider is not configured"}


def serialize_result(result: dict[str, Any]) -> bytes:
    return json.dumps(result, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
