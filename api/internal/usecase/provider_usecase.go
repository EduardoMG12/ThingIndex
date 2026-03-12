package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/crypto"
)

var supportedProvidersByFamily = map[string]map[string]struct{}{
	domain.FamilyStorage: {
		domain.ProviderMinio:      {},
		domain.ProviderAWSS3:      {},
		domain.ProviderGCPStorage: {},
		domain.ProviderCloudflare: {},
	},
	domain.FamilyLLM: {
		domain.ProviderOpenAI:   {},
		domain.ProviderGemini:   {},
		domain.ProviderDeepSeek: {},
		domain.ProviderGrok:     {},
		domain.ProviderClaude:   {},
	},
}

type providerUseCase struct {
	repo domain.ProviderRepository
	cfg  *config.Config
}

func NewProviderUseCase(repo domain.ProviderRepository, cfg *config.Config) domain.ProviderUseCase {
	return &providerUseCase{
		repo: repo,
		cfg:  cfg,
	}
}

func (u *providerUseCase) SaveCredentials(ctx context.Context, userID uint, family, providerType string, credentials map[string]string) error {
	normalizedFamily, normalizedProviderType, err := normalizeProviderSelection(family, providerType)
	if err != nil {
		return err
	}

	if err := validateCredentials(credentials); err != nil {
		return err
	}

	jsonData, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	encrypted, err := crypto.Encrypt(string(jsonData), u.cfg.MasterEncryptionKey, userID)
	if err != nil {
		return err
	}

	config := &models.UserProviderConfig{
		UserID:        userID,
		Family:        normalizedFamily,
		ProviderType:  normalizedProviderType,
		EncryptedData: encrypted,
		IsEnabled:     true,
	}

	return u.repo.SaveConfig(ctx, config)
}

func (u *providerUseCase) GetDecryptedCredentials(ctx context.Context, userID uint, providerType string) (map[string]string, error) {
	config, err := u.repo.GetConfig(ctx, userID, strings.ToUpper(strings.TrimSpace(providerType)))
	if err != nil {
		return nil, err
	}
	return u.decryptConfig(config, userID)
}

func (u *providerUseCase) GetFamilyCredentials(ctx context.Context, userID uint, family string) (string, map[string]string, error) {
	normalizedFamily := strings.ToUpper(strings.TrimSpace(family))
	config, err := u.repo.GetConfigByFamily(ctx, userID, normalizedFamily)
	if err != nil {
		return "", nil, err
	}
	creds, err := u.decryptConfig(config, userID)
	if err != nil {
		return "", nil, err
	}
	return config.ProviderType, creds, nil
}

func (u *providerUseCase) decryptConfig(config *models.UserProviderConfig, userID uint) (map[string]string, error) {
	decrypted, err := crypto.Decrypt(config.EncryptedData, u.cfg.MasterEncryptionKey, userID)
	if err != nil {
		return nil, err
	}

	var credentials map[string]string
	if err := json.Unmarshal([]byte(decrypted), &credentials); err != nil {
		return nil, err
	}

	return credentials, nil
}

func (u *providerUseCase) ListUserProviders(ctx context.Context, userID uint) ([]models.UserProviderConfig, error) {
	return u.repo.ListConfigs(ctx, userID)
}

func normalizeProviderSelection(family, providerType string) (string, string, error) {
	normalizedFamily := strings.ToUpper(strings.TrimSpace(family))
	allowedProviders, ok := supportedProvidersByFamily[normalizedFamily]
	if !ok {
		return "", "", domain.ErrProviderFamilyInvalid
	}

	normalizedProviderType := strings.ToUpper(strings.TrimSpace(providerType))
	if _, ok := allowedProviders[normalizedProviderType]; !ok {
		return "", "", domain.ErrProviderTypeInvalid
	}

	return normalizedFamily, normalizedProviderType, nil
}

func validateCredentials(credentials map[string]string) error {
	if len(credentials) == 0 {
		return domain.ErrProviderCredentialsInvalid
	}

	for key, value := range credentials {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return domain.ErrProviderCredentialsInvalid
		}
	}

	return nil
}
