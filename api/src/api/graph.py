from fastapi import APIRouter, Depends
from typing import List, Dict, Any
from src.schemas.graph import TopicListResponse, GraphVisualizeResponse
from src.core.protocols import GraphDataGateway
from src.api.dependencies import get_graph_service

router = APIRouter(prefix="/v1/graph", tags=["Graph Visualization"])

@router.get("/topics", response_model=TopicListResponse, summary="List main topics connected to Origin")
async def get_topics(client: GraphDataGateway = Depends(get_graph_service)):
    """Returns a list of sibling topics connected to the central 'Brasil' origin node."""
    topics = await client.get_topics()
    return TopicListResponse(topics=topics)

@router.get("/visualize", response_model=GraphVisualizeResponse, summary="Get graph data for visualization")
async def get_visualize_graph(
    limit: int = 500, 
    context: str = "all",
    client: GraphDataGateway = Depends(get_graph_service)
):
    """
    Returns nodes and links formatted for the react-force-graph-2d frontend component.
    Limits the number of relationships returned to avoid overloading the browser.
    Can be filtered by an optional 'context' string (e.g., 'Código Penal').
    """
    graph_data = await client.get_visualize_graph(limit=limit, context=context)
    return GraphVisualizeResponse(
        nodes=graph_data["nodes"],
        links=graph_data["links"],
        total_nodes=graph_data.get("totalNodes", 0)
    )
