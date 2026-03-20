package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/crypto"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/provider"
	"time"

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

var providerFamilyByType = func() map[string]string {
	index := make(map[string]string)
	for family, providers := range supportedProvidersByFamily {
		for providerType := range providers {
			index[providerType] = family
		}
	}
	return index
}()

var providerTypeAliases = map[string]string{
	"AWS":              domain.ProviderAWSS3,
	"S3":               domain.ProviderAWSS3,
	"CLOUDFLARE":       domain.ProviderCloudflare,
	"R2":               domain.ProviderCloudflare,
	"DIGITALOCEAN":     domain.ProviderDOSpaces,
	"DO_SPACES":        domain.ProviderDOSpaces,
	"BACKBLAZE":        domain.ProviderB2,
	"B2":               domain.ProviderB2,
	"GEMINI":           domain.ProviderGemini,
	"GOOGLE":           domain.ProviderGemini,
	"DEEPSEEK":         domain.ProviderDeepSeek,
	"OPEN_ROUTER":      domain.ProviderOpenRouter,
	"OPEN-ROUTER":      domain.ProviderOpenRouter,
	"ANTHROPIC":        domain.ProviderClaude,
	"CLAUDE_ANTHROPIC": domain.ProviderClaude,
}

const globalModelCatalogScope = "GLOBAL"

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

	var existingConfig *models.UserProviderConfig
	existingConfig, err = u.repo.GetConfig(ctx, userID, normalizedProviderType)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		existingConfig = nil
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

	operation := "CREATED"
	if existingConfig != nil {
		operation = "UPDATED"
	}

	auditEvent := buildProviderAuditEvent(ctx, config, "PROVIDER_CONFIG_SAVE", "SUCCESS", buildProviderConfigAuditDetail(normalizedFamily, normalizedProviderType, operation), map[string]any{
		"operation":          operation,
		"credential_fields":  sanitizeProviderCredentialKeys(credentials),
		"public_metadata":    sanitizeProviderPublicMetadata(normalizedFamily, credentials),
		"activation_state":   providerActivationMetadata(normalizedFamily, activation),
		"had_prior_config":   existingConfig != nil,
		"provider_scope_key": normalizedFamily + ":" + normalizedProviderType,
	})

	return u.repo.SaveConfig(ctx, config, &auditEvent)
}

func (u *providerUseCase) TestConnection(ctx context.Context, userID uint, family, providerType string, credentials map[string]string) error {
	normalizedFamily, normalizedProviderType, err := normalizeProviderSelection(family, providerType)
	if err != nil {
		return err
	}

	if err := u.validateCredentials(ctx, normalizedFamily, normalizedProviderType, credentials); err != nil {
		return err
	}

	if err := u.validateProviderConnection(ctx, normalizedFamily, normalizedProviderType, credentials); err != nil {
		return err
	}

	return nil
}

func (u *providerUseCase) ListModels(ctx context.Context, userID uint, family, providerType string, credentials map[string]string) ([]string, error) {
	normalizedFamily, normalizedProviderType, err := normalizeProviderSelection(family, providerType)
	if err != nil {
		return nil, err
	}
	if normalizedFamily != domain.FamilyLLM {
		return nil, domain.ErrProviderFamilyInvalid
	}

	hasDiscoveryOverrides := false
	for _, key := range []string{"base_url", "api_key", "model"} {
		if strings.TrimSpace(credentials[key]) != "" {
			hasDiscoveryOverrides = true
			break
		}
	}

	if !hasDiscoveryOverrides {
		if cachedModels, err := u.listCachedModels(ctx, userID, normalizedProviderType); err == nil && len(cachedModels) > 0 {
			return cachedModels, nil
		}
	}

	if err := u.validateModelDiscoveryCredentials(ctx, normalizedProviderType, credentials); err != nil {
		return nil, err
	}

	discoveryCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	models, err := provider.ListLLMModels(discoveryCtx, normalizedProviderType, credentials)
	if err != nil {
		if cachedModels, cacheErr := u.listCachedModels(ctx, userID, normalizedProviderType); cacheErr == nil && len(cachedModels) > 0 {
			return cachedModels, nil
		}

		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(discoveryCtx.Err(), context.DeadlineExceeded):
			return nil, domain.ErrProviderModelsTimeout
		case strings.Contains(strings.ToLower(err.Error()), "not implemented"):
			return nil, domain.ErrProviderModelsUnsupported
		default:
			return nil, fmt.Errorf("%w: %v", domain.ErrProviderModelsUnavailable, err)
		}
	}

	if err := u.saveModelCatalog(ctx, userID, normalizedProviderType, models); err != nil {
		log.Printf("provider model catalog save skipped for provider=%s user=%d: %v", normalizedProviderType, userID, err)
	}

	return models, nil
}

