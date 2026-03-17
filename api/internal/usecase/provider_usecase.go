package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/crypto"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/provider"

	"gorm.io/gorm"
)

var supportedProvidersByFamily = map[string]map[string]struct{}{
	domain.FamilyStorage: {
		domain.ProviderMinio:      {},
		domain.ProviderAWSS3:      {},
		domain.ProviderCloudflare: {},
		domain.ProviderDOSpaces:   {},
		domain.ProviderWasabi:     {},
		domain.ProviderB2:         {},
	},
	domain.FamilyLLM: {
		domain.ProviderOpenAI:     {},
		domain.ProviderGemini:     {},
		domain.ProviderDeepSeek:   {},
		domain.ProviderGrok:       {},
		domain.ProviderClaude:     {},
		domain.ProviderOllama:     {},
		domain.ProviderOpenRouter: {},
	},
}

type providerUseCase struct {
	repo              domain.ProviderRepository
	cfg               *config.Config
	endpointValidator *netsec.StorageEndpointValidator
}

func NewProviderUseCase(repo domain.ProviderRepository, cfg *config.Config) domain.ProviderService {
	return &providerUseCase{
		repo:              repo,
		cfg:               cfg,
		endpointValidator: netsec.NewStorageEndpointValidator(cfg.AllowPrivateStorageEndpoints),
	}
}

func (u *providerUseCase) SaveCredentials(ctx context.Context, userID uint, family, providerType string, credentials map[string]string, activation domain.ProviderActivation) error {
	normalizedFamily, normalizedProviderType, err := normalizeProviderSelection(family, providerType)
	if err != nil {
		return err
	}

	if err := u.validateCredentials(ctx, normalizedFamily, normalizedProviderType, credentials); err != nil {
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
		UserID:             userID,
		Family:             normalizedFamily,
		ProviderType:       normalizedProviderType,
		EncryptedData:      encrypted,
		IsEnabled:          normalizedFamily == domain.FamilyStorage && activation.IsEnabled,
		EnabledForIndexing: normalizedFamily == domain.FamilyLLM && activation.EnabledForIndexing,
		EnabledForChat:     normalizedFamily == domain.FamilyLLM && activation.EnabledForChat,
	}

	if normalizedFamily == domain.FamilyStorage && !activation.IsEnabled {
		config.IsEnabled = true
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

func (u *providerUseCase) SetProviderEnabled(ctx context.Context, userID uint, family, providerType string, enabled bool) error {
	normalizedFamily, normalizedProviderType, err := normalizeProviderSelection(family, providerType)
	if err != nil {
		return err
	}

	if err := u.repo.SetConfigEnabled(ctx, userID, normalizedFamily, normalizedProviderType, enabled); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProviderCredentialsInvalid
		}
		return err
	}

	return nil
}

func (u *providerUseCase) SetLLMUsageEnabled(ctx context.Context, userID uint, providerType, usage string, enabled bool) error {
	normalizedProviderType := strings.ToUpper(strings.TrimSpace(providerType))
	if _, ok := supportedProvidersByFamily[domain.FamilyLLM][normalizedProviderType]; !ok {
		return domain.ErrProviderTypeInvalid
	}

	normalizedUsage := strings.ToUpper(strings.TrimSpace(usage))
	if normalizedUsage != domain.LLMUsageIndexing && normalizedUsage != domain.LLMUsageChat {
		return domain.ErrProviderFamilyInvalid
	}

	if err := u.repo.SetLLMConfigUsage(ctx, userID, normalizedProviderType, normalizedUsage, enabled); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProviderCredentialsInvalid
		}
		return err
	}

	return nil
}

func (u *providerUseCase) ResolveStorage(ctx context.Context, userID uint) (domain.StorageService, error) {
	config, err := u.repo.GetConfigByFamily(ctx, userID, domain.FamilyStorage)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrStorageProviderNotConfigured
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrStorageProviderUnavailable, err)
	}

	creds, err := u.decryptConfig(config, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStorageProviderUnavailable, err)
	}
	if err := u.validateCredentials(ctx, config.Family, config.ProviderType, creds); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStorageProviderUnavailable, err)
	}

	storageService, err := provider.NewStorageProvider(config.ProviderType, creds, provider.StorageProviderOptions{
		EndpointValidator: u.endpointValidator,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStorageProviderUnavailable, err)
	}

	return storageService, nil
}

func (u *providerUseCase) ResolveLLMForIndexing(ctx context.Context, userID uint) (domain.LLMService, error) {
	return u.resolveLLMByUsage(ctx, userID, domain.LLMUsageIndexing)
}

func (u *providerUseCase) ResolveLLMForChat(ctx context.Context, userID uint) (domain.LLMService, error) {
	return u.resolveLLMByUsage(ctx, userID, domain.LLMUsageChat)
}

