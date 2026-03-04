from typing import List, Optional
from pydantic import Field
from src.schemas.base import ProtectedSchema


class AssistantQuestionRequest(ProtectedSchema):
    """Request body for the legal assistant endpoint."""
    message: str = Field(
        ...,
        min_length=5,
        max_length=2000,
        description="The legal question to ask the assistant",
    )
    ai_model: str = Field(
        ...,
        min_length=1,
        description="LLM provider name (e.g. 'gemini', 'openai', 'grok')",
    )
    api_token: str = Field(
        ...,
        min_length=1,
        description="API token/key for the chosen LLM provider",
    )


class LawIdentification(ProtectedSchema):
    """Structured output from the LLM's law identification step."""
    law_name: str = Field(default="", description="Name of the identified law or code")
    article_number: str = Field(default="", description="Article number if applicable")
    search_terms: List[str] = Field(default_factory=list, description="Search terms for Neo4j lookup")


class AssistantSource(ProtectedSchema):
    """A single source node from the graph context."""
    label: str
    type: str
    content: Optional[str] = None


class AssistantResponse(ProtectedSchema):
    """Final response from the legal assistant."""
    answer: str = Field(..., description="The AI-generated answer with graph context")
    law_identification: LawIdentification = Field(
        ..., description="The law/article identified by the LLM"
    )
    sources: List[AssistantSource] = Field(
        default_factory=list,
        description="Source nodes from the graph used as context",
    )
    graph_context_found: bool = Field(
        default=False,
        description="Whether relevant data was found in Neo4j",
    )
