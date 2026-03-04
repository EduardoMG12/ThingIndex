"""
Neo4j connection lifecycle manager.
Handles connect, disconnect, and lazy reconnection.
"""
import logging
import asyncio
from neo4j import AsyncGraphDatabase
from src.core.config import settings

logger = logging.getLogger(__name__)


class Neo4jConnection:
    """Manages the async Neo4j driver lifecycle. Shared by all services."""

    def __init__(self):
        self.driver = None

    async def connect(self):
        """Initialize the async Neo4j driver with retries."""
        if not self.driver:
            for attempt in range(5):
                try:
                    if "/" in settings.neo4j_auth:
                        user, password = settings.neo4j_auth.split("/", 1)
                        auth = (user, password)
                    else:
                        auth = None

                    self.driver = AsyncGraphDatabase.driver(settings.neo4j_uri, auth=auth)
                    await self.driver.verify_connectivity()
                    logger.info("Connected to Neo4j.")
                    return
                except Exception as e:
                    logger.warning(f"Neo4j connection attempt {attempt + 1}/5 failed: {e}")
                    self.driver = None
                    await asyncio.sleep(2)
            logger.error("Failed to connect to Neo4j after 5 attempts.")

    async def ensure_connected(self) -> bool:
        """Lazy logic to ensure driver is active before session use."""
        if not self.driver:
            await self.connect()
        return self.driver is not None

    async def close(self):
        """Close the async Neo4j driver."""
        if self.driver:
            await self.driver.close()
            logger.info("Disconnected from Neo4j.")
            self.driver = None


# Singleton — shared by all services
neo4j_connection = Neo4jConnection()
