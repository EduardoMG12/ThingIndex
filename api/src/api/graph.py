from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from typing import List, Dict, Any
from src.services.neo4j_client import neo4j_client

router = APIRouter(prefix="/v1/graph", tags=["Graph Visualization"])

class TopicListResponse(BaseModel):
    topics: List[str]

class GraphNode(BaseModel):
    id: str
    label: str
    group: int

class GraphLink(BaseModel):
    source: str
    target: str
    label: str

class GraphVisualizeResponse(BaseModel):
    nodes: List[GraphNode]
    links: List[GraphLink]
    total_nodes: int

@router.get("/topics", response_model=TopicListResponse, summary="List main topics connected to Origin")
async def get_topics():
    """Returns a list of sibling topics connected to the central 'Brasil' origin node."""
    try:
        topics = await neo4j_client.get_topics()
        return TopicListResponse(topics=topics)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/visualize", response_model=GraphVisualizeResponse, summary="Get graph data for visualization")
async def get_visualize_graph(limit: int = 500, context: str = "all"):
    """
    Returns nodes and links formatted for the react-force-graph-2d frontend component.
    Limits the number of relationships returned to avoid overloading the browser.
    Can be filtered by an optional 'context' string (e.g., 'Código Penal').
    """
    try:
        graph_data = await neo4j_client.get_visualize_graph(limit=limit, context=context)
        return GraphVisualizeResponse(
            nodes=graph_data["nodes"],
            links=graph_data["links"],
            total_nodes=graph_data.get("totalNodes", 0)
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
