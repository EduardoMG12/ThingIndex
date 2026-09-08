package dto

import (
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type ProviderConfigResponse struct {
	ID                 uint              `json:"id"`
	UserID             uint              `json:"user_id"`
	Family             string            `json:"family"`
	ProviderType       string            `json:"provider_type"`
	IsEnabled          bool              `json:"is_enabled"`
	EnabledForIndexing bool              `json:"enabled_for_indexing"`
	EnabledForChat     bool              `json:"enabled_for_chat"`
	PublicMetadata     map[string]string `json:"public_metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type ProviderConnectionRequest struct {
	Family       string            `json:"family" example:"LLM"`
	ProviderType string            `json:"provider_type" example:"OPENAI"`
	Credentials  map[string]string `json:"credentials"`
}

type ProviderModelsResponse struct {
	Models []string `json:"models"`
}

type ProviderSupportCapabilitiesResponse struct {
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

type ProviderSupportResponse struct {
	Family         string                              `json:"family"`
	ProviderType   string                              `json:"provider_type"`
	DisplayName    string                              `json:"display_name"`
	SupportLevel   string                              `json:"support_level"`
	Transport      string                              `json:"transport"`
	Summary        string                              `json:"summary"`
	SuggestedSetup map[string]string                   `json:"suggested_setup,omitempty"`
	Capabilities   ProviderSupportCapabilitiesResponse `json:"capabilities"`
	Notes          []string                            `json:"notes,omitempty"`
}

func ToProviderConfigResponse(config *models.UserProviderConfig, publicMetadata map[string]string) *ProviderConfigResponse {
	if config == nil {
		return nil
	}

	return &ProviderConfigResponse{
		ID:                 config.ID,
		UserID:             config.UserID,
		Family:             config.Family,
		ProviderType:       config.ProviderType,
		IsEnabled:          config.IsEnabled,
		EnabledForIndexing: config.EnabledForIndexing,
		EnabledForChat:     config.EnabledForChat,
		PublicMetadata:     publicMetadata,
		CreatedAt:          config.CreatedAt,
		UpdatedAt:          config.UpdatedAt,
	}
}

func ToProviderConfigResponses(configs []models.UserProviderConfig, publicMetadataByProvider map[string]map[string]string) []ProviderConfigResponse {
	items := make([]ProviderConfigResponse, 0, len(configs))
	for i := range configs {
		if item := ToProviderConfigResponse(&configs[i], publicMetadataByProvider[configs[i].ProviderType]); item != nil {
			items = append(items, *item)
		}
	}
	return items
}

func ToProviderSupportResponse(item domain.ProviderSupportEntry) ProviderSupportResponse {
	return ProviderSupportResponse{
		Family:         item.Family,
		ProviderType:   item.ProviderType,
		DisplayName:    item.DisplayName,
		SupportLevel:   item.SupportLevel,
		Transport:      item.Transport,
		Summary:        item.Summary,
		SuggestedSetup: cloneProviderSupportSetup(item.SuggestedSetup),
		Capabilities: ProviderSupportCapabilitiesResponse{
			SupportsCredentialValidation:   item.Capabilities.SupportsCredentialValidation,
			SupportsConnectionTest:         item.Capabilities.SupportsConnectionTest,
			SupportsModelDiscovery:         item.Capabilities.SupportsModelDiscovery,
			SupportsStreaming:              item.Capabilities.SupportsStreaming,
			SupportsIndexing:               item.Capabilities.SupportsIndexing,
			SupportsChat:                   item.Capabilities.SupportsChat,
			SupportsOptionalLiveTests:      item.Capabilities.SupportsOptionalLiveTests,
			RequiresExplicitModel:          item.Capabilities.RequiresExplicitModel,
			RequiresExplicitBaseURL:        item.Capabilities.RequiresExplicitBaseURL,
			RequiresOpenAICompatibleBridge: item.Capabilities.RequiresOpenAICompatibleBridge,
		},
		Notes: copiaDeTextos(item.Notes),
	}
}

func cloneProviderSupportSetup(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func ToProviderSupportResponses(items []domain.ProviderSupportEntry) []ProviderSupportResponse {
	responses := make([]ProviderSupportResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, ToProviderSupportResponse(item))
	}
	return responses
}
