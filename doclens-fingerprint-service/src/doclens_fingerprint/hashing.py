"""Private normalization and deterministic hash functions."""

from __future__ import annotations

import hashlib
import logging
from typing import Optional

import cv2
import numpy as np
from PIL import Image
from io import BytesIO

try:
    import tlsh  # type: ignore
except ImportError:  # pragma: no cover - exercised in minimal local installs
    tlsh = None


TLSH_MIN_BYTES = 256


def sha256_hex(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def tlsh_digest(raw: bytes, minimum_size: int = TLSH_MIN_BYTES,
                logger: Optional[logging.Logger] = None) -> Optional[str]:
    if len(raw) < minimum_size:
        if logger:
            logger.info("tlsh_skipped", extra={"reason": "input_below_minimum", "size": len(raw)})
        return None
    if tlsh is None:
        if logger:
            logger.warning("tlsh_unavailable")
        return None
    digest = tlsh.hash(raw)
    return digest if digest and digest != "TNULL" else None


def _corners(image: np.ndarray) -> Optional[np.ndarray]:
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    edges = cv2.Canny(gray, 50, 150)
    contours, _ = cv2.findContours(edges, cv2.RETR_LIST, cv2.CHAIN_APPROX_SIMPLE)
    for contour in sorted(contours, key=cv2.contourArea, reverse=True)[:20]:
        perimeter = cv2.arcLength(contour, True)
        polygon = cv2.approxPolyDP(contour, 0.02 * perimeter, True)
        if len(polygon) == 4 and cv2.contourArea(polygon) > image.shape[0] * image.shape[1] * 0.2:
            return polygon.reshape(4, 2).astype(np.float32)
    return None


def _order_points(points: np.ndarray) -> np.ndarray:
    sums, diffs = points.sum(axis=1), np.diff(points, axis=1).ravel()
    return np.array([points[np.argmin(sums)], points[np.argmin(diffs)],
                     points[np.argmax(sums)], points[np.argmax(diffs)]], dtype=np.float32)


def _hough_deskew(image: np.ndarray) -> np.ndarray:
    """Use dominant Hough lines when no document quadrilateral is detectable."""
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    edges = cv2.Canny(gray, 50, 150)
    lines = cv2.HoughLinesP(edges, 1, np.pi / 180, threshold=80,
                            minLineLength=max(image.shape[:2]) // 4, maxLineGap=20)
    if lines is None:
        return image
    angles = []
    for line in lines[:, 0]:
        x1, y1, x2, y2 = line
        angle = np.degrees(np.arctan2(y2 - y1, x2 - x1))
        if abs(angle) <= 15 or abs(abs(angle) - 90) <= 15:
            angles.append(angle if abs(angle) <= 45 else angle - np.sign(angle) * 90)
    if not angles:
        return image
    angle = float(np.median(angles))
    center = (image.shape[1] / 2, image.shape[0] / 2)
    matrix = cv2.getRotationMatrix2D(center, angle, 1.0)
    return cv2.warpAffine(image, matrix, (image.shape[1], image.shape[0]),
                          flags=cv2.INTER_LINEAR, borderMode=cv2.BORDER_REPLICATE)


def normalize_image(raw: bytes) -> bytes:
    """Return a new PNG; the input bytes are never modified."""
    image = cv2.imdecode(np.frombuffer(raw, dtype=np.uint8), cv2.IMREAD_COLOR)
    if image is None:
        raise ValueError("raw object is not a decodable image")
    points = _corners(image)
    if points is not None:
        points = _order_points(points)
        width = int(max(np.linalg.norm(points[1] - points[0]), np.linalg.norm(points[2] - points[3])))
        height = int(max(np.linalg.norm(points[3] - points[0]), np.linalg.norm(points[2] - points[1])))
        if width >= 32 and height >= 32:
            target = np.array([[0, 0], [width - 1, 0], [width - 1, height - 1], [0, height - 1]], dtype=np.float32)
            image = cv2.warpPerspective(image, cv2.getPerspectiveTransform(points, target), (width, height))
    else:
        image = _hough_deskew(image)
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
    normalized = clahe.apply(gray)
    normalized = cv2.GaussianBlur(normalized, (3, 3), 0)
    ok, encoded = cv2.imencode(".png", normalized)
    if not ok:
        raise ValueError("could not encode normalized image")
    return encoded.tobytes()


def phash_bits(image_bytes: bytes, hash_size: int = 8, highfreq_factor: int = 4) -> str:
    image = Image.open(BytesIO(image_bytes)).convert("L")
    size = hash_size * highfreq_factor
    pixels = np.asarray(image.resize((size, size), Image.Resampling.LANCZOS), dtype=np.float32)
    dct = cv2.dct(pixels)
    low = dct[:hash_size, :hash_size]
    cutoff = np.median(low[1:, :])
    return "".join("1" if value > cutoff else "0" for value in low.flatten())


def hamming_distance(first: Optional[str], second: Optional[str]) -> Optional[int]:
    if not first or not second or len(first) != len(second):
        return None
    try:
        return sum(a != b for a, b in zip(first, second))
    except TypeError:
        return None


def tlsh_distance(first: Optional[str], second: Optional[str]) -> Optional[int]:
    """Compare TLSH digests independently; distance is digest bit Hamming distance."""
    if not first or not second:
        return None
    try:
        left = bytes.fromhex(first[2:] if first.startswith("T1") else first)
        right = bytes.fromhex(second[2:] if second.startswith("T1") else second)
    except ValueError:
        return None
    if len(left) != len(right):
        return None
    return sum((a ^ b).bit_count() for a, b in zip(left, right))
