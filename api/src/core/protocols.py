from typing import Protocol, List, Dict, Any, runtime_checkable
from fastapi import WebSocket


@runtime_checkable
class GraphDataGateway(Protocol):
    """Interface for graph visualization queries (Neo4j)."""
    async def get_topics(self) -> List[str]: ...
    async def get_visualize_graph(self, limit: int = 500, context: str = "all") -> Dict[str, Any]: ...


@runtime_checkable
class LawSearchGateway(Protocol):
    """Interface for law search queries used by the AI assistant."""
    async def search_law_tree(self, search_terms: List[str]) -> Dict[str, Any]: ...


@runtime_checkable
class TaskQueueGateway(Protocol):
    """Interface for Task Queue operations (RabbitMQ)."""
    def publish_task(self, url: str, depth: int, dir_name: str = "", priority: int = 0) -> None: ...
    def purge_queue(self) -> None: ...


@runtime_checkable
class TelemetryGateway(Protocol):
    """Interface for Telemetry broadcasting (WebSockets)."""
    def register(self, ws: WebSocket) -> None: ...
    def unregister(self, ws: WebSocket) -> None: ...
    async def start(self) -> None: ...
    async def stop(self) -> None: ...


@runtime_checkable
class LLMProviderGateway(Protocol):
    """Interface for LLM providers (Gemini, OpenAI, etc.)."""
    async def identify_law(self, question: str) -> Dict[str, Any]:
        """Given a legal question, identify the relevant law/article.
        Returns dict with keys: law_name, article_number, search_terms."""
        ...

    async def answer_with_context(self, question: str, context: str) -> str:
        """Answer the question using the provided graph context as priority source."""
        ...