func (u *providerUseCase) validateModelDiscoveryCredentials(ctx context.Context, providerType string, credentials map[string]string) error {
	switch providerType {
	case domain.ProviderOllama:
		// Ollama Cloud exposes /api/tags publicly, so model discovery must not require api_key.
	default:
		if err := u.validateLLMCredentials(ctx, providerType, credentials); err != nil {
			return err
		}
	}

	if err := u.validateLLMBaseURL(ctx, credentials["base_url"]); err != nil {
		return domain.ErrProviderCredentialsInvalid
	}

	return nil
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

func (u *providerUseCase) ListProviderAuditEvents(ctx context.Context, userID uint, filter domain.ProviderAuditFilter) ([]models.ProviderAuditEvent, error) {
	filter.Family = strings.ToUpper(strings.TrimSpace(filter.Family))
	filter.ProviderType = normalizeProviderType(filter.ProviderType)
	filter.Action = strings.ToUpper(strings.TrimSpace(filter.Action))
	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	return u.repo.ListProviderAuditEvents(ctx, userID, filter)
}

func (u *providerUseCase) SyncModelCatalogs(ctx context.Context) error {
	for providerType := range publicModelCatalogProviders() {
		if err := u.syncProviderModelCatalog(ctx, 0, providerType, map[string]string{
			"base_url": publicProviderDefaultBaseURL(providerType),
		}); err != nil {
			log.Printf("public provider model catalog sync failed for provider=%s: %v", providerType, err)
		}
	}

	configs, err := u.repo.ListAllConfigs(ctx)
	if err != nil {
		return err
	}

	for i := range configs {
		config := configs[i]
		if config.Family != domain.FamilyLLM || isPublicModelCatalogProvider(config.ProviderType) {
			continue
		}

		credentials, err := u.decryptConfig(&config, config.UserID)
		if err != nil {
			log.Printf("provider model catalog decrypt failed for provider=%s user=%d: %v", config.ProviderType, config.UserID, err)
			continue
		}

		if err := u.syncProviderModelCatalog(ctx, config.UserID, config.ProviderType, credentials); err != nil {
			log.Printf("private provider model catalog sync failed for provider=%s user=%d: %v", config.ProviderType, config.UserID, err)
		}
	}

	return nil
}

func (u *providerUseCase) SetProviderEnabled(ctx context.Context, userID uint, family, providerType string, enabled bool) error {
	normalizedFamily, normalizedProviderType, err := normalizeProviderSelection(family, providerType)
	if err != nil {
		return err
	}

	existingConfig, err := u.repo.GetConfig(ctx, userID, normalizedProviderType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProviderCredentialsInvalid
		}
		return err
	}

	auditEvent := buildProviderAuditEvent(ctx, existingConfig, "PROVIDER_ACTIVATION_CHANGE", "SUCCESS", fmt.Sprintf("Provider %s %s para storage.", normalizedProviderType, enabledDisabledLabel(enabled)), map[string]any{
		"activation_target": "STORAGE",
		"previous_enabled":  existingConfig.IsEnabled,
		"current_enabled":   enabled,
	})

	if err := u.repo.SetConfigEnabled(ctx, userID, normalizedFamily, normalizedProviderType, enabled, &auditEvent); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProviderCredentialsInvalid
		}
		return err
	}

	return nil
}

