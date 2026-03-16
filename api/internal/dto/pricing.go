package dto

import (
	"time"

	"thing-index/internal/models"
)

type SavePricingSourceRequest struct {
	ProviderType string `json:"provider_type" binding:"required"`
	Name         string `json:"name"`
	SourceURL    string `json:"source_url" binding:"required"`
	IsEnabled    *bool  `json:"is_enabled,omitempty"`
}

type PricingSourceResponse struct {
	ID           uint       `json:"id"`
	UserID       uint       `json:"user_id"`
	ProviderType string     `json:"provider_type"`
	Name         string     `json:"name"`
	SourceURL    string     `json:"source_url"`
	IsEnabled    bool       `json:"is_enabled"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ModelPriceResponse struct {
	ID                           uint      `json:"id"`
	PricingSourceID              uint      `json:"pricing_source_id"`
	ProviderType                 string    `json:"provider_type"`
	ModelKey                     string    `json:"model_key"`
	ModelName                    string    `json:"model_name"`
	Unit                         string    `json:"unit"`
	Currency                     string    `json:"currency"`
	FlatPriceUSD                 float64   `json:"flat_price_usd"`
	InputCostPerMillionUSD       float64   `json:"input_cost_per_million_usd"`
	OutputCostPerMillionUSD      float64   `json:"output_cost_per_million_usd"`
	CachedInputCostPerMillionUSD float64   `json:"cached_input_cost_per_million_usd"`
	ContextCostPerMillionUSD     float64   `json:"context_cost_per_million_usd"`
	PriceNotes                   string    `json:"price_notes,omitempty"`
	LastSyncedAt                 time.Time `json:"last_synced_at"`
}

func ToPricingSourceResponse(source *models.PricingSource) *PricingSourceResponse {
	if source == nil {
		return nil
	}
	return &PricingSourceResponse{
		ID:           source.ID,
		UserID:       source.UserID,
		ProviderType: source.ProviderType,
		Name:         source.Name,
		SourceURL:    source.SourceURL,
		IsEnabled:    source.IsEnabled,
		LastSyncedAt: source.LastSyncedAt,
		CreatedAt:    source.CreatedAt,
		UpdatedAt:    source.UpdatedAt,
	}
}

func ToPricingSourceResponses(sources []models.PricingSource) []PricingSourceResponse {
	items := make([]PricingSourceResponse, 0, len(sources))
	for i := range sources {
		if item := ToPricingSourceResponse(&sources[i]); item != nil {
			items = append(items, *item)
		}
	}
	return items
}

func ToModelPriceResponses(prices []models.ModelPrice) []ModelPriceResponse {
	items := make([]ModelPriceResponse, 0, len(prices))
	for i := range prices {
		items = append(items, ModelPriceResponse{
			ID:                           prices[i].ID,
			PricingSourceID:              prices[i].PricingSourceID,
			ProviderType:                 prices[i].ProviderType,
			ModelKey:                     prices[i].ModelKey,
			ModelName:                    prices[i].ModelName,
			Unit:                         prices[i].Unit,
			Currency:                     prices[i].Currency,
			FlatPriceUSD:                 prices[i].FlatPriceUSD,
			InputCostPerMillionUSD:       prices[i].InputCostPerMillionUSD,
			OutputCostPerMillionUSD:      prices[i].OutputCostPerMillionUSD,
			CachedInputCostPerMillionUSD: prices[i].CachedInputCostPerMillionUSD,
			ContextCostPerMillionUSD:     prices[i].ContextCostPerMillionUSD,
			PriceNotes:                   prices[i].PriceNotes,
			LastSyncedAt:                 prices[i].LastSyncedAt,
		})
	}
	return items
}
