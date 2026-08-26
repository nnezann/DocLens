from __future__ import annotations

from typing import Protocol
from urllib.parse import urlparse

import boto3
from botocore.config import Config


class ObjectStore(Protocol):
    def get(self, ref: str) -> bytes: ...
    def put(self, ref: str, content: bytes, content_type: str = "image/png") -> None: ...


class S3ObjectStore:
    def __init__(self, endpoint: str, access_key: str, secret_key: str, bucket: str,
                 region: str = "auto", timeout: float = 10.0):
        self.bucket = bucket
        self.client = boto3.client(
            "s3", endpoint_url=endpoint or None, aws_access_key_id=access_key or None,
            aws_secret_access_key=secret_key or None, region_name=region,
            config=Config(connect_timeout=timeout, read_timeout=timeout, retries={"max_attempts": 2}),
        )

    def get(self, ref: str) -> bytes:
        bucket, key = self._location(ref)
        return self.client.get_object(Bucket=bucket, Key=key)["Body"].read()

    def put(self, ref: str, content: bytes, content_type: str = "image/png") -> None:
        bucket, key = self._location(ref)
        self.client.put_object(Bucket=bucket, Key=key, Body=content, ContentType=content_type)

    def _location(self, ref: str) -> tuple[str, str]:
        parsed = urlparse(ref)
        if parsed.scheme == "s3" and parsed.netloc:
            return parsed.netloc, parsed.path.lstrip("/")
        return self.bucket, ref