func (u *providerUseCase) SetLLMUsageEnabled(ctx context.Context, userID uint, providerType, usage string, enabled bool) error {
	normalizedProviderType := normalizeProviderType(providerType)
	if _, ok := supportedProvidersByFamily[domain.FamilyLLM][normalizedProviderType]; !ok {
		return domain.ErrProviderTypeInvalid
	}

	normalizedUsage := strings.ToUpper(strings.TrimSpace(usage))
	if normalizedUsage != domain.LLMUsageIndexing && normalizedUsage != domain.LLMUsageChat {
		return domain.ErrProviderFamilyInvalid
	}

	existingConfig, err := u.repo.GetConfig(ctx, userID, normalizedProviderType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProviderCredentialsInvalid
		}
		return err
	}

	previousEnabled := existingConfig.EnabledForIndexing
	if normalizedUsage == domain.LLMUsageChat {
		previousEnabled = existingConfig.EnabledForChat
	}

	auditEvent := buildProviderAuditEvent(ctx, existingConfig, "PROVIDER_ACTIVATION_CHANGE", "SUCCESS", fmt.Sprintf("Provider %s %s para uso %s.", normalizedProviderType, enabledDisabledLabel(enabled), strings.ToLower(normalizedUsage)), map[string]any{
		"activation_target": strings.ToLower(normalizedUsage),
		"previous_enabled":  previousEnabled,
		"current_enabled":   enabled,
	})

	if err := u.repo.SetLLMConfigUsage(ctx, userID, normalizedProviderType, normalizedUsage, enabled, &auditEvent); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrProviderCredentialsInvalid
		}
		return err
	}

	return nil
}

func buildProviderAuditEvent(ctx context.Context, config *models.UserProviderConfig, action, status, detail string, metadata map[string]any) models.ProviderAuditEvent {
	eventMetadata := cloneMetadata(metadata)
	if eventMetadata == nil {
		eventMetadata = map[string]any{}
	}

	actorType := "USER"
	var actorID *uint
	var correlationID string
	var origin string

	if extracted, ok := domain.ExtractAuditContext(ctx); ok {
		if extracted.ActorType != "" {
			actorType = extracted.ActorType
		}
		actorID = extracted.ActorID
		correlationID = extracted.CorrelationID
		origin = extracted.Origin
		if extracted.RequestMethod != "" {
			eventMetadata["request_method"] = extracted.RequestMethod
		}
		if extracted.RoutePath != "" {
			eventMetadata["route_path"] = extracted.RoutePath
		}
		if extracted.RemoteAddr != "" {
			eventMetadata["remote_addr"] = extracted.RemoteAddr
		}
	}

	event := models.ProviderAuditEvent{
		ActorType:     actorType,
		ActorID:       actorID,
		Action:        action,
		Status:        status,
		CorrelationID: correlationID,
		Origin:        origin,
		Detail:        detail,
		Metadata:      eventMetadata,
	}

	if config != nil {
		event.UserID = config.UserID
		event.Family = config.Family
		event.ProviderType = config.ProviderType
		if config.ID != 0 {
			event.ProviderConfigID = &config.ID
		}
	}

	return event
}

func buildProviderConfigAuditDetail(family, providerType, operation string) string {
	return fmt.Sprintf("Configuracao do provider %s/%s %s com sucesso.", family, providerType, strings.ToLower(operation))
}

