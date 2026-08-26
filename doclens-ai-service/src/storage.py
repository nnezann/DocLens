"""S3-compatible object storage abstraction."""

import asyncio
from dataclasses import dataclass
from typing import Protocol

from config import Settings


class ObjectStorage(Protocol):
    async def read(self, storage_ref: str) -> bytes: ...
    async def write(self, storage_ref: str, data: bytes, content_type: str = "application/octet-stream") -> str: ...


@dataclass
class S3ObjectStorage:
    settings: Settings

    def __post_init__(self) -> None:
        import boto3
        from botocore.config import Config

        if not self.settings.s3_bucket:
            raise ValueError("S3_BUCKET is required for S3 object storage")
        self.bucket = self.settings.s3_bucket
        self.client = boto3.client(
            "s3",
            endpoint_url=self.settings.s3_endpoint,
            region_name=self.settings.s3_region,
            aws_access_key_id=self.settings.s3_access_key_id,
            aws_secret_access_key=self.settings.s3_secret_access_key,
            config=Config(
                connect_timeout=self.settings.s3_connect_timeout_seconds,
                read_timeout=self.settings.s3_read_timeout_seconds,
                retries={"max_attempts": 2, "mode": "standard"},
            ),
        )

    async def read(self, storage_ref: str) -> bytes:
        response = await asyncio.to_thread(self.client.get_object, Bucket=self.bucket, Key=storage_ref)
        return await asyncio.to_thread(response["Body"].read)

    async def write(self, storage_ref: str, data: bytes, content_type: str = "application/octet-stream") -> str:
        await asyncio.to_thread(
            self.client.put_object,
            Bucket=self.bucket,
            Key=storage_ref,
            Body=data,
            ContentType=content_type,
        )
        return storage_ref


class MemoryObjectStorage:
    """Useful for contract tests and local development without a cloud provider."""

    def __init__(self) -> None:
        self.objects: dict[str, tuple[bytes, str]] = {}

    async def read(self, storage_ref: str) -> bytes:
        return self.objects[storage_ref][0]

    async def write(self, storage_ref: str, data: bytes, content_type: str = "application/octet-stream") -> str:
        self.objects[storage_ref] = (data, content_type)
        return storage_ref
