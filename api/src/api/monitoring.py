"""
WebSocket endpoint for real-time telemetry streaming.

Clients connect to /api/v1/monitoring/stream and receive JSON events
as they arrive from the pipeline workers via RabbitMQ.
"""

import logging

from fastapi import APIRouter, WebSocket, WebSocketDisconnect, Depends

from src.core.protocols import TelemetryGateway
from src.api.dependencies import get_telemetry_consumer

logger = logging.getLogger("Monitoring")

monitoring_router = APIRouter(prefix="/api/v1/monitoring", tags=["Monitoring"])


@monitoring_router.websocket("/stream")
async def telemetry_stream(
    ws: WebSocket, 
    consumer: TelemetryGateway = Depends(get_telemetry_consumer)
) -> None:
    """
    WebSocket endpoint that streams real-time telemetry events.
    """
    await ws.accept()
    consumer.register(ws)
    logger.info("Client connected to telemetry stream")

    try:
        # Keep the WebSocket alive — client may send pings or messages
        while True:
            # We don't expect messages from the client, but we need to
            # keep receiving to detect disconnection and handle pings
            await ws.receive_text()
    except WebSocketDisconnect:
        logger.info("Client disconnected from telemetry stream")
    finally:
        consumer.unregister(ws)
