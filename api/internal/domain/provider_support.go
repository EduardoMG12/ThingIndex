package domain

const (
	ProviderSupportLevelReady        = "READY"
	ProviderSupportLevelPartial      = "PARTIAL"
	ProviderSupportLevelExperimental = "EXPERIMENTAL"

	ProviderSupportTransportNativeOpenAI         = "NATIVE_OPENAI"
	ProviderSupportTransportNativeOllama         = "NATIVE_OLLAMA"
	ProviderSupportTransportNativeGCS            = "NATIVE_GCS"
	ProviderSupportTransportOpenAICompatible     = "OPENAI_COMPATIBLE"
	ProviderSupportTransportOpenAICompatibleShim = "OPENAI_COMPATIBLE_BRIDGE"
	ProviderSupportTransportS3Compatible         = "S3_COMPATIBLE"
)

type ProviderSupportCapabilities struct {
	SupportsCredentialValidation   bool `json:"supports_credential_validation"`
	SupportsConnectionTest         bool `json:"supports_connection_test"`
	SupportsModelDiscovery         bool `json:"supports_model_discovery"`
	SupportsStreaming              bool `json:"supports_streaming"`
	SupportsIndexing               bool `json:"supports_indexing"`
	SupportsChat                   bool `json:"supports_chat"`
	SupportsOptionalLiveTests      bool `json:"supports_optional_live_tests"`
	RequiresExplicitModel          bool `json:"requires_explicit_model"`
	RequiresExplicitBaseURL        bool `json:"requires_explicit_base_url"`
	RequiresOpenAICompatibleBridge bool `json:"requires_openai_compatible_bridge"`
}

type ProviderSupportEntry struct {
	Family         string                      `json:"family"`
	ProviderType   string                      `json:"provider_type"`
	DisplayName    string                      `json:"display_name"`
	SupportLevel   string                      `json:"support_level"`
	Transport      string                      `json:"transport"`
	Summary        string                      `json:"summary"`
	SuggestedSetup map[string]string           `json:"suggested_setup,omitempty"`
	Capabilities   ProviderSupportCapabilities `json:"capabilities"`
	Notes          []string                    `json:"notes,omitempty"`
}