func sanitizeProviderCredentialKeys(credentials map[string]string) []string {
	keys := make([]string, 0, len(credentials))
	for key, value := range credentials {
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sanitizeProviderPublicMetadata(family string, credentials map[string]string) map[string]string {
	publicMetadata := map[string]string{}

	switch family {
	case domain.FamilyLLM:
		if model := strings.TrimSpace(credentials["model"]); model != "" {
			publicMetadata["model"] = model
		}
		if baseURL := strings.TrimSpace(credentials["base_url"]); baseURL != "" {
			publicMetadata["base_url"] = sanitizeProviderURL(baseURL)
		}
	case domain.FamilyStorage:
		if endpoint := strings.TrimSpace(credentials["endpoint"]); endpoint != "" {
			publicMetadata["endpoint"] = endpoint
		}
		if bucket := strings.TrimSpace(credentials["bucket_name"]); bucket != "" {
			publicMetadata["bucket_name"] = bucket
		}
		if region := strings.TrimSpace(credentials["region"]); region != "" {
			publicMetadata["region"] = region
		}
	}

	return publicMetadata
}

func sanitizeProviderURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return strings.TrimSpace(rawURL)
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func providerActivationMetadata(family string, activation domain.ProviderActivation) map[string]bool {
	if family == domain.FamilyStorage {
		return map[string]bool{
			"is_enabled": activation.IsEnabled,
		}
	}

	return map[string]bool{
		"enabled_for_indexing": activation.EnabledForIndexing,
		"enabled_for_chat":     activation.EnabledForChat,
	}
}

func enabledDisabledLabel(enabled bool) string {
	if enabled {
		return "habilitado"
	}

	return "desabilitado"
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
	normalizedProviderType := normalizeProviderType(providerType)
	inferredFamily, ok := providerFamilyByType[normalizedProviderType]
	if !ok {
		return "", "", domain.ErrProviderTypeInvalid
	}

	normalizedFamily := strings.ToUpper(strings.TrimSpace(family))
	if normalizedFamily == "" {
		return inferredFamily, normalizedProviderType, nil
	}
	if _, ok := supportedProvidersByFamily[normalizedFamily]; !ok {
		return "", "", domain.ErrProviderFamilyInvalid
	}
	if normalizedFamily != inferredFamily {
		return inferredFamily, normalizedProviderType, nil
	}

	return normalizedFamily, normalizedProviderType, nil
}

func normalizeProviderType(providerType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(providerType))
	if alias, ok := providerTypeAliases[normalized]; ok {
		return alias
	}
	return normalized
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
		if isOllamaCloudBaseURL(credentials["base_url"]) && strings.TrimSpace(credentials["api_key"]) == "" {
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

func (u *providerUseCase) validateProviderConnection(ctx context.Context, family, providerType string, credentials map[string]string) error {
	switch family {
	case domain.FamilyStorage:
		if err := provider.ValidateStorageConnection(ctx, providerType, credentials, provider.StorageProviderOptions{
			EndpointValidator: u.endpointValidator,
		}); err != nil {
			return classifyProviderConnectionError(family, providerType, err)
		}
	case domain.FamilyLLM:
		if err := provider.ValidateLLMConnection(ctx, providerType, credentials); err != nil {
			return classifyProviderConnectionError(family, providerType, err)
		}
	}

	return nil
}

func classifyProviderConnectionError(family, providerType string, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case isProviderTimeoutError(err):
		return fmt.Errorf("%w: %v", domain.ErrProviderConnectionTimeout, err)
	case isProviderAuthenticationError(err):
		return fmt.Errorf("%w: %v", domain.ErrProviderAuthenticationFailed, err)
	case family == domain.FamilyLLM && isProviderModelError(err):
		return fmt.Errorf("%w: %v", domain.ErrProviderModelInvalid, err)
	case isProviderEndpointError(err):
		return fmt.Errorf("%w: %v", domain.ErrProviderEndpointUnavailable, err)
	default:
		return fmt.Errorf("%w: %v", domain.ErrProviderConnectionFailed, err)
	}
}

func isProviderTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "timeout") || strings.Contains(lowerErr, "deadline exceeded")
}

func isProviderAuthenticationError(err error) bool {
	lowerErr := strings.ToLower(err.Error())
	patterns := []string{
		"invalid api key",
		"incorrect api key",
		"unauthorized",
		"forbidden",
		"authentication",
		"auth failed",
		"access denied",
		"invalid access key",
		"signature does not match",
		"security token",
		"invalid token",
		"permission denied",
	}

	return containsAnyPattern(lowerErr, patterns)
}

func isProviderModelError(err error) bool {
	lowerErr := strings.ToLower(err.Error())
	patterns := []string{
		"model not found",
		"unknown model",
		"invalid model",
		"unsupported model",
		"does not exist",
		"no such model",
		"model is required",
	}

	return containsAnyPattern(lowerErr, patterns)
}

