from fastapi import APIRouter, Depends

from src.core.protocols import LawSearchGateway
from src.api.dependencies import get_assistant_service
from src.schemas.assistant import AssistantQuestionRequest, AssistantResponse
from src.use_cases.ask_legal_question import ask_legal_question
from src.services.llm.factory import create_provider

router = APIRouter(prefix="/v1/assistant", tags=["AI Legal Assistant"])


@router.post(
    "/ask",
    response_model=AssistantResponse,
    summary="Ask a legal question to the AI assistant",
    description=(
        "Sends a legal question through a two-step AI pipeline: "
        "1) The LLM identifies the relevant law/article, "
        "2) Neo4j provides the full legislative tree as context, "
        "3) The LLM answers prioritizing the graph data. "
        "The client must supply ai_model and api_token in the request body."
    ),
)
async def ask_question(
    body: AssistantQuestionRequest,
    neo4j: LawSearchGateway = Depends(get_assistant_service),
) -> AssistantResponse:
    llm = create_provider(
        provider_name=body.ai_model,
        api_key=body.api_token,
    )

    return await ask_legal_question(
        question=body.message,
        llm=llm,
        neo4j=neo4j,
    )
