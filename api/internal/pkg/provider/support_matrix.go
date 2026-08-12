package provider

import (
	"sort"
	"strings"

	"thing-index/internal/domain"
)

type providerSupportSeed struct {
	displayName    string
	supportLevel   string
	transport      string
	summary        string
	suggestedSetup map[string]string
	notes          []string
	liveTests      bool
}

var storageProviderSupportSeeds = map[string]providerSupportSeed{
	domain.ProviderMinio: {
		displayName:  "MinIO / S3-compatible",
		supportLevel: domain.ProviderSupportLevelReady,
		transport:    domain.ProviderSupportTransportS3Compatible,
		summary:      "Ready for the main runtime with upload, preview, download, indexing and live integration coverage against the local stack.",
		suggestedSetup: map[string]string{
			"endpoint":    "minio.example.com:9000",
			"bucket_name": "thingindex-docs",
		},
		notes: []string{
			"Uses the shared S3-compatible path backed by MinIO's Go client.",
			"Suitable for the default local runtime and self-hosted deployments that want a direct object-storage path.",
		},
		liveTests: true,
	},
	domain.ProviderAWSS3: {
		displayName:  "AWS S3",
		supportLevel: domain.ProviderSupportLevelReady,
		transport:    domain.ProviderSupportTransportS3Compatible,
		summary:      "Ready through the shared S3-compatible adapter with region-aware defaults, readiness checks and the same document lifecycle used by MinIO.",
		suggestedSetup: map[string]string{
			"region":      "us-east-1",
			"bucket_name": "thingindex-docs",
		},
		notes: []string{
			"Runtime bootstrap can stay on MinIO while user-scoped providers point to AWS S3 buckets.",
			"Custom endpoints remain optional for S3-compatible gateways layered in front of S3.",
		},
		liveTests: false,
	},
	domain.ProviderCloudflare: {
		displayName:  "Cloudflare R2",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportS3Compatible,
		summary:      "Supported through the shared S3-compatible adapter with account-aware endpoint defaults and document lifecycle parity.",
		suggestedSetup: map[string]string{
			"account_id":  "your-account-id",
			"bucket_name": "thingindex-docs",
		},
		notes: []string{
			"Uses the same object-storage adapter as MinIO/S3, not a Cloudflare-specific SDK path.",
			"Validate vendor-specific behaviors in your own deployment before assuming parity with every R2 feature.",
		},
		liveTests: false,
	},
	domain.ProviderDOSpaces: {
		displayName:  "DigitalOcean Spaces",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportS3Compatible,
		summary:      "Supported through the shared S3-compatible adapter when you provide the regional endpoint, bucket and keys.",
		suggestedSetup: map[string]string{
			"endpoint":    "nyc3.digitaloceanspaces.com",
			"region":      "nyc3",
			"bucket_name": "thingindex-docs",
		},
		notes: []string{
			"Operational behavior follows the S3-compatible code path already used by the document pipeline.",
			"Treat provider-specific CDN, ACL or lifecycle policies as deployment concerns outside the main runtime contract.",
		},
		liveTests: false,
	},
	domain.ProviderWasabi: {
		displayName:  "Wasabi",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportS3Compatible,
		summary:      "Supported through the shared S3-compatible adapter with explicit regional endpoints and bucket validation.",
		suggestedSetup: map[string]string{
			"endpoint":    "s3.us-east-1.wasabisys.com",
			"region":      "us-east-1",
			"bucket_name": "thingindex-docs",
		},
		notes: []string{
			"Uses the same upload, preview and deletion flow exercised by the S3-compatible storage path.",
			"Keep deployment validation focused on your concrete Wasabi region and bucket policy configuration.",
		},
		liveTests: false,
	},
	domain.ProviderB2: {
		displayName:  "Backblaze B2",
		supportLevel: domain.ProviderSupportLevelPartial,
		transport:    domain.ProviderSupportTransportS3Compatible,
		summary:      "Supported through the shared S3-compatible adapter with explicit endpoint, region and bucket settings.",
		suggestedSetup: map[string]string{
			"endpoint":    "s3.us-west-004.backblazeb2.com",
			"region":      "us-west-004",
			"bucket_name": "thingindex-docs",
		},
		notes: []string{
			"Backblaze B2 rides the common S3-compatible code path instead of a dedicated B2 SDK.",
			"Validate B2-specific operational policies separately when hard guarantees depend on them.",
		},
		liveTests: false,
	},
	domain.ProviderGCPStorage: {
		displayName:  "Google Cloud Storage",
		supportLevel: domain.ProviderSupportLevelReady,
		transport:    domain.ProviderSupportTransportNativeGCS,
		summary:      "Ready for the main runtime with a native Google Cloud Storage path, encrypted credential storage, connection testing and integration coverage via emulator.",
		suggestedSetup: map[string]string{
			"project_id":  "thingindex-dev",
			"bucket_name": "thingindex-docs",
			"location":    "us-central1",
		},
		notes: []string{
			"Supports service-account JSON for real buckets and emulator_host for local development/integration runs.",
			"Reduces practical dependence on the S3-compatible stack while keeping the same document lifecycle contract.",
		},
		liveTests: true,
	},
}

