package models

import "time"

type UserProviderConfig struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	UserID             uint      `gorm:"not null;uniqueIndex:idx_user_provider,priority:1" json:"user_id"`
	Family             string    `gorm:"size:20;not null;index" json:"family"`                                           // e.g., 'STORAGE', 'LLM'
	ProviderType       string    `gorm:"size:50;not null;uniqueIndex:idx_user_provider,priority:2" json:"provider_type"` // e.g., 'OPENAI', 'AWS_S3'
	EncryptedData      string    `gorm:"type:text;not null" json:"encrypted_data"`                                       // JSON encrypted with AES
	IsEnabled          bool      `gorm:"default:true" json:"is_enabled"`
	EnabledForIndexing bool      `gorm:"default:false" json:"enabled_for_indexing"`
	EnabledForChat     bool      `gorm:"default:false" json:"enabled_for_chat"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ProviderModelCatalog struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	ScopeKey     string    `gorm:"size:64;not null;uniqueIndex:idx_provider_model_catalog_scope,priority:1;index" json:"scope_key"`
	UserID       *uint     `gorm:"index" json:"user_id,omitempty"`
	ProviderType string    `gorm:"size:50;not null;uniqueIndex:idx_provider_model_catalog_scope,priority:2;index" json:"provider_type"`
	ModelID      string    `gorm:"size:255;not null;uniqueIndex:idx_provider_model_catalog_scope,priority:3" json:"model_id"`
	DisplayName  string    `gorm:"size:255;not null" json:"display_name"`
	MetadataJSON string    `gorm:"type:text" json:"metadata_json,omitempty"`
	FetchedAt    time.Time `gorm:"not null;index" json:"fetched_at"`
	IsActive     bool      `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
