import logging
from typing import List, Dict, Any

from src.core.protocols import LLMProviderGateway, LawSearchGateway
from src.schemas.assistant import LawIdentification, AssistantSource, AssistantResponse

logger = logging.getLogger(__name__)


def _format_graph_context(graph_data: Dict[str, Any]) -> str:
    """Convert Neo4j nodes and relationships into a human-readable text
    that can be injected as context into the LLM prompt."""
    nodes = graph_data.get("nodes", [])
    relationships = graph_data.get("relationships", [])

    if not nodes:
        return "Nenhum dado encontrado no grafo de conhecimento legislativo."

    # Build a node ID → label map for relationship display
    node_map: Dict[str, str] = {}
    lines: List[str] = []

    for node in nodes:
        node_id = node.get("id", "")
        labels = node.get("labels", [])
        name = node.get("name", "")
        numero = node.get("numero", "")
        conteudo = node.get("conteudo", "")
        topic = node.get("topic", "")

        display = name or numero or ", ".join(labels)
        node_map[node_id] = display

        type_label = labels[0] if labels else "Nó"
        parts = [f"[{type_label}] {display}"]
        if numero and name:
            parts.append(f"  Número: {numero}")
        if conteudo:
            parts.append(f"  Conteúdo: {conteudo}")
        if topic:
            parts.append(f"  Tópico: {topic}")

        lines.append("\n".join(parts))

    context_text = "=== NÓDOS ENCONTRADOS ===\n\n" + "\n\n".join(lines)

    if relationships:
        rel_lines = []
        for rel in relationships:
            src = node_map.get(rel.get("source", ""), "?")
            tgt = node_map.get(rel.get("target", ""), "?")
            rel_type = rel.get("type", "RELACIONADO")
            rel_lines.append(f"  {src}  --[{rel_type}]-->  {tgt}")

        context_text += "\n\n=== RELAÇÕES ===\n\n" + "\n".join(rel_lines)

    return context_text


def _extract_sources(graph_data: Dict[str, Any]) -> List[AssistantSource]:
    """Extract a simplified list of sources from graph nodes."""
    sources: List[AssistantSource] = []
    for node in graph_data.get("nodes", []):
        labels = node.get("labels", [])
        name = node.get("name", "") or node.get("numero", "")
        conteudo = node.get("conteudo", "")

        if name:
            sources.append(AssistantSource(
                label=name,
                type=labels[0] if labels else "Unknown",
                content=conteudo if conteudo else None,
            ))
    return sources


async def ask_legal_question(
    question: str,
    llm: LLMProviderGateway,
    neo4j: LawSearchGateway,
) -> AssistantResponse:
    """Orchestrates the two-step AI legal assistant flow:

    1. Ask the LLM to identify the relevant law/article
    2. Search Neo4j for that law's full tree
    3. Format the graph context as text
    4. Ask the LLM to answer using the graph context as priority
    5. Return the structured response
    """
    # Step 1: Identify the law
    logger.info(f"Step 1 — Identifying law for question: {question[:80]}...")
    identification_data = await llm.identify_law(question)

    law_id = LawIdentification(
        law_name=identification_data.get("law_name", ""),
        article_number=identification_data.get("article_number", ""),
        search_terms=identification_data.get("search_terms", []),
    )

    logger.info(f"LLM identified: {law_id.law_name} art.{law_id.article_number}, terms={law_id.search_terms}")

    # Step 2: Search Neo4j
    search_terms = list(law_id.search_terms)
    if law_id.law_name and law_id.law_name not in search_terms:
        search_terms.append(law_id.law_name)
    if law_id.article_number and law_id.article_number not in search_terms:
        search_terms.append(law_id.article_number)

    logger.info(f"Step 2 — Searching Neo4j with terms: {search_terms}")
    graph_data = await neo4j.search_law_tree(search_terms)

    nodes_found = len(graph_data.get("nodes", []))
    graph_context_found = nodes_found > 0
    logger.info(f"Neo4j returned {nodes_found} nodes")

    # Step 3: Format context
    context_text = _format_graph_context(graph_data)

    # Step 4: Answer with context
    logger.info("Step 3 — Asking LLM to answer with graph context...")
    answer = await llm.answer_with_context(question, context_text)

    # Step 5: Build response
    sources = _extract_sources(graph_data)

    return AssistantResponse(
        answer=answer,
        law_identification=law_id,
        sources=sources,
        graph_context_found=graph_context_found,
    )
