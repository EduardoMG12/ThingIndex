package models

import "time"

type PricingSource struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	UserID       uint       `gorm:"not null;index:idx_pricing_source_user_provider,priority:1" json:"user_id"`
	ProviderType string     `gorm:"size:50;not null;index:idx_pricing_source_user_provider,priority:2" json:"provider_type"`
	Name         string     `gorm:"size:120;not null" json:"name"`
	SourceURL    string     `gorm:"size:1024;not null" json:"source_url"`
	IsEnabled    bool       `gorm:"not null;default:true;index" json:"is_enabled"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ModelPrice struct {
	ID                           uint      `gorm:"primarykey" json:"id"`
	UserID                       uint      `gorm:"not null;index:idx_model_price_user_source_model,priority:1" json:"user_id"`
	PricingSourceID              uint      `gorm:"not null;index:idx_model_price_user_source_model,priority:2" json:"pricing_source_id"`
	ProviderType                 string    `gorm:"size:50;not null;index" json:"provider_type"`
	ModelKey                     string    `gorm:"size:120;not null;index:idx_model_price_user_source_model,priority:3" json:"model_key"`
	ModelName                    string    `gorm:"size:255;not null" json:"model_name"`
	Unit                         string    `gorm:"size:80;not null" json:"unit"`
	Currency                     string    `gorm:"size:10;not null;default:'USD'" json:"currency"`
	FlatPriceUSD                 float64   `json:"flat_price_usd"`
	InputCostPerMillionUSD       float64   `json:"input_cost_per_million_usd"`
	OutputCostPerMillionUSD      float64   `json:"output_cost_per_million_usd"`
	CachedInputCostPerMillionUSD float64   `json:"cached_input_cost_per_million_usd"`
	ContextCostPerMillionUSD     float64   `json:"context_cost_per_million_usd"`
	PriceNotes                   string    `gorm:"type:text" json:"price_notes,omitempty"`
	LastSyncedAt                 time.Time `gorm:"not null;index" json:"last_synced_at"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}
