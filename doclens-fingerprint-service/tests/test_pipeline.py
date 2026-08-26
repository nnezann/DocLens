import io
import unittest
from PIL import Image, ImageDraw, ImageEnhance

from doclens_fingerprint.config import Settings
from doclens_fingerprint.hashing import hamming_distance, tlsh_distance
from doclens_fingerprint.models import DuplicateMatch
from doclens_fingerprint.service import FingerprintProcessor


def sample_image(brightness=1.0, quality=90):
    image = Image.new("RGB", (480, 640), "white")
    draw = ImageDraw.Draw(image)
    draw.rectangle((35, 35, 445, 605), outline="black", width=5)
    draw.text((75, 100), "DOC LENS 12345", fill="black")
    image = ImageEnhance.Brightness(image).enhance(brightness)
    result = io.BytesIO()
    image.save(result, "JPEG", quality=quality)
    return result.getvalue()


class MemoryStore:
    def __init__(self, objects):
        self.objects = dict(objects)
        self.writes = {}

    def get(self, ref):
        return self.objects[ref]

    def put(self, ref, content, content_type="image/png"):
        self.writes[ref] = content


class MemoryRepository:
    def __init__(self):
        self.items = []
        self.events = []

    def find_candidates(self, current, limit):
        return [
            DuplicateMatch(
                document_id=item.document_id,
                tlsh_distance=tlsh_distance(current.tlsh, item.tlsh),
                phash_distance=hamming_distance(current.phash, item.phash),
                exact_match=current.sha256 == item.sha256,
            )
            for item in self.items[-limit:]
        ]

    def save_with_outbox(self, fingerprint, event, organization_id, event_id=None, inbox_event_id=None):
        self.items.append(fingerprint)
        self.events.append(event)


class PipelineTests(unittest.TestCase):
    def test_three_upload_pair_scenarios_persist_private_normalized_copies(self):
        objects = {
            "same.jpg": sample_image(),
            "rescan.jpg": sample_image(quality=65),
            "lighting.jpg": sample_image(brightness=0.72, quality=80),
        }
        storage, repository = MemoryStore(objects), MemoryRepository()
        # Fixture tuning only: production thresholds require representative samples.
        processor = FingerprintProcessor(storage, repository, Settings(phash_hamming_threshold=16))

        first = processor.process_upload("doc-same", "org", "same.jpg", "evt-same")
        exact = processor.process_upload("doc-rescan", "org", "same.jpg", "evt-rescan")
        rescan = processor.process_upload("doc-resave", "org", "rescan.jpg", "evt-resave")
        lighting = processor.process_upload("doc-lighting", "org", "lighting.jpg", "evt-lighting")

        self.assertIsNone(first.duplicate_of)
        self.assertEqual(exact.duplicate_of, "doc-same")
        self.assertEqual(rescan.duplicate_of, "doc-same")
        self.assertEqual(lighting.duplicate_of, "doc-same")
        self.assertEqual(len(repository.events), 4)
        self.assertEqual(len(storage.writes), 4)
        self.assertTrue(all(key.startswith("fingerprints/org/") for key in storage.writes))


if __name__ == "__main__":
    unittest.main()
