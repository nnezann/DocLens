from dataclasses import dataclass
from datetime import datetime
from typing import Optional


@dataclass(frozen=True)
class Fingerprint:
    document_id: str
    original_ref: str
    normalized_ref: str
    sha256: str
    tlsh: Optional[str]
    phash: str
    duplicate_of: Optional[str]
    created_at: datetime


@dataclass(frozen=True)
class DuplicateMatch:
    document_id: str
    tlsh_distance: Optional[int]
    phash_distance: Optional[int]
    exact_match: bool
    duplicate_of: Optional[str] = None
