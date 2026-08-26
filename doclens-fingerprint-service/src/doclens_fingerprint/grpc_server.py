from __future__ import annotations

from concurrent import futures
from datetime import datetime, timezone
import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from doclens.fingerprint.v1 import fingerprint_pb2, fingerprint_pb2_grpc

from .logging import request_id, trace_id
from .service import FingerprintProcessor
from .models import Fingerprint


def _propagate_metadata(context: grpc.ServicerContext) -> None:
    metadata = dict(context.invocation_metadata())
    request_id.set(metadata.get("x-request-id", metadata.get("request-id", "")))
    trace_id.set(metadata.get("x-trace-id", metadata.get("traceparent", "")))


def _to_proto(item):
    return fingerprint_pb2.Fingerprint(
        document_id=item.document_id, original_ref=item.original_ref,
        normalized_ref=item.normalized_ref, sha256=item.sha256, tlsh=item.tlsh or "",
        phash=item.phash, duplicate_of=item.duplicate_of or "",
        created_at=item.created_at.isoformat(),
    )


class FingerprintServicer(fingerprint_pb2_grpc.FingerprintServiceServicer):
    def __init__(self, processor: FingerprintProcessor):
        self.processor = processor

    def GetFingerprint(self, request, context):
        _propagate_metadata(context)
        item = self.processor.repository.get(request.document_id)
        if not item:
            context.abort(grpc.StatusCode.NOT_FOUND, "fingerprint not found")
        return fingerprint_pb2.FingerprintResponse(fingerprint=_to_proto(item))

    def FindDuplicates(self, request, context):
        _propagate_metadata(context)
        current = None
        if request.document_id:
            current = self.processor.repository.get(request.document_id)
        elif request.sha256 or request.tlsh or request.phash:
            current = Fingerprint(
                document_id="", original_ref="", normalized_ref="",
                sha256=request.sha256, tlsh=request.tlsh or None, phash=request.phash,
                duplicate_of=None, created_at=datetime.now(timezone.utc),
            )
        if not current:
            context.abort(grpc.StatusCode.NOT_FOUND, "fingerprint not found")
        limit = min(max(request.limit or 100, 1), self.processor.settings.max_candidates)
        matches = self.processor.qualifying_matches(
            self.processor.repository.find_candidates(current, limit)
        )
        return fingerprint_pb2.DuplicateResults(matches=[
            fingerprint_pb2.DuplicateMatch(
                document_id=item.document_id,
                tlsh_distance=item.tlsh_distance or 0,
                phash_distance=item.phash_distance or 0,
                exact_match=item.exact_match,
                duplicate_of=item.duplicate_of or "",
            ) for item in matches
        ])


def create_grpc_server(processor: FingerprintProcessor, address: str, max_workers: int = 8):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))
    fingerprint_pb2_grpc.add_FingerprintServiceServicer_to_server(FingerprintServicer(processor), server)
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set("doclens.fingerprint.v1.FingerprintService",
                        health_pb2.HealthCheckResponse.SERVING)
    server.add_insecure_port(address)
    return server
