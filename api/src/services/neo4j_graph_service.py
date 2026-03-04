"""
Graph visualization service — queries for topics and graph data.
"""
import logging
from typing import List, Dict, Any

from src.services.neo4j_connection import Neo4jConnection
from src.repositories.neo4j_queries import (
    GET_TOPICS_QUERY,
    GET_VISUALIZE_GRAPH_QUERY,
    GET_VISUALIZE_CONTEXT_QUERY,
    GET_TOTAL_NODES_ALL,
    GET_TOTAL_NODES_CONTEXT,
)

logger = logging.getLogger(__name__)


class Neo4jGraphService:
    """Handles graph visualization queries. Implements GraphDataGateway."""

    def __init__(self, connection: Neo4jConnection):
        self._conn = connection

    async def get_topics(self) -> List[str]:
        """Get the sibling topics connected to the origin."""
        if not await self._conn.ensure_connected():
            return []

        async with self._conn.driver.session() as session:
            result = await session.run(GET_TOPICS_QUERY)
            records = await result.data()
            return [record["topic"] for record in records]

    async def get_visualize_graph(self, limit: int = 500, context: str = "all") -> Dict[str, Any]:
        """Get nodes and links formatted for the frontend GraphViewer."""
        if not await self._conn.ensure_connected():
            return {"nodes": [], "links": [], "totalNodes": 0}

        async with self._conn.driver.session() as session:
            if context and context.lower() != "all":
                count_query = GET_TOTAL_NODES_CONTEXT
                viz_query = GET_VISUALIZE_CONTEXT_QUERY.format(limit=limit)
                params = {"context": context}
            else:
                count_query = GET_TOTAL_NODES_ALL
                viz_query = GET_VISUALIZE_GRAPH_QUERY.format(limit=limit)
                params = {}

            count_result = await session.run(count_query, params)
            count_record = await count_result.single()
            total_nodes = count_record["total"] if count_record else 0

            viz_result = await session.run(viz_query, params)
            viz_record = await viz_result.single()

            if viz_record:
                return {
                    "nodes": viz_record["nodes"],
                    "links": viz_record["links"],
                    "totalNodes": total_nodes,
                }
            return {"nodes": [], "links": [], "totalNodes": total_nodes}
