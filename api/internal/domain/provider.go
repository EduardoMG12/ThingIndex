package domain

import (
	"context"
	"thing-index/internal/models"
)

const (
	// Families
	FamilyStorage = "STORAGE"
	FamilyLLM     = "LLM"

	LLMUsageIndexing = "INDEXING"
	LLMUsageChat     = "CHAT"

	// Storage Providers
	ProviderMinio      = "MINIO"
	ProviderAWSS3      = "AWS_S3"
	ProviderGCPStorage = "GCP_STORAGE"
	ProviderCloudflare = "CLOUDFLARE_R2"
	ProviderDOSpaces   = "DIGITALOCEAN_SPACES"
	ProviderWasabi     = "WASABI"
	ProviderB2         = "BACKBLAZE_B2"

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
	GetConfigByLLMUsage(ctx context.Context, userID uint, usage string) (*models.UserProviderConfig, error)
	ListConfigs(ctx context.Context, userID uint) ([]models.UserProviderConfig, error)
	SetConfigEnabled(ctx context.Context, userID uint, family, providerType string, enabled bool) error
	SetLLMConfigUsage(ctx context.Context, userID uint, providerType, usage string, enabled bool) error
	DeleteConfig(ctx context.Context, userID uint, providerType string) error
}

type ProviderUseCase interface {
	SaveCredentials(ctx context.Context, userID uint, family, providerType string, credentials map[string]string, activation ProviderActivation) error
	GetDecryptedCredentials(ctx context.Context, userID uint, providerType string) (map[string]string, error)
	ListUserProviders(ctx context.Context, userID uint) ([]models.UserProviderConfig, error)
	SetProviderEnabled(ctx context.Context, userID uint, family, providerType string, enabled bool) error
	SetLLMUsageEnabled(ctx context.Context, userID uint, providerType, usage string, enabled bool) error
}

type ProviderService interface {
	ProviderUseCase
	UserStorageResolver
	UserLLMResolver
}

type LLMService interface {
	GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type UserLLMResolver interface {
	ResolveLLMForIndexing(ctx context.Context, userID uint) (LLMService, error)
	ResolveLLMForChat(ctx context.Context, userID uint) (LLMService, error)
}

type ProviderActivation struct {
	IsEnabled          bool
	EnabledForIndexing bool
	EnabledForChat     bool
}
