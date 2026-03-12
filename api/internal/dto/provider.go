package dto

import (
	"time"

	"thing-index/internal/models"
)

type ProviderConfigResponse struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	Family       string    `json:"family"`
	ProviderType string    `json:"provider_type"`
	IsEnabled    bool      `json:"is_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func ToProviderConfigResponse(config *models.UserProviderConfig) *ProviderConfigResponse {
	if config == nil {
		return nil
	}

	return &ProviderConfigResponse{
		ID:           config.ID,
		UserID:       config.UserID,
		Family:       config.Family,
		ProviderType: config.ProviderType,
		IsEnabled:    config.IsEnabled,
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	}
}

func ToProviderConfigResponses(configs []models.UserProviderConfig) []ProviderConfigResponse {
	items := make([]ProviderConfigResponse, 0, len(configs))
	for i := range configs {
		if item := ToProviderConfigResponse(&configs[i]); item != nil {
			items = append(items, *item)
		}
	}
	return items
}
