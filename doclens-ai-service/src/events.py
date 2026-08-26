"""RabbitMQ event envelope and consumer/publisher implementation."""

import asyncio
import json
import logging
from time import perf_counter
from datetime import datetime, timezone
from typing import Any
from uuid import uuid4

from config import Settings
from logging_utils import organization_id, request_id, trace_id
from metrics import (
    ERRORS, EVENT_PUBLICATION_FAILURES, JOB_STATUS, QUEUE_RETRIES, STORAGE_FAILURES, UPLOAD_BYTES, UPLOADS,
)
from processor import DeterministicProcessor, serialize_result

logger = logging.getLogger(__name__)


def document_processed_event(organization: str, document: str, payload: dict[str, Any]) -> dict[str, Any]:
    return {
        "event_id": f"evt_{uuid4().hex}",
        "event_type": "DocumentProcessed",
        "event_version": 1,
        "occurred_at": datetime.now(timezone.utc).isoformat(),
        "organization_id": organization,
        "document_id": document,
        "payload": payload,
    }


class ProcessingHandler:
    def __init__(self, repository: Any, storage: Any, settings: Settings) -> None:
        self.repository = repository
        self.storage = storage
        self.settings = settings
        self.processor = DeterministicProcessor(settings)

    async def handle(self, event: dict[str, Any], headers: dict[str, Any] | None = None) -> dict[str, Any] | None:
        event_id = event.get("event_id")
        organization = event.get("organization_id")
        document = event.get("document_id")
        if not event_id or not organization or not document or event.get("event_type") != "DocumentUploaded":
            raise ValueError("invalid DocumentUploaded event envelope")
        if event.get("event_version") != 1:
            raise ValueError("unsupported DocumentUploaded event version")
        payload = event.get("payload") or {}
        upload_items = payload.get("uploads") or event.get("uploads") or []
        if not upload_items and payload.get("storage_ref"):
            upload_items = [payload]
        if not upload_items:
            raise ValueError("DocumentUploaded event has no uploads")

        request_token = request_id.set(str((headers or {}).get("request_id") or event.get("request_id") or event_id))
        trace_token = trace_id.set(str((headers or {}).get("trace_id") or event.get("trace_id") or ""))
        org_token = organization_id.set(organization)
        started_at = perf_counter()
        try:
            if not await self.repository.claim_event(event_id):
                return None
            job_id = await self.repository.start_job(event_id, organization, document)
            outputs = []
            try:
                for upload in upload_items:
                    storage_ref = upload.get("storage_ref")
                    if (
                        not storage_ref
                        or storage_ref.startswith("/")
                        or ".." in storage_ref.split("/")
                        or not storage_ref.startswith(f"{organization}/")
                    ):
                        raise ValueError("unsafe or missing storage_ref")
                    raw = await self.storage.read(storage_ref)
                    UPLOADS.inc()
                    UPLOAD_BYTES.inc(len(raw))
                    processed = self.processor.process(
                        raw, upload.get("filename", "document"), upload.get("content_type", "application/octet-stream")
                    )
                    output_ref = f"processing/{organization}/{document}/{event_id}/{upload.get('id', 'upload')}.json"
                    await self.storage.write(output_ref, serialize_result(processed), "application/json")
                    outputs.append({"upload_id": upload.get("id"), "storage_ref": storage_ref, "result_ref": output_ref, **processed})
                result_ref = outputs[0]["result_ref"] if len(outputs) == 1 else f"processing/{organization}/{document}/{event_id}/result.json"
                if len(outputs) > 1:
                    await self.storage.write(result_ref, serialize_result({"uploads": outputs}), "application/json")
                result = {"status": "processed", "document_id": document, "organization_id": organization,
                          "job_id": job_id, "result_ref": result_ref, "uploads": outputs}
                event_out = document_processed_event(organization, document, {
                    "document_type": payload.get("type"),
                    "status": "processed",
                    "job_id": job_id,
                    "result_ref": result_ref,
                    "uploads": [{"upload_id": item.get("upload_id"), "result_ref": item["result_ref"]} for item in outputs],
                    "model_inference_enabled": self.settings.enable_model_inference,
                })
                if request_id.get():
                    event_out["request_id"] = request_id.get()
                if trace_id.get():
                    event_out["trace_id"] = trace_id.get()
                await self.repository.complete_job(job_id, result, event_out)
                JOB_STATUS.labels(status="processed").inc()
                logger.info(
                    "document processed",
                    extra={"operation": "process", "resource_id": document,
                           "duration_ms": round((perf_counter() - started_at) * 1000, 2)},
                )
                return event_out
            except Exception as exc:
                await self.repository.fail_job(job_id, str(exc))
                await self.repository.release_event(event_id)
                JOB_STATUS.labels(status="failed").inc()
                STORAGE_FAILURES.inc() if "storage" in str(exc).lower() else ERRORS.labels(operation="process").inc()
                raise
        finally:
            request_id.reset(request_token)
            trace_id.reset(trace_token)
            organization_id.reset(org_token)


