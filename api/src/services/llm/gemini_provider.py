import json
import logging
from typing import Dict, Any

import google.generativeai as genai

from src.services.llm.prompts import IDENTIFY_LAW_SYSTEM_PROMPT, ANSWER_WITH_CONTEXT_SYSTEM_PROMPT
from src.services.llm.registry import register_provider

logger = logging.getLogger(__name__)


@register_provider("gemini")
class GeminiProvider:
    """Google Gemini implementation of LLMProviderGateway."""

    def __init__(self, api_key: str, model_name: str = "gemini-2.0-flash"):
        if not api_key:
            raise ValueError("Gemini API key is required.")

        genai.configure(api_key=api_key)
        self._model = genai.GenerativeModel(model_name)
        logger.info(f"GeminiProvider initialized with model: {model_name}")

    async def identify_law(self, question: str) -> Dict[str, Any]:
        """Ask Gemini to identify which law/article the question refers to."""
        try:
            response = await self._model.generate_content_async(
                contents=[
                    {"role": "user", "parts": [f"{IDENTIFY_LAW_SYSTEM_PROMPT}\n\nPergunta do usuário: {question}"]}
                ],
                generation_config=genai.types.GenerationConfig(
                    temperature=0.1,
                    max_output_tokens=500,
                ),
            )

            raw_text = response.text.strip()

            # Clean markdown code fences if present
            if raw_text.startswith("```"):
                lines = raw_text.split("\n")
                raw_text = "\n".join(lines[1:-1] if lines[-1].strip() == "```" else lines[1:])

            parsed = json.loads(raw_text)
            logger.info(f"Gemini identified law: {parsed}")
            return {
                "law_name": parsed.get("law_name", ""),
                "article_number": parsed.get("article_number", ""),
                "search_terms": parsed.get("search_terms", []),
            }
        except json.JSONDecodeError as e:
            logger.warning(f"Failed to parse Gemini response as JSON: {e}. Raw: {raw_text}")
            return {
                "law_name": "",
                "article_number": "",
                "search_terms": [question],
            }
        except Exception as e:
            logger.error(f"Gemini identify_law failed: {e}", exc_info=True)
            raise

    async def answer_with_context(self, question: str, context: str) -> str:
        """Ask Gemini to answer the question using the graph context."""
        try:
            prompt = (
                f"{ANSWER_WITH_CONTEXT_SYSTEM_PROMPT}\n\n"
                f"--- CONTEXTO DO GRAFO DE CONHECIMENTO ---\n{context}\n"
                f"--- FIM DO CONTEXTO ---\n\n"
                f"Pergunta do usuário: {question}"
            )

            response = await self._model.generate_content_async(
                contents=[{"role": "user", "parts": [prompt]}],
                generation_config=genai.types.GenerationConfig(
                    temperature=0.4,
                    max_output_tokens=2048,
                ),
            )

            return response.text.strip()
        except Exception as e:
            logger.error(f"Gemini answer_with_context failed: {e}", exc_info=True)
            raise
