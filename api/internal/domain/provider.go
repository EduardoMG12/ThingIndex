package domain

import (
	"context"
	"thing-index/internal/models"
)

type ProviderRepository interface {
	SaveConfig(ctx context.Context, config *models.UserProviderConfig) error
	GetConfig(ctx context.Context, userID uint, providerType string) (*models.UserProviderConfig, error)
	ListConfigs(ctx context.Context, userID uint) ([]models.UserProviderConfig, error)
	DeleteConfig(ctx context.Context, userID uint, providerType string) error
}

type ProviderUseCase interface {
	SaveCredentials(ctx context.Context, userID uint, providerType string, credentials map[string]string) error
	GetDecryptedCredentials(ctx context.Context, userID uint, providerType string) (map[string]string, error)
	ListUserProviders(ctx context.Context, userID uint) ([]models.UserProviderConfig, error)
}

type LLMService interface {
	Summarize(ctx context.Context, text string) (string, error)
}
