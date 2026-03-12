package domain

import (
	"context"
	"thing-index/internal/models"
)

const (
	// Families
	FamilyStorage = "STORAGE"
	FamilyLLM     = "LLM"

	// Storage Providers
	ProviderMinio      = "MINIO"
	ProviderAWSS3      = "AWS_S3"
	ProviderGCPStorage = "GCP_STORAGE"
	ProviderCloudflare = "CLOUDFLARE_R2"

	// LLM Providers
	ProviderOpenAI   = "OPENAI"
	ProviderGemini   = "GOOGLE_GEMINI"
	ProviderDeepSeek = "DEEP_SEEK"
	ProviderGrok     = "GROK"
	ProviderClaude   = "CLAUDE"
)

type ProviderRepository interface {
	SaveConfig(ctx context.Context, config *models.UserProviderConfig) error
	GetConfig(ctx context.Context, userID uint, providerType string) (*models.UserProviderConfig, error)
	GetConfigByFamily(ctx context.Context, userID uint, family string) (*models.UserProviderConfig, error)
	ListConfigs(ctx context.Context, userID uint) ([]models.UserProviderConfig, error)
	DeleteConfig(ctx context.Context, userID uint, providerType string) error
}

type ProviderUseCase interface {
	SaveCredentials(ctx context.Context, userID uint, family, providerType string, credentials map[string]string) error
	GetDecryptedCredentials(ctx context.Context, userID uint, providerType string) (map[string]string, error)
	ListUserProviders(ctx context.Context, userID uint) ([]models.UserProviderConfig, error)
}

type ProviderService interface {
	ProviderUseCase
	UserStorageResolver
}

type LLMService interface {
	Summarize(ctx context.Context, text string) (string, error)
}
