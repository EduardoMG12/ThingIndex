package provider

import (
	"sort"
	"strings"

	"thing-index/internal/domain"
)

type providerSupportSeed struct {
	displayName  string
	supportLevel string
	transport    string
	summary      string
	notes        []string
	liveTests    bool
}

var llmProviderSupportSeeds = map[string]providerSupportSeed{
	domain.ProviderOpenAI: {
		displayName:  "OpenAI",
		supportLevel: domain.ProviderSupportLevelReady,
		transport:    domain.ProviderSupportTransportNativeOpenAI,
		summary:      "Ready for the main runtime with direct OpenAI defaults, connection test, model discovery and grounded chat/indexing.",
		notes: []string{
			"Uses the direct OpenAI API path with optional custom base_url overrides.",
			"Model can stay implicit because the runtime falls back to gpt-4o-mini when the field is empty.",
		},
		liveTests: true,
	},
	domain.ProviderOllama: {
		displayName:  "Ollama",
		supportLevel: domain.ProviderSupportLevelReady,
		transport:    domain.ProviderSupportTransportNativeOllama,
		summary:      "Ready for the main runtime in self-hosted deployments, with native Ollama chat, local defaults and live model discovery.",
		notes: []string{
			"Local Ollama works without api_key; Ollama Cloud requires api_key.",
			"Model must be filled explicitly in saved configs even though the runtime suggests llama3.1:8b.",
		},
		liveTests: true,
	},
	domain.ProviderDeepSeek: {
		displayName:  "DeepSeek",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportOpenAICompatible,
		summary:      "Supported through the shared OpenAI-compatible adapter with DeepSeek defaults, readiness checks and optional live validation.",
		notes: []string{
			"Provider-specific behavior still rides the shared OpenAI-compatible client instead of a dedicated DeepSeek transport.",
			"Suitable for controlled use in indexing and chat, but not presented as a first-class runtime path above OpenAI/Ollama.",
		},
		liveTests: true,
	},
	domain.ProviderGrok: {
		displayName:  "Grok",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportOpenAICompatible,
		summary:      "Supported through the shared OpenAI-compatible adapter with xAI defaults, readiness checks and optional live validation.",
		notes: []string{
			"Uses the same OpenAI-compatible transport path as other bridge-style providers.",
			"Keep validation scoped to concrete deployments instead of assuming parity with every xAI feature.",
		},
		liveTests: true,
	},
	domain.ProviderOpenRouter: {
		displayName:  "OpenRouter",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportOpenAICompatible,
		summary:      "Supported through the OpenAI-compatible gateway path with readiness checks, model discovery and optional live validation.",
		notes: []string{
			"Requires an explicit model id in saved configs even though the runtime suggests openai/gpt-4o-mini.",
			"Best treated as a gateway layer whose upstream behavior depends on the selected routed model.",
		},
		liveTests: true,
	},
	domain.ProviderGemini: {
		displayName:  "Google Gemini",
		supportLevel: domain.ProviderSupportLevelExperimental,
		transport:    domain.ProviderSupportTransportOpenAICompatibleShim,
		summary:      "Experimental bridge-only support. The runtime can work when Gemini is exposed behind an OpenAI-compatible endpoint with explicit model and base_url.",
		notes: []string{
			"Native Gemini API integration is not implemented in the main runtime.",
			"Treat this as bridge-backed validation, not as a first-class Gemini transport.",
		},
		liveTests: true,
	},
	domain.ProviderClaude: {
		displayName:  "Claude",
		supportLevel: domain.ProviderSupportLevelExperimental,
		transport:    domain.ProviderSupportTransportOpenAICompatibleShim,
		summary:      "Experimental bridge-only support. The runtime can work when Claude is exposed through an OpenAI-compatible endpoint with explicit model and base_url.",
		notes: []string{
			"Native Anthropic API integration is not implemented in the main runtime.",
			"Use only when you control the bridge behavior and can validate the deployed endpoint directly.",
		},
		liveTests: true,
	},
}

