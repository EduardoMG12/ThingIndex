package provider

import (
	"fmt"

	"thing-index/internal/config"
	"thing-index/internal/domain"
)

func NewDefaultStorageProvider(cfg *config.Config) (domain.StorageService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	providerType, credentials, err := cfg.DefaultStorageCredentials()
	if err != nil {
		return nil, err
	}

	return NewStorageProvider(providerType, credentials, StorageProviderOptions{})
}
