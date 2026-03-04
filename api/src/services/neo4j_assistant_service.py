"""
Assistant search service — law tree queries for the AI assistant.
"""
import logging
from typing import List, Dict, Any

from src.services.neo4j_connection import Neo4jConnection
from src.repositories.neo4j_queries import SEARCH_LAW_TREE_QUERY

logger = logging.getLogger(__name__)


class Neo4jAssistantService:
    """Handles law tree search queries. Implements LawSearchGateway."""

    def __init__(self, connection: Neo4jConnection):
        self._conn = connection

    async def search_law_tree(self, search_terms: List[str]) -> Dict[str, Any]:
        """Search for laws/articles by terms and return the full tree context."""
        if not await self._conn.ensure_connected():
            return {"nodes": [], "relationships": []}

        if not search_terms:
            return {"nodes": [], "relationships": []}

        async with self._conn.driver.session() as session:
            result = await session.run(
                SEARCH_LAW_TREE_QUERY,
                {"terms": search_terms},
            )
            record = await result.single()

            if record:
                return {
                    "nodes": record["nodes"] or [],
                    "relationships": record["relationships"] or [],
                }
            return {"nodes": [], "relationships": []}
