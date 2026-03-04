"""
Registry Pattern for LLM providers.

Providers self-register via the @register_provider decorator.
No if/elif chains — adding a new provider is just creating a file
with a decorated class.
"""
import logging
from typing import Dict, Type, Callable

from src.core.protocols import LLMProviderGateway

logger = logging.getLogger(__name__)

# provider_name -> class
_PROVIDER_REGISTRY: Dict[str, Type] = {}


def register_provider(name: str) -> Callable:
    """Decorator that registers an LLM provider class under *name*.

    Usage:
        @register_provider("gemini")
        class GeminiProvider:
            ...
    """
    def decorator(cls: Type) -> Type:
        key = name.lower().strip()
        if key in _PROVIDER_REGISTRY:
            logger.warning(f"Provider '{key}' is being overridden by {cls.__name__}")
        _PROVIDER_REGISTRY[key] = cls
        logger.info(f"Registered LLM provider: '{key}' -> {cls.__name__}")
        return cls
    return decorator


def create_provider(provider_name: str, api_key: str) -> LLMProviderGateway:
    """Instantiate a registered provider by name.

    Raises ValueError if the provider is not registered.
    """
    key = provider_name.lower().strip()
    if key not in _PROVIDER_REGISTRY:
        available = ", ".join(sorted(_PROVIDER_REGISTRY.keys())) or "(none)"
        raise ValueError(
            f"Unknown LLM provider: '{key}'. Available: {available}"
        )

    cls = _PROVIDER_REGISTRY[key]
    return cls(api_key=api_key)


def list_providers() -> list[str]:
    """Return a sorted list of registered provider names."""
    return sorted(_PROVIDER_REGISTRY.keys())
