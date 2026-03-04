"""
Factory module — thin wrapper around the registry.

The old elif-based factory is gone. Now it simply delegates
to the Registry Pattern via create_provider().
"""

from src.services.llm.registry import create_provider, list_providers

# Trigger auto-registration of all providers
import src.services.llm  # noqa: F401

__all__ = ["create_provider", "list_providers"]
