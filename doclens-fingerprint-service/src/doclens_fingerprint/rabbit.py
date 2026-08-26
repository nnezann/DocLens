from __future__ import annotations

import asyncio
from datetime import datetime, timedelta, timezone
import json
import logging

import aio_pika
from aio_pika import ExchangeType, Message

from .config import Settings
from .events import DOCUMENT_PROCESSED
from .repository import PostgresRepository
from .service import FingerprintProcessor
from .logging import request_id, trace_id


async def consume_document_processed(processor: FingerprintProcessor, settings: Settings,
                                    logger: logging.Logger) -> None:
    if not settings.rabbitmq_url:
        logger.info("rabbitmq_disabled")
        return
    connection = await asyncio.wait_for(aio_pika.connect_robust(settings.rabbitmq_url), settings.external_timeout_seconds)
    channel = await connection.channel()
    exchange = await channel.declare_exchange(settings.rabbitmq_exchange, ExchangeType.TOPIC, durable=True)
    queue = await channel.declare_queue(settings.rabbitmq_queue, durable=True)
    await queue.bind(exchange, routing_key=DOCUMENT_PROCESSED)

    async def handle(message: aio_pika.abc.AbstractIncomingMessage) -> None:
        try:
            event = json.loads(message.body)
            request_id.set(event.get("request_id", ""))
            trace_id.set(event.get("trace_id", event.get("traceparent", "")))
            await asyncio.wait_for(asyncio.to_thread(processor.process_event, event),
                                   settings.external_timeout_seconds)
            await message.ack()
        except Exception:
            logger.exception("document_processed_processing_failed")
            await message.nack(requeue=True)

    await queue.consume(handle)
    try:
        await asyncio.Future()
    finally:
        await connection.close()


async def publish_outbox(repository: PostgresRepository, settings: Settings,
                          logger: logging.Logger) -> None:
    if not settings.rabbitmq_url:
        logger.info("outbox_publisher_disabled")
        return
    connection = await asyncio.wait_for(aio_pika.connect_robust(settings.rabbitmq_url), settings.external_timeout_seconds)
    channel = await connection.channel()
    exchange = await channel.declare_exchange(settings.rabbitmq_exchange, ExchangeType.TOPIC, durable=True)
    try:
        while True:
            try:
                records = await asyncio.wait_for(asyncio.to_thread(repository.claim_outbox, 50),
                                                 settings.external_timeout_seconds)
                for record in records:
                    try:
                        await asyncio.wait_for(exchange.publish(
                            Message(body=json.dumps(record["payload"]).encode(), content_type="application/json",
                                    delivery_mode=aio_pika.DeliveryMode.PERSISTENT,
                                    message_id=record["id"]),
                            routing_key=record["routing_key"]), settings.external_timeout_seconds)
                        await asyncio.wait_for(asyncio.to_thread(repository.mark_outbox_published, record["id"]),
                                               settings.external_timeout_seconds)
                    except Exception as exc:
                        delay = min(2 ** min(int(record["attempt_count"]), 6), 60)
                        await asyncio.to_thread(repository.mark_outbox_failed, record["id"], str(exc),
                                                datetime.now(timezone.utc) + timedelta(seconds=delay))
                        logger.exception("outbox_publish_failed", extra={"event_id": record["id"]})
            except asyncio.CancelledError:
                raise
            except Exception:
                logger.exception("outbox_poll_failed")
            await asyncio.sleep(settings.outbox_poll_seconds)
    finally:
        await connection.close()