func (u *providerUseCase) resolveLLMByUsage(ctx context.Context, userID uint, usage string) (domain.LLMService, error) {
	config, err := u.repo.GetConfigByLLMUsage(ctx, userID, usage)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLLMProviderNotConfigured
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrLLMProviderUnavailable, err)
	}

	creds, err := u.decryptConfig(config, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrLLMProviderUnavailable, err)
	}
	if err := u.validateCredentials(ctx, config.Family, config.ProviderType, creds); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrLLMProviderUnavailable, err)
	}

	llmService, err := provider.NewLLMProvider(config.ProviderType, creds)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrLLMProviderUnavailable, err)
	}

	return llmService, nil
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

func (u *providerUseCase) validateCredentials(ctx context.Context, family, providerType string, credentials map[string]string) error {
	if len(credentials) == 0 {
		return domain.ErrProviderCredentialsInvalid
	}

	for key, value := range credentials {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return domain.ErrProviderCredentialsInvalid
		}
	}

	switch family {
	case domain.FamilyStorage:
		switch providerType {
		case domain.ProviderMinio:
			for _, requiredKey := range []string{"endpoint", "access_key", "secret_key", "bucket_name"} {
				if strings.TrimSpace(credentials[requiredKey]) == "" {
					return domain.ErrProviderCredentialsInvalid
				}
			}
			if err := u.endpointValidator.Validate(ctx, credentials["endpoint"]); err != nil {
				return domain.ErrProviderCredentialsInvalid
			}
		case domain.ProviderAWSS3:
			for _, requiredKey := range []string{"access_key", "secret_key", "bucket_name", "region"} {
				if strings.TrimSpace(credentials[requiredKey]) == "" {
					return domain.ErrProviderCredentialsInvalid
				}
			}
			if endpoint := strings.TrimSpace(credentials["endpoint"]); endpoint != "" {
				if err := u.endpointValidator.Validate(ctx, endpoint); err != nil {
					return domain.ErrProviderCredentialsInvalid
				}
			}
		case domain.ProviderCloudflare:
			for _, requiredKey := range []string{"account_id", "access_key", "secret_key", "bucket_name"} {
				if strings.TrimSpace(credentials[requiredKey]) == "" {
					return domain.ErrProviderCredentialsInvalid
				}
			}
			if endpoint := strings.TrimSpace(credentials["endpoint"]); endpoint != "" {
				if err := u.endpointValidator.Validate(ctx, endpoint); err != nil {
					return domain.ErrProviderCredentialsInvalid
				}
			}
		case domain.ProviderDOSpaces, domain.ProviderWasabi, domain.ProviderB2:
			for _, requiredKey := range []string{"endpoint", "access_key", "secret_key", "bucket_name", "region"} {
				if strings.TrimSpace(credentials[requiredKey]) == "" {
					return domain.ErrProviderCredentialsInvalid
				}
			}
			if err := u.endpointValidator.Validate(ctx, credentials["endpoint"]); err != nil {
				return domain.ErrProviderCredentialsInvalid
			}
		}
	case domain.FamilyLLM:
		if err := u.validateLLMCredentials(ctx, providerType, credentials); err != nil {
			return err
		}
	}

	return nil
}

func (u *providerUseCase) validateLLMCredentials(ctx context.Context, providerType string, credentials map[string]string) error {
	switch providerType {
	case domain.ProviderOpenAI, domain.ProviderDeepSeek, domain.ProviderGrok:
		if strings.TrimSpace(credentials["api_key"]) == "" {
			return domain.ErrProviderCredentialsInvalid
		}
	case domain.ProviderGemini, domain.ProviderClaude:
		if strings.TrimSpace(credentials["api_key"]) == "" || strings.TrimSpace(credentials["model"]) == "" || strings.TrimSpace(credentials["base_url"]) == "" {
			return domain.ErrProviderCredentialsInvalid
		}
	case domain.ProviderOpenRouter:
		if strings.TrimSpace(credentials["api_key"]) == "" || strings.TrimSpace(credentials["model"]) == "" {
			return domain.ErrProviderCredentialsInvalid
		}
	case domain.ProviderOllama:
		if strings.TrimSpace(credentials["model"]) == "" {
			return domain.ErrProviderCredentialsInvalid
		}
	default:
		return domain.ErrProviderTypeInvalid
	}

	if err := u.validateLLMBaseURL(ctx, credentials["base_url"]); err != nil {
		return domain.ErrProviderCredentialsInvalid
	}

	return nil
}

func (u *providerUseCase) validateLLMBaseURL(ctx context.Context, rawURL string) error {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return nil
	}

	parsed, err := url.Parse(trimmedURL)
	if err != nil {
		return err
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return domain.ErrProviderCredentialsInvalid
	}
	if parsed.Host == "" {
		return domain.ErrProviderCredentialsInvalid
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme == "" {
		return domain.ErrProviderCredentialsInvalid
	}
	if u.cfg != nil && u.cfg.IsDevelopment() {
		if scheme != "https" && scheme != "http" {
			return domain.ErrProviderCredentialsInvalid
		}
	} else if scheme != "https" {
		return domain.ErrProviderCredentialsInvalid
	}

	if err := u.endpointValidator.Validate(ctx, parsed.Host); err != nil {
		return domain.ErrProviderCredentialsInvalid
	}

	return nil
}
