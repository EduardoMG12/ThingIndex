"""
Async telemetry consumer that reads events from the RabbitMQ
'pipeline_telemetry' topic exchange and broadcasts them to connected
WebSocket clients.
"""

import asyncio
import json
import logging
from typing import Set

import aio_pika
from fastapi import WebSocket

from src.core.config import settings

logger = logging.getLogger("TelemetryConsumer")


class TelemetryConsumer:
    """Consumes telemetry events from RabbitMQ and broadcasts via WebSockets."""

    def __init__(self) -> None:
        self._clients: Set[WebSocket] = set()
        self._connection: aio_pika.abc.AbstractRobustConnection | None = None
        self._channel: aio_pika.abc.AbstractChannel | None = None
        self._task: asyncio.Task | None = None

    # -- Client management --------------------------------------------------

    def register(self, ws: WebSocket) -> None:
        self._clients.add(ws)
        logger.info("WebSocket client connected (%d total)", len(self._clients))

    def unregister(self, ws: WebSocket) -> None:
        self._clients.discard(ws)
        logger.info("WebSocket client disconnected (%d remaining)", len(self._clients))

    # -- Lifecycle -----------------------------------------------------------

    async def start(self) -> None:
        """Connect to RabbitMQ and start consuming in a background task."""
        rabbitmq_url = settings.rabbitmq_url
        max_retries = 10

        for attempt in range(1, max_retries + 1):
            try:
                self._connection = await aio_pika.connect_robust(rabbitmq_url)
                self._channel = await self._connection.channel()

                # Declare the topic exchange (must match what workers declare)
                exchange = await self._channel.declare_exchange(
                    settings.telemetry_exchange,
                    aio_pika.ExchangeType.TOPIC,
                    durable=False,
                )

                # Exclusive auto-delete queue — dies when the API disconnects
                queue = await self._channel.declare_queue(exclusive=True, auto_delete=True)

                # Bind to ALL telemetry events with wildcard routing key
                await queue.bind(exchange, routing_key="#")

                # Start consuming
                self._task = asyncio.create_task(self._consume(queue))
                logger.info("Telemetry consumer started on exchange '%s'", settings.telemetry_exchange)
                return  # success

            except Exception as e:
                if attempt < max_retries:
                    delay = min(2 ** attempt, 10)
                    logger.info("RabbitMQ not ready, retrying in %ds... (%d/%d)", delay, attempt, max_retries)
                    await asyncio.sleep(delay)
                else:
                    logger.warning("Could not start telemetry consumer after %d attempts: %s", max_retries, e)

    async def stop(self) -> None:
        """Gracefully shut down the consumer."""
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        if self._channel:
            await self._channel.close()
        if self._connection:
            await self._connection.close()
        logger.info("Telemetry consumer stopped")

    # -- Internal ------------------------------------------------------------

    async def _consume(self, queue: aio_pika.abc.AbstractQueue) -> None:
        """Read messages from the queue and broadcast to WebSocket clients."""
        async with queue.iterator() as queue_iter:
            async for message in queue_iter:
                async with message.process():
                    await self._broadcast(message.body.decode())

    async def _broadcast(self, payload: str) -> None:
        """Send a telemetry event to all connected WebSocket clients."""
        if not self._clients:
            return

        dead: list[WebSocket] = []
        for ws in self._clients:
            try:
                await ws.send_text(payload)
            except Exception:
                dead.append(ws)

        for ws in dead:
            self._clients.discard(ws)


# Singleton instance shared across the application
telemetry_consumer = TelemetryConsumer()
