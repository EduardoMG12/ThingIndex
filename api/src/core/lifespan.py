import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI

from src.services.telemetry_consumer import telemetry_consumer
from src.services.neo4j_client import neo4j_client

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

    # Connect to Neo4j database
    await neo4j_client.connect()
    
    yield

    # Shutdown
    await telemetry_consumer.stop()
    await neo4j_client.close()
    logger.info("API is shutting down...")

