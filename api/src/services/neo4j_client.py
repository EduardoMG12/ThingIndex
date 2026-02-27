import logging
from neo4j import AsyncGraphDatabase
from src.core.config import settings
from src.repositories.neo4j_queries import GET_TOPICS_QUERY, GET_VISUALIZE_GRAPH_QUERY

logger = logging.getLogger(__name__)

class Neo4jClient:
    def __init__(self):
        self.driver = None

    async def connect(self):
        """Initialize the async Neo4j driver."""
        if not self.driver:
            try:
                # NEO4J_AUTH is usually format "user/password"
                if "/" in settings.neo4j_auth:
                    user, password = settings.neo4j_auth.split("/", 1)
                    auth = (user, password)
                else:
                    auth = None

                self.driver = AsyncGraphDatabase.driver(settings.neo4j_uri, auth=auth)
                await self.driver.verify_connectivity()
                logger.info("Connected to Neo4j.")
            except Exception as e:
                logger.error(f"Failed to connect to Neo4j: {e}")
                self.driver = None

    async def close(self):
        """Close the async Neo4j driver."""
        if self.driver:
            await self.driver.close()
            logger.info("Disconnected from Neo4j.")
            self.driver = None

    async def get_topics(self):
        """Get the sibling topics connected to the origin."""
        if not self.driver:
            return []
            
        async with self.driver.session() as session:
            result = await session.run(GET_TOPICS_QUERY)
            records = await result.data()
            return [record["topic"] for record in records]

    async def get_visualize_graph(self, limit: int = 500):
        """Get nodes and links formatted for the frontend GraphViewer."""
        query = GET_VISUALIZE_GRAPH_QUERY.format(limit=limit)
        
        if not self.driver:
            return {"nodes": [], "links": []}
            
        async with self.driver.session() as session:
            result = await session.run(query)
            record = await result.single()
            if record:
                return {
                    "nodes": record["nodes"],
                    "links": record["links"]
                }
            return {"nodes": [], "links": []}

# Singleton instance
neo4j_client = Neo4jClient()
