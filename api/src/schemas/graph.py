from pydantic import Field
from typing import List
from src.schemas.base import ProtectedSchema

class TopicListResponse(ProtectedSchema):
    topics: List[str]

class GraphNode(ProtectedSchema):
    id: str
    label: str
    group: int

class GraphLink(ProtectedSchema):
    source: str
    target: str
    label: str

class GraphVisualizeResponse(ProtectedSchema):
    nodes: List[GraphNode]
    links: List[GraphLink]
    total_nodes: int = Field(..., alias="total_nodes")
