"""Small structured logging and correlation context helpers."""

from contextvars import ContextVar
import json
import logging
import sys
from datetime import datetime, timezone
from typing import Any


request_id: ContextVar[str | None] = ContextVar("request_id", default=None)
trace_id: ContextVar[str | None] = ContextVar("trace_id", default=None)
organization_id: ContextVar[str | None] = ContextVar("organization_id", default=None)


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "service": getattr(self, "service_name", "doclens-document-processing"),
            "level": record.levelname,
            "message": record.getMessage(),
            "operation": getattr(record, "operation", record.name),
        }
        for key, value in (
            ("request_id", request_id.get()),
            ("trace_id", trace_id.get()),
            ("organization_id", organization_id.get()),
            ("resource_id", getattr(record, "resource_id", None)),
            ("duration_ms", getattr(record, "duration_ms", None)),
        ):
            if value is not None:
                payload[key] = value
        if record.exc_info:
            payload["exception"] = self.formatException(record.exc_info)
        return json.dumps(payload, separators=(",", ":"))


def configure_logging(service_name: str = "doclens-document-processing") -> None:
    handler = logging.StreamHandler(sys.stdout)
    formatter = JsonFormatter()
    formatter.service_name = service_name
    handler.setFormatter(formatter)
    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(logging.INFO)
