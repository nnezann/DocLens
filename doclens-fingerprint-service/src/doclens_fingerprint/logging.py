import json
import logging
import sys
from contextvars import ContextVar
from typing import Any

request_id: ContextVar[str] = ContextVar("request_id", default="")
trace_id: ContextVar[str] = ContextVar("trace_id", default="")


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        item: dict[str, Any] = {
            "timestamp": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"),
            "level": record.levelname,
            "service": "fingerprint",
            "message": record.getMessage(),
        }
        rid, tid = request_id.get(), trace_id.get()
        if rid:
            item["request_id"] = rid
        if tid:
            item["trace_id"] = tid
        for key in ("document_id", "event_id", "has_tlsh", "reason", "size"):
            if key in record.__dict__:
                item[key] = record.__dict__[key]
        if record.exc_info:
            item["exception"] = self.formatException(record.exc_info)
        return json.dumps(item, separators=(",", ":"))


def configure_logging() -> logging.Logger:
    logger = logging.getLogger("doclens.fingerprint")
    if not logger.handlers:
        handler = logging.StreamHandler(sys.stdout)
        handler.setFormatter(JsonFormatter())
        logger.addHandler(handler)
        logger.setLevel(logging.INFO)
        logger.propagate = False
    return logger