class RabbitPublisher:
    def __init__(self, connection: Any, settings: Settings) -> None:
        self.connection = connection
        self.settings = settings
        self.channel = None

    async def start(self) -> None:
        self.channel = await self.connection.channel()
        await self.channel.set_qos(prefetch_count=self.settings.rabbitmq_prefetch)
        self.exchange = await self.channel.declare_exchange(self.settings.rabbitmq_exchange, "topic", durable=True)

    async def publish_outbox(self, repository: Any) -> None:
        if not self.channel:
            return
        from aio_pika import Message, DeliveryMode
        for item in await repository.pending_outbox():
            try:
                await self.exchange.publish(
                    Message(json.dumps(item["payload"]).encode(), delivery_mode=DeliveryMode.PERSISTENT,
                            message_id=item["event_id"], headers={
                                "request_id": item["payload"].get("request_id", ""),
                                "trace_id": item["payload"].get("trace_id", ""),
                            }),
                    routing_key=item["routing_key"],
                )
                await repository.mark_outbox_published(item["event_id"])
            except Exception:
                EVENT_PUBLICATION_FAILURES.inc()
                attempts = int(item.get("attempts", 0)) + 1
                await repository.mark_outbox_retry(
                    item["event_id"], attempts,
                    min(self.settings.retry_backoff_seconds * (2 ** (attempts - 1)),
                        self.settings.retry_backoff_max_seconds),
                    attempts >= self.settings.retry_max_attempts,
                )
                logger.exception("event publication failed", extra={"operation": "publish_outbox"})


class RabbitConsumer:
    def __init__(self, handler: ProcessingHandler, repository: Any, settings: Settings) -> None:
        self.handler, self.repository, self.settings = handler, repository, settings
        self.connection = None
        self.channel = None
        self.publisher: RabbitPublisher | None = None
        self.publisher_task: asyncio.Task | None = None
        self.started = False

    async def start(self) -> None:
        if not self.settings.rabbitmq_url:
            return
        import aio_pika
        self.connection = await aio_pika.connect_robust(self.settings.rabbitmq_url, timeout=10)
        self.channel = await self.connection.channel()
        await self.channel.set_qos(prefetch_count=self.settings.rabbitmq_prefetch)
        exchange = await self.channel.declare_exchange(self.settings.rabbitmq_exchange, "topic", durable=True)
        dlx = await self.channel.declare_exchange(f"{self.settings.rabbitmq_exchange}.dlx", "topic", durable=True)
        dlq = await self.channel.declare_queue(f"{self.settings.rabbitmq_queue}.dlq", durable=True)
        await dlq.bind(dlx, routing_key=self.settings.rabbitmq_routing_key)
        queue = await self.channel.declare_queue(
            self.settings.rabbitmq_queue,
            durable=True,
            arguments={
                "x-dead-letter-exchange": dlx.name,
                "x-dead-letter-routing-key": self.settings.rabbitmq_routing_key,
            },
        )
        await queue.bind(exchange, routing_key=self.settings.rabbitmq_routing_key)
        await queue.consume(self._consume, no_ack=False)
        self.publisher = RabbitPublisher(self.connection, self.settings)
        await self.publisher.start()
        self.publisher_task = asyncio.create_task(self._publish_loop())
        self.started = True

    async def _publish_loop(self) -> None:
        while True:
            await self.publisher.publish_outbox(self.repository)
            await asyncio.sleep(1)

    async def _consume(self, message: Any) -> None:
        headers = message.headers or {}
        retries = int(headers.get("x-retry-count", 0))
        try:
            event = json.loads(message.body)
            await self.handler.handle(event, headers)
            await message.ack()
        except Exception:
            if retries < self.settings.retry_max_attempts:
                await asyncio.sleep(min(self.settings.retry_backoff_seconds * (2 ** retries),
                                        self.settings.retry_backoff_max_seconds))
                QUEUE_RETRIES.inc()
                headers["x-retry-count"] = retries + 1
                from aio_pika import Message, DeliveryMode
                await self.publisher.exchange.publish(
                    Message(
                        body=message.body, headers=headers, delivery_mode=DeliveryMode.PERSISTENT,
                        content_type=message.content_type, message_id=message.message_id,
                    ),
                    routing_key=self.settings.rabbitmq_routing_key,
                )
                await message.ack()
            else:
                await message.reject(requeue=False)
                logger.exception("message moved to dead-letter queue", extra={"operation": "consume"})

    async def close(self) -> None:
        if self.publisher_task:
            self.publisher_task.cancel()
            await asyncio.gather(self.publisher_task, return_exceptions=True)
        if self.connection:
            await self.connection.close()
