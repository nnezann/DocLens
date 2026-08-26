"""Prometheus metrics with a dependency-free fallback for minimal local runs."""

try:
    from prometheus_client import Counter, Gauge, Histogram, generate_latest, CONTENT_TYPE_LATEST
except ImportError:  # pragma: no cover - used only before dependencies are installed
    CONTENT_TYPE_LATEST = "text/plain"

    class _Metric:
        def labels(self, **_: str) -> "_Metric":
            return self

        def inc(self, *_: object, **__: object) -> None:
            return None

        def observe(self, *_: object, **__: object) -> None:
            return None

        def set(self, *_: object, **__: object) -> None:
            return None

    Counter = Gauge = Histogram = lambda *args, **kwargs: _Metric()  # type: ignore

    def generate_latest() -> bytes:
        return b"# prometheus_client is not installed\n"


REQUESTS = Counter("processing_requests_total", "HTTP and processing requests", ["operation", "status"])
LATENCY = Histogram("processing_request_duration_seconds", "Operation latency", ["operation"])
ERRORS = Counter("processing_errors_total", "Processing errors", ["operation"])
UPLOAD_BYTES = Counter("processing_input_bytes_total", "Input bytes processed")
UPLOADS = Counter("processing_uploads_total", "Uploads processed")
STORAGE_FAILURES = Counter("processing_storage_failures_total", "Object storage failures")
EVENT_PUBLICATION_FAILURES = Counter("processing_event_publication_failures_total", "Event publication failures")
QUEUE_RETRIES = Counter("processing_queue_retries_total", "Messages retried")
JOB_STATUS = Gauge("processing_jobs", "Jobs by status", ["status"])
