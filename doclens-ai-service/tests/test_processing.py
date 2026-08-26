import asyncio
import hashlib
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parents[1] / "src"))

from config import Settings
from events import ProcessingHandler
from persistence import InMemoryRepository
from processor import DeterministicProcessor
from storage import MemoryObjectStorage


def run(coro):
    return asyncio.run(coro)


class ProcessingTests(unittest.TestCase):
    def test_deterministic_hash_and_image_metadata(self):
        processor = DeterministicProcessor(Settings())
        raw = b"stable document bytes"
        result = processor.process(raw, "doc.bin", "application/octet-stream")
        self.assertEqual(result["sha256"], hashlib.sha256(raw).hexdigest())
        self.assertEqual(result["deterministic"]["format"], "binary")
        self.assertFalse(result["model_inference"]["enabled"])

    def test_duplicate_delivery_is_idempotent_and_publishes_versioned_event(self):
        async def scenario():
            settings = Settings()
            storage = MemoryObjectStorage()
            await storage.write("org-a/raw/upload-1.bin", b"content")
            repository = InMemoryRepository()
            handler = ProcessingHandler(repository, storage, settings)
            event = {
                "event_id": "evt-1", "event_type": "DocumentUploaded", "event_version": 1,
                "organization_id": "org-a", "document_id": "doc-1",
                "payload": {"type": "unknown", "uploads": [{
                    "id": "upload-1", "storage_ref": "org-a/raw/upload-1.bin",
                    "filename": "upload-1.bin", "content_type": "application/octet-stream",
                }]},
            }
            first = await handler.handle(event)
            second = await handler.handle(event)
            return first, second, repository, storage

        first, second, repository, storage = run(scenario())
        self.assertEqual(first["event_type"], "DocumentProcessed")
        self.assertEqual(first["event_version"], 1)
        self.assertIsNone(second)
        self.assertEqual(len(repository.outbox), 1)
        self.assertTrue(any(key.startswith("processing/org-a/doc-1/") for key in storage.objects))

    def test_unsafe_storage_reference_fails_without_output(self):
        async def scenario():
            repository = InMemoryRepository()
            storage = MemoryObjectStorage()
            handler = ProcessingHandler(repository, storage, Settings())
            event = {
                "event_id": "evt-bad", "event_type": "DocumentUploaded", "event_version": 1,
                "organization_id": "org-a", "document_id": "doc-1",
                "payload": {"uploads": [{"storage_ref": "../secret", "filename": "x"}]},
            }
            with self.assertRaises(ValueError):
                await handler.handle(event)
            return repository, storage

        repository, storage = run(scenario())
        self.assertFalse(storage.objects)
        self.assertEqual(repository.jobs["job_evt-bad"]["status"], "failed")

    def test_trace_and_request_ids_are_propagated_to_output_event(self):
        async def scenario():
            repository = InMemoryRepository()
            storage = MemoryObjectStorage()
            await storage.write("org-a/raw/upload.bin", b"content")
            handler = ProcessingHandler(repository, storage, Settings())
            event = {
                "event_id": "evt-context", "event_type": "DocumentUploaded", "event_version": 1,
                "organization_id": "org-a", "document_id": "doc-1",
                "payload": {"uploads": [{"id": "u1", "storage_ref": "org-a/raw/upload.bin"}]},
            }
            return await handler.handle(event, {"request_id": "req-1", "trace_id": "trace-1"})

        output = run(scenario())
        self.assertEqual(output["request_id"], "req-1")
        self.assertEqual(output["trace_id"], "trace-1")


if __name__ == "__main__":
    unittest.main()
