package dto

import (
	"time"

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