func ListProviderSupportMatrix(family string) []domain.ProviderSupportEntry {
	normalizedFamily := strings.ToUpper(strings.TrimSpace(family))
	if normalizedFamily != "" && normalizedFamily != domain.FamilyLLM {
		return nil
	}

	items := make([]domain.ProviderSupportEntry, 0, len(llmProviderSupportSeeds))
	for providerType, seed := range llmProviderSupportSeeds {
		entry := domain.ProviderSupportEntry{
			Family:           domain.FamilyLLM,
			ProviderType:     providerType,
			DisplayName:      seed.displayName,
			SupportLevel:     seed.supportLevel,
			Transport:        seed.transport,
			Summary:          seed.summary,
			SuggestedBaseURL: suggestedBaseURL(providerType),
			SuggestedModel:   suggestedModel(providerType),
			Capabilities: domain.ProviderSupportCapabilities{
				SupportsCredentialValidation:   true,
				SupportsConnectionTest:         true,
				SupportsModelDiscovery:         SupportsLLMModelDiscovery(providerType),
				SupportsStreaming:              true,
				SupportsIndexing:               true,
				SupportsChat:                   true,
				SupportsOptionalLiveTests:      seed.liveTests,
				RequiresExplicitModel:          requiresExplicitModel(providerType),
				RequiresExplicitBaseURL:        requiresExplicitBaseURL(providerType),
				RequiresOpenAICompatibleBridge: requiresOpenAICompatibleBridge(providerType),
			},
			Notes: append([]string(nil), seed.notes...),
		}
		items = append(items, entry)
	}

	sort.Slice(items, func(i, j int) bool {
		leftWeight := supportLevelWeight(items[i].SupportLevel)
		rightWeight := supportLevelWeight(items[j].SupportLevel)
		if leftWeight != rightWeight {
			return leftWeight < rightWeight
		}
		return items[i].DisplayName < items[j].DisplayName
	})

	return items
}

func LookupProviderSupport(providerType string) (domain.ProviderSupportEntry, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(providerType))
	for _, item := range ListProviderSupportMatrix(domain.FamilyLLM) {
		if item.ProviderType == normalized {
			return item, true
		}
	}

	return domain.ProviderSupportEntry{}, false
}

func supportLevelWeight(level string) int {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case domain.ProviderSupportLevelReady:
		return 0
	case domain.ProviderSupportLevelPartial:
		return 1
	case domain.ProviderSupportLevelExperimental:
		return 2
	default:
		return 3
	}
}

func suggestedBaseURL(providerType string) string {
	switch strings.ToUpper(strings.TrimSpace(providerType)) {
	case domain.ProviderOllama:
		return "http://localhost:11434/v1"
	default:
		definition, ok := llmProviderDefinitions[strings.ToUpper(strings.TrimSpace(providerType))]
		if !ok {
			return ""
		}
		return definition.defaultBaseURL
	}
}

func suggestedModel(providerType string) string {
	definition, ok := llmProviderDefinitions[strings.ToUpper(strings.TrimSpace(providerType))]
	if !ok {
		return ""
	}

	return definition.defaultModel
}

func requiresExplicitModel(providerType string) bool {
	switch strings.ToUpper(strings.TrimSpace(providerType)) {
	case domain.ProviderGemini, domain.ProviderClaude, domain.ProviderOpenRouter, domain.ProviderOllama:
		return true
	default:
		return false
	}
}

func requiresExplicitBaseURL(providerType string) bool {
	switch strings.ToUpper(strings.TrimSpace(providerType)) {
	case domain.ProviderGemini, domain.ProviderClaude:
		return true
	default:
		return false
	}
}

func requiresOpenAICompatibleBridge(providerType string) bool {
	switch strings.ToUpper(strings.TrimSpace(providerType)) {
	case domain.ProviderGemini, domain.ProviderClaude:
		return true
	default:
		return false
	}
}
