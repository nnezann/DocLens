from dataclasses import dataclass
import os


def _int(name: str, default: int) -> int:
    try:
        return int(os.getenv(name, str(default)))
    except ValueError:
        return default


def _float(name: str, default: float) -> float:
    try:
        return float(os.getenv(name, str(default)))
    except ValueError:
        return default


@dataclass(frozen=True)
class Settings:
    grpc_addr: str = os.getenv("FINGERPRINT_GRPC_ADDR", ":9004")
    http_addr: str = os.getenv("FINGERPRINT_HTTP_ADDR", "0.0.0.0:8084")
    database_url: str = os.getenv("DATABASE_URL", "")
    rabbitmq_url: str = os.getenv("RABBITMQ_URL", "")
    rabbitmq_exchange: str = os.getenv("RABBITMQ_EXCHANGE", "doclens.events")
    rabbitmq_queue: str = os.getenv("FINGERPRINT_RABBITMQ_QUEUE", "doclens.fingerprint")
    s3_endpoint: str = os.getenv("S3_ENDPOINT", os.getenv("R2_ENDPOINT", ""))
    s3_access_key: str = os.getenv("S3_ACCESS_KEY_ID", os.getenv("R2_ACCESS_KEY_ID", ""))
    s3_secret_key: str = os.getenv("S3_SECRET_ACCESS_KEY", os.getenv("R2_SECRET_ACCESS_KEY", ""))
    s3_bucket: str = os.getenv("S3_BUCKET", os.getenv("R2_BUCKET", ""))
    s3_region: str = os.getenv("S3_REGION", "auto")
    tls_hamming_threshold: int = _int("FINGERPRINT_TLSH_HAMMING_THRESHOLD", 64)
    phash_hamming_threshold: int = _int("FINGERPRINT_PHASH_HAMMING_THRESHOLD", 12)
    max_candidates: int = _int("FINGERPRINT_MAX_CANDIDATES", 1000)
    tls_min_bytes: int = _int("FINGERPRINT_TLSH_MIN_BYTES", 256)
    external_timeout_seconds: float = _float("FINGERPRINT_EXTERNAL_TIMEOUT_SECONDS", 10.0)
    outbox_poll_seconds: float = _float("FINGERPRINT_OUTBOX_POLL_SECONDS", 2.0)

