import asyncio
from collections.abc import Callable
from fastapi import FastAPI, HTTPException
from fastapi import Request

from .logging import request_id, trace_id


def create_health_app(readiness_check: Callable[[], None] | None = None,
                      timeout_seconds: float = 2.0) -> FastAPI:
    app = FastAPI(title="DocLens Fingerprint Service", docs_url="/docs")

    @app.middleware("http")
    async def propagate_context(request: Request, call_next):
        request_id.set(request.headers.get("x-request-id", ""))
        trace_id.set(request.headers.get("x-trace-id", request.headers.get("traceparent", "")))
        return await call_next(request)

    @app.get("/health/live")
    async def live() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/health/ready")
    async def ready() -> dict[str, str]:
        if readiness_check:
            try:
                await asyncio.wait_for(asyncio.to_thread(readiness_check), timeout_seconds)
            except Exception as exc:
                raise HTTPException(status_code=503, detail="dependencies unavailable") from exc
        return {"status": "ok"}

    return app