func isProviderEndpointError(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	lowerErr := strings.ToLower(err.Error())
	patterns := []string{
		"no such host",
		"server misbehaving",
		"connection refused",
		"network is unreachable",
		"connection reset by peer",
		"broken pipe",
		"eof",
		"dial tcp",
		"lookup ",
		"tls",
		"x509",
		"certificate",
		"failed to resolve",
	}

	return containsAnyPattern(lowerErr, patterns)
}

func containsAnyPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}

	return false
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

	if isOllamaCloudBaseURL(trimmedURL) {
		return nil
	}

	if err := u.endpointValidator.Validate(ctx, parsed.Host); err != nil {
		return domain.ErrProviderCredentialsInvalid
	}

	return nil
}

func isOllamaCloudBaseURL(rawURL string) bool {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return false
	}

	parsed, err := url.Parse(trimmedURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return host == "ollama.com" || host == "www.ollama.com"
}

func (u *providerUseCase) listCachedModels(ctx context.Context, userID uint, providerType string) ([]string, error) {
	if u.repo == nil {
		return nil, nil
	}

	scopeKeys := modelCatalogScopeOrder(userID, providerType)
	for _, scopeKey := range scopeKeys {
		items, err := u.repo.ListModelCatalog(ctx, scopeKey, providerType)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			continue
		}

		models := make([]string, 0, len(items))
		for _, item := range items {
			modelID := strings.TrimSpace(item.ModelID)
			if modelID == "" {
				continue
			}
			models = append(models, modelID)
		}
		if len(models) > 0 {
			return models, nil
		}
	}

	return nil, nil
}

func (u *providerUseCase) saveModelCatalog(ctx context.Context, userID uint, providerType string, discovered []string) error {
	if u.repo == nil {
		return nil
	}

	scopeKey, scopedUserID := modelCatalogScope(userID, providerType)
	entries := make([]models.ProviderModelCatalog, 0, len(discovered))
	fetchedAt := time.Now().UTC()

	for _, item := range discovered {
		modelID := strings.TrimSpace(item)
		if modelID == "" {
			continue
		}

		entries = append(entries, models.ProviderModelCatalog{
			ScopeKey:     scopeKey,
			UserID:       scopedUserID,
			ProviderType: providerType,
			ModelID:      modelID,
			DisplayName:  modelID,
			FetchedAt:    fetchedAt,
			IsActive:     true,
		})
	}

	return u.repo.ReplaceModelCatalog(ctx, scopeKey, scopedUserID, providerType, entries)
}

func (u *providerUseCase) syncProviderModelCatalog(ctx context.Context, userID uint, providerType string, credentials map[string]string) error {
	if u.repo == nil {
		return nil
	}

	normalizedProviderType := normalizeProviderType(providerType)
	if err := u.validateModelDiscoveryCredentials(ctx, normalizedProviderType, credentials); err != nil {
		return err
	}

	discoveryCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	models, err := provider.ListLLMModels(discoveryCtx, normalizedProviderType, credentials)
	if err != nil {
		return err
	}

	return u.saveModelCatalog(ctx, userID, normalizedProviderType, models)
}

func publicModelCatalogProviders() map[string]struct{} {
	return map[string]struct{}{
		domain.ProviderOllama: {},
	}
}

func isPublicModelCatalogProvider(providerType string) bool {
	_, ok := publicModelCatalogProviders()[normalizeProviderType(providerType)]
	return ok
}

func publicProviderDefaultBaseURL(providerType string) string {
	switch normalizeProviderType(providerType) {
	case domain.ProviderOllama:
		return "https://ollama.com/api"
	default:
		return ""
	}
}

func modelCatalogScope(userID uint, providerType string) (string, *uint) {
	if isPublicModelCatalogProvider(providerType) {
		return globalModelCatalogScope, nil
	}

	scopeUserID := userID
	return "USER:" + strconv.FormatUint(uint64(userID), 10), &scopeUserID
}

func modelCatalogScopeOrder(userID uint, providerType string) []string {
	if isPublicModelCatalogProvider(providerType) {
		return []string{globalModelCatalogScope}
	}

	return []string{
		"USER:" + strconv.FormatUint(uint64(userID), 10),
		globalModelCatalogScope,
	}
}
