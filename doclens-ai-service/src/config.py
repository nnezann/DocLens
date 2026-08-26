"""Environment-driven configuration for the document processing service."""

from dataclasses import dataclass
import os


def _bool(name: str, default: bool = False) -> bool:
    value = os.getenv(name)
    return default if value is None else value.strip().lower() in {"1", "true", "yes", "on"}


def _int(name: str, default: int) -> int:
    try:
        return int(os.getenv(name, str(default)))
    except ValueError as exc:
        raise ValueError(f"{name} must be an integer") from exc


@dataclass(frozen=True)
class Settings:
    service_name: str = os.getenv("SERVICE_NAME", "doclens-document-processing")
    host: str = os.getenv("SERVICE_HOST", "0.0.0.0")
    port: int = _int("SERVICE_PORT", 8080)
    database_url: str | None = os.getenv("DATABASE_URL")
    database_schema: str = os.getenv("PROCESSING_DB_SCHEMA", "doclens_processing")
    rabbitmq_url: str | None = os.getenv("RABBITMQ_URL")
    rabbitmq_exchange: str = os.getenv("RABBITMQ_EXCHANGE", "doclens.events")
    rabbitmq_queue: str = os.getenv("PROCESSING_QUEUE", "doclens.document-processing")
    rabbitmq_routing_key: str = os.getenv("PROCESSING_ROUTING_KEY", "document.uploaded")
    rabbitmq_prefetch: int = _int("RABBITMQ_PREFETCH", 4)
    retry_max_attempts: int = _int("PROCESSING_MAX_RETRIES", 3)
    retry_backoff_seconds: float = float(os.getenv("PROCESSING_RETRY_BACKOFF_SECONDS", "1"))
    retry_backoff_max_seconds: float = float(os.getenv("PROCESSING_RETRY_BACKOFF_MAX_SECONDS", "30"))
    s3_endpoint: str | None = os.getenv("S3_ENDPOINT")
    s3_region: str = os.getenv("S3_REGION", "us-east-1")
    s3_access_key_id: str | None = os.getenv("S3_ACCESS_KEY_ID")
    s3_secret_access_key: str | None = os.getenv("S3_SECRET_ACCESS_KEY")
    s3_bucket: str | None = os.getenv("S3_BUCKET")
    s3_connect_timeout_seconds: int = _int("S3_CONNECT_TIMEOUT_SECONDS", 10)
    s3_read_timeout_seconds: int = _int("S3_READ_TIMEOUT_SECONDS", 60)
    enable_model_inference: bool = _bool("ENABLE_MODEL_INFERENCE", False)
    ocr_provider: str = os.getenv("OCR_PROVIDER", "disabled")
    model_url: str | None = os.getenv("MODEL_INFERENCE_URL")
    model_name: str | None = os.getenv("MODEL_NAME")
    model_timeout_seconds: int = _int("MODEL_INFERENCE_TIMEOUT_SECONDS", 60)
    max_input_bytes: int = _int("PROCESSING_MAX_INPUT_BYTES", 50 * 1024 * 1024)


settings = Settings()
