import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI

from src.services.telemetry_consumer import telemetry_consumer

logger = logging.getLogger("CrawlerAPI")

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Print beautiful route log
    logger.info("=========================================")
    logger.info("🚀 CRAWLER API COMPLETED STARTUP 🚀")
    logger.info("=========================================")
    logger.info("Available Routes:")
    for route in app.routes:
        if hasattr(route, "methods"):
            # Exclude default internal endpoints if desired, but let's show all
            methods = ", ".join(route.methods)
            logger.info(f"  {methods: <7} -> {route.path}")
    logger.info("=========================================")
    logger.info("Interactive Docs (Scalar): http://[host]:[port]/scalar")

    # Start the telemetry consumer (connects to RabbitMQ and listens for events)
    await telemetry_consumer.start()
    
    yield

    # Shutdown
    await telemetry_consumer.stop()
    logger.info("API is shutting down...")

