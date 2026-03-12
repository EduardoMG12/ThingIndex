package usecase

import (
	"context"
	"encoding/json"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/crypto"
)

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
	jsonData, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	encrypted, err := crypto.Encrypt(string(jsonData), u.cfg.MasterEncryptionKey)
	if err != nil {
		return err
	}

	config := &models.UserProviderConfig{
		UserID:        userID,
		Family:        family,
		ProviderType:  providerType,
		EncryptedData: encrypted,
		IsEnabled:     true,
	}

	return u.repo.SaveConfig(ctx, config)
}

func (u *providerUseCase) GetDecryptedCredentials(ctx context.Context, userID uint, providerType string) (map[string]string, error) {
	config, err := u.repo.GetConfig(ctx, userID, providerType)
	if err != nil {
		return nil, err
	}
	return u.decryptConfig(config)
}

func (u *providerUseCase) GetFamilyCredentials(ctx context.Context, userID uint, family string) (string, map[string]string, error) {
	config, err := u.repo.GetConfigByFamily(ctx, userID, family)
	if err != nil {
		return "", nil, err
	}
	creds, err := u.decryptConfig(config)
	if err != nil {
		return "", nil, err
	}
	return config.ProviderType, creds, nil
}

func (u *providerUseCase) decryptConfig(config *models.UserProviderConfig) (map[string]string, error) {
	decrypted, err := crypto.Decrypt(config.EncryptedData, u.cfg.MasterEncryptionKey)
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
