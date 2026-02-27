import logging
from neo4j import AsyncGraphDatabase
from src.core.config import settings
from src.repositories.neo4j_queries import (
    GET_TOPICS_QUERY, 
    GET_VISUALIZE_GRAPH_QUERY,
    GET_VISUALIZE_CONTEXT_QUERY,
    GET_TOTAL_NODES_ALL,
    GET_TOTAL_NODES_CONTEXT
)

logger = logging.getLogger(__name__)

class Neo4jClient:
    def __init__(self):
        self.driver = None

    async def connect(self):
        """Initialize the async Neo4j driver with retries."""
        import asyncio
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

    async def _ensure_connected(self):
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

    async def get_topics(self):
        """Get the sibling topics connected to the origin."""
        if not await self._ensure_connected():
            return []
            
        async with self.driver.session() as session:
            result = await session.run(GET_TOPICS_QUERY)
            records = await result.data()
            return [record["topic"] for record in records]

    async def get_visualize_graph(self, limit: int = 500, context: str = "all"):
        """Get nodes and links formatted for the frontend GraphViewer, optionally filtered by context."""
        if not await self._ensure_connected():
            return {"nodes": [], "links": [], "totalNodes": 0}
            
        async with self.driver.session() as session:
            # 1. Determine which queries to use and get total count
            if context and context.lower() != "all":
                count_query = GET_TOTAL_NODES_CONTEXT
                viz_query = GET_VISUALIZE_CONTEXT_QUERY.format(limit=limit)
                params = {"context": context}
            else:
                count_query = GET_TOTAL_NODES_ALL
                viz_query = GET_VISUALIZE_GRAPH_QUERY.format(limit=limit)
                params = {}
                
            # Fetch total count
            count_result = await session.run(count_query, params)
            count_record = await count_result.single()
            total_nodes = count_record["total"] if count_record else 0
            
            # Fetch graph data
            viz_result = await session.run(viz_query, params)
            viz_record = await viz_result.single()
            
            if viz_record:
                return {
                    "nodes": viz_record["nodes"],
                    "links": viz_record["links"],
                    "totalNodes": total_nodes
                }
            return {"nodes": [], "links": [], "totalNodes": total_nodes}

# Singleton instance
neo4j_client = Neo4jClient()
