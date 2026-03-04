"""
LLM Services Package.

Import all provider modules here so they auto-register
via the @register_provider decorator.
"""

# Auto-register all providers by importing them.
# To add a new provider: create the file and import it here.
from src.services.llm import gemini_provider  # noqa: F401
