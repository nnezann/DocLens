import io
import unittest

from PIL import Image, ImageDraw, ImageEnhance

from doclens_fingerprint.hashing import (
    hamming_distance,
    normalize_image,
    phash_bits,
    sha256_hex,
    tlsh_digest,
)
import doclens_fingerprint.hashing as hashing


def image_bytes(brightness=1.0, quality=95):
    image = Image.new("RGB", (480, 640), "white")
    draw = ImageDraw.Draw(image)
    draw.rectangle((35, 35, 445, 605), outline="black", width=5)
    draw.text((75, 100), "DOC LENS 12345", fill="black")
    draw.line((75, 180, 400, 180), fill="black", width=3)
    image = ImageEnhance.Brightness(image).enhance(brightness)
    output = io.BytesIO()
    image.save(output, format="JPEG", quality=quality)
    return output.getvalue()


class HashingTests(unittest.TestCase):
    def test_sha256_known_input(self):
        self.assertEqual(sha256_hex(b"DocLens"), "79e52961a50fdc5ccb7092b9cc6610de56f60aeb8fd794c37aa4385d0b639044")

    def test_tlsh_skips_small_input(self):
        self.assertIsNone(tlsh_digest(b"small"))

    def test_tlsh_known_input_when_binding_is_installed(self):
        if hashing.tlsh is None:
            self.skipTest("TLSH binding is optional in minimal environments")
        self.assertEqual(
            tlsh_digest(bytes(range(256)) * 2),
            "T118F09524E6514D7D1F175ADC904E44DF554FCDE302C5002517F186D1C510294440ED1D",
        )

    def test_phash_known_constant_image(self):
        image = Image.new("L", (32, 32), 0)
        output = io.BytesIO()
        image.save(output, format="PNG")
        self.assertEqual(phash_bits(output.getvalue()), "0" * 64)

    def test_three_image_pair_scenarios_have_perceptual_evidence(self):
        same = image_bytes()
        rescan = image_bytes(quality=70)
        lighting = image_bytes(brightness=0.72, quality=85)
        for left, right in ((same, same), (same, rescan), (same, lighting)):
            left_hash = phash_bits(normalize_image(left))
            right_hash = phash_bits(normalize_image(right))
            self.assertEqual(len(left_hash), 64)
            self.assertEqual(len(right_hash), 64)
            self.assertLessEqual(hamming_distance(left_hash, right_hash), 20)

    def test_normalization_does_not_mutate_input(self):
        original = image_bytes()
        copy = bytes(original)
        normalized = normalize_image(original)
        self.assertEqual(original, copy)
        self.assertNotEqual(normalized, original)


if __name__ == "__main__":
    unittest.main()
