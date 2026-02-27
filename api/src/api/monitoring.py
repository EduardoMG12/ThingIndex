"""
WebSocket endpoint for real-time telemetry streaming.

Clients connect to /api/v1/monitoring/stream and receive JSON events
as they arrive from the pipeline workers via RabbitMQ.
"""

import logging

from fastapi import APIRouter, WebSocket, WebSocketDisconnect

from src.services.telemetry_consumer import telemetry_consumer

logger = logging.getLogger("Monitoring")

monitoring_router = APIRouter(prefix="/api/v1/monitoring", tags=["Monitoring"])


@monitoring_router.websocket("/stream")
async def telemetry_stream(ws: WebSocket) -> None:
    """
    WebSocket endpoint that streams real-time telemetry events.

    Each message is a JSON object:
    {
      "worker": "crawler" | "parser" | "populator",
      "stage": "<stage_id>",
      "event_type": "info" | "warning" | "error" | "success",
      "message": "...",
      "timestamp": "2026-02-27T11:16:28-03:00",
      "metadata": { "queue_messages": 42, ... }
    }
    """
    await ws.accept()
    telemetry_consumer.register(ws)
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
        telemetry_consumer.unregister(ws)