var llmProviderSupportSeeds = map[string]providerSupportSeed{
	domain.ProviderOpenAI: {
		displayName:  "OpenAI",
		supportLevel: domain.ProviderSupportLevelReady,
		transport:    domain.ProviderSupportTransportNativeOpenAI,
		summary:      "Ready for the main runtime with direct OpenAI defaults, connection test, model discovery and grounded chat/indexing.",
		suggestedSetup: map[string]string{
			"base_url": "https://api.openai.com/v1",
			"model":    "gpt-4o-mini",
		},
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
		suggestedSetup: map[string]string{
			"base_url": "http://localhost:11434/v1",
			"model":    "llama3.1:8b",
		},
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
		suggestedSetup: map[string]string{
			"base_url": "https://api.deepseek.com",
			"model":    "deepseek-chat",
		},
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
		suggestedSetup: map[string]string{
			"base_url": "https://api.x.ai/v1",
			"model":    "grok-2-latest",
		},
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
		suggestedSetup: map[string]string{
			"base_url": "https://openrouter.ai/api/v1",
			"model":    "openai/gpt-4o-mini",
		},
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
		suggestedSetup: map[string]string{
			"base_url": "Bring your bridge URL",
			"model":    "gemini-2.5-flash",
		},
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
		suggestedSetup: map[string]string{
			"base_url": "Bring your bridge URL",
			"model":    "claude-3-5-sonnet",
		},
		notes: []string{
			"Native Anthropic API integration is not implemented in the main runtime.",
			"Use only when you control the bridge behavior and can validate the deployed endpoint directly.",
		},
		liveTests: true,
	},
}

func ListProviderSupportMatrix(family string) []domain.ProviderSupportEntry {
	normalizedFamily := strings.ToUpper(strings.TrimSpace(family))
	if normalizedFamily != "" && normalizedFamily != domain.FamilyLLM && normalizedFamily != domain.FamilyStorage {
		return nil
	}

	items := make([]domain.ProviderSupportEntry, 0, len(storageProviderSupportSeeds)+len(llmProviderSupportSeeds))
	if normalizedFamily == "" || normalizedFamily == domain.FamilyStorage {
		items = append(items, buildStorageSupportEntries()...)
	}
	if normalizedFamily == "" || normalizedFamily == domain.FamilyLLM {
		items = append(items, buildLLMSupportEntries()...)
	}

	sort.Slice(items, func(i, j int) bool {
		leftFamilyWeight := supportFamilyWeight(items[i].Family)
		rightFamilyWeight := supportFamilyWeight(items[j].Family)
		if leftFamilyWeight != rightFamilyWeight {
			return leftFamilyWeight < rightFamilyWeight
		}
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
	for _, item := range ListProviderSupportMatrix("") {
		if item.ProviderType == normalized {
			return item, true
		}
	}

	return domain.ProviderSupportEntry{}, false
}

func buildStorageSupportEntries() []domain.ProviderSupportEntry {
	items := make([]domain.ProviderSupportEntry, 0, len(storageProviderSupportSeeds))
	for providerType, seed := range storageProviderSupportSeeds {
		items = append(items, domain.ProviderSupportEntry{
			Family:         domain.FamilyStorage,
			ProviderType:   providerType,
			DisplayName:    seed.displayName,
			SupportLevel:   seed.supportLevel,
			Transport:      seed.transport,
			Summary:        seed.summary,
			SuggestedSetup: cloneSupportSetup(seed.suggestedSetup),
			Capabilities: domain.ProviderSupportCapabilities{
				SupportsCredentialValidation: true,
				SupportsConnectionTest:       true,
				SupportsModelDiscovery:       false,
				SupportsStreaming:            false,
				SupportsIndexing:             true,
				SupportsChat:                 false,
				SupportsOptionalLiveTests:    seed.liveTests,
			},
			Notes: append([]string(nil), seed.notes...),
		})
	}

	return items
}

func buildLLMSupportEntries() []domain.ProviderSupportEntry {
	items := make([]domain.ProviderSupportEntry, 0, len(llmProviderSupportSeeds))
	for providerType, seed := range llmProviderSupportSeeds {
		items = append(items, domain.ProviderSupportEntry{
			Family:         domain.FamilyLLM,
			ProviderType:   providerType,
			DisplayName:    seed.displayName,
			SupportLevel:   seed.supportLevel,
			Transport:      seed.transport,
			Summary:        seed.summary,
			SuggestedSetup: cloneSupportSetup(seed.suggestedSetup),
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
		})
	}

	return items
}

func cloneSupportSetup(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func supportFamilyWeight(family string) int {
	switch strings.ToUpper(strings.TrimSpace(family)) {
	case domain.FamilyStorage:
		return 0
	case domain.FamilyLLM:
		return 1
	default:
		return 2
	}
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
