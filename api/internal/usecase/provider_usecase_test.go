package usecase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/provider"

	"gorm.io/gorm"
)

type providerRepoStub struct {
	savedConfigs      []*models.UserProviderConfig
	savedConfigAudits []models.ProviderAuditEvent
	recordedAudits    []models.ProviderAuditEvent
	modelCatalog      map[string][]models.ProviderModelCatalog
	allConfigs        []models.UserProviderConfig
	configByProvider  map[string]*models.UserProviderConfig
	setEnabledCalls   []providerEnabledCall
	setLLMUsageCalls  []providerUsageCall
}

type providerEnabledCall struct {
	userID       uint
	family       string
	providerType string
	enabled      bool
	auditEvent   *models.ProviderAuditEvent
}

type providerUsageCall struct {
	userID       uint
	providerType string
	usage        string
	enabled      bool
	auditEvent   *models.ProviderAuditEvent
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (r *providerRepoStub) SaveConfig(_ context.Context, config *models.UserProviderConfig, auditEvent *models.ProviderAuditEvent) error {
	if config.ID == 0 {
		config.ID = uint(len(r.savedConfigs) + 1)
	}
	r.savedConfigs = append(r.savedConfigs, config)
	if r.configByProvider == nil {
		r.configByProvider = map[string]*models.UserProviderConfig{}
	}
	cloned := *config
	r.configByProvider[config.ProviderType] = &cloned
	if auditEvent != nil {
		copied := *auditEvent
		r.savedConfigAudits = append(r.savedConfigAudits, copied)
	}
	return nil
}

func (r *providerRepoStub) CreateProviderAuditEvents(_ context.Context, events []models.ProviderAuditEvent) error {
	r.recordedAudits = append(r.recordedAudits, events...)
	return nil
}

func (r *providerRepoStub) GetConfig(_ context.Context, _ uint, providerType string) (*models.UserProviderConfig, error) {
	if r.configByProvider == nil {
		return nil, gorm.ErrRecordNotFound
	}
	config := r.configByProvider[providerType]
	if config == nil {
		return nil, gorm.ErrRecordNotFound
	}
	cloned := *config
	return &cloned, nil
}

func (r *providerRepoStub) GetConfigByFamily(context.Context, uint, string) (*models.UserProviderConfig, error) {
	return nil, nil
}

func (r *providerRepoStub) GetConfigByLLMUsage(context.Context, uint, string) (*models.UserProviderConfig, error) {
	return nil, nil
}

func (r *providerRepoStub) ListConfigs(context.Context, uint) ([]models.UserProviderConfig, error) {
	return nil, nil
}

func (r *providerRepoStub) ListAllConfigs(context.Context) ([]models.UserProviderConfig, error) {
	return r.allConfigs, nil
}

func (r *providerRepoStub) ListModelCatalog(_ context.Context, scopeKey, providerType string) ([]models.ProviderModelCatalog, error) {
	if r.modelCatalog == nil {
		return nil, nil
	}
	return r.modelCatalog[scopeKey+":"+providerType], nil
}

func (r *providerRepoStub) ReplaceModelCatalog(_ context.Context, scopeKey string, userID *uint, providerType string, entries []models.ProviderModelCatalog) error {
	if r.modelCatalog == nil {
		r.modelCatalog = make(map[string][]models.ProviderModelCatalog)
	}
	r.modelCatalog[scopeKey+":"+providerType] = entries
	return nil
}

func (r *providerRepoStub) SetConfigEnabled(_ context.Context, userID uint, family, providerType string, enabled bool, auditEvent *models.ProviderAuditEvent) error {
	call := providerEnabledCall{
		userID:       userID,
		family:       family,
		providerType: providerType,
		enabled:      enabled,
	}
	if auditEvent != nil {
		copied := *auditEvent
		call.auditEvent = &copied
	}
	r.setEnabledCalls = append(r.setEnabledCalls, call)
	if r.configByProvider == nil {
		return gorm.ErrRecordNotFound
	}
	config := r.configByProvider[providerType]
	if config == nil {
		return gorm.ErrRecordNotFound
	}
	config.IsEnabled = enabled
	return nil
}

func (r *providerRepoStub) SetLLMConfigUsage(_ context.Context, userID uint, providerType, usage string, enabled bool, auditEvent *models.ProviderAuditEvent) error {
	call := providerUsageCall{
		userID:       userID,
		providerType: providerType,
		usage:        usage,
		enabled:      enabled,
	}
	if auditEvent != nil {
		copied := *auditEvent
		call.auditEvent = &copied
	}
	r.setLLMUsageCalls = append(r.setLLMUsageCalls, call)
	if r.configByProvider == nil {
		return gorm.ErrRecordNotFound
	}
	config := r.configByProvider[providerType]
	if config == nil {
		return gorm.ErrRecordNotFound
	}
	switch usage {
	case domain.LLMUsageIndexing:
		config.EnabledForIndexing = enabled
	case domain.LLMUsageChat:
		config.EnabledForChat = enabled
	}
	return nil
}

func (r *providerRepoStub) DeleteConfig(context.Context, uint, string) error {
	return nil
}

func TestValidateCredentialsRejectsPrivateLLMBaseURL(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "secret",
		"base_url": "https://127.0.0.1:11434/v1",
	})
	if err == nil {
		t.Fatal("expected private llm base url to be rejected")
	}
}

func TestValidateCredentialsAcceptsPublicHTTPSLLMBaseURL(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "secret",
		"base_url": "https://8.8.8.8/v1",
	})
	if err != nil {
		t.Fatalf("expected public https llm base url to be accepted, got %v", err)
	}
}

func TestValidateCredentialsRejectsHTTPBaseURLOutsideDevelopment(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "secret",
		"base_url": "http://8.8.8.8/v1",
	})
	if err == nil {
		t.Fatal("expected http llm base url to be rejected outside development")
	}
}

func TestValidateCredentialsAllowsOllamaWithoutAPIKey(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"model":    "llama3.1:8b",
		"base_url": "http://127.0.0.1:11434/v1",
	})
	if err != nil {
		t.Fatalf("expected ollama credentials without api key to be accepted, got %v", err)
	}
}

func TestValidateCredentialsRequiresAPIKeyForOllamaCloud(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"model":    "deepseek-v3.1:671b",
		"base_url": "https://ollama.com/v1",
	})
	if err == nil {
		t.Fatal("expected ollama cloud credentials without api key to be rejected")
	}
}

func TestValidateCredentialsAcceptsOllamaCloudWithAPIKey(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"api_key": "secret",
		"model":   "deepseek-v3.1:671b",
	})
	if err != nil {
		t.Fatalf("expected ollama cloud credentials with api key to be accepted, got %v", err)
	}
}

func TestValidateCredentialsRejectsOpenRouterWithoutModel(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenRouter, map[string]string{
		"api_key":  "secret",
		"base_url": "https://openrouter.ai/api/v1",
	})
	if err == nil {
		t.Fatal("expected openrouter credentials without model to be rejected")
	}
}

func TestNormalizeProviderSelectionInfersFamilyFromProviderType(t *testing.T) {
	t.Parallel()

	family, providerType, err := normalizeProviderSelection(domain.FamilyStorage, domain.ProviderOllama)
	if err != nil {
		t.Fatalf("expected ollama provider type to be accepted, got %v", err)
	}
	if family != domain.FamilyLLM {
		t.Fatalf("expected family %q, got %q", domain.FamilyLLM, family)
	}
	if providerType != domain.ProviderOllama {
		t.Fatalf("expected provider type %q, got %q", domain.ProviderOllama, providerType)
	}
}

func TestNormalizeProviderSelectionAcceptsLegacyAliases(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"open_router": domain.ProviderOpenRouter,
		"deepseek":    domain.ProviderDeepSeek,
		"gemini":      domain.ProviderGemini,
		"b2":          domain.ProviderB2,
	}

	for input, expected := range testCases {
		input := input
		expected := expected

		t.Run(input, func(t *testing.T) {
			t.Parallel()

			_, providerType, err := normalizeProviderSelection("", input)
			if err != nil {
				t.Fatalf("expected alias %q to be accepted, got %v", input, err)
			}
			if providerType != expected {
				t.Fatalf("expected canonical provider type %q, got %q", expected, providerType)
			}
		})
	}
}

func TestNormalizeProviderTypeAcceptsLegacyAlias(t *testing.T) {
	t.Parallel()

	got := normalizeProviderType("open_router")
	if got != domain.ProviderOpenRouter {
		t.Fatalf("expected %q, got %q", domain.ProviderOpenRouter, got)
	}
}

func TestSaveCredentialsPersistsLLMEvenWhenConnectionValidationFails(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"invalid api key"}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	repo := &providerRepoStub{}
	useCase := &providerUseCase{
		repo:              repo,
		cfg:               &config.Config{Environment: "development", MasterEncryptionKey: "0123456789abcdef0123456789abcdef"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.SaveCredentials(context.Background(), 7, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "bad-key",
		"model":    "gpt-4o-mini",
		"base_url": "http://8.8.8.8/v1",
	}, domain.ProviderActivation{EnabledForIndexing: true})
	if err != nil {
		t.Fatalf("expected credentials to be saved without live connection validation, got %v", err)
	}
	if len(repo.savedConfigs) != 1 {
		t.Fatalf("expected one saved config, got %d", len(repo.savedConfigs))
	}
	if !repo.savedConfigs[0].EnabledForIndexing {
		t.Fatal("expected saved config to keep indexing activation")
	}
}

func TestSaveCredentialsPersistsProviderAuditWithoutSecrets(t *testing.T) {
	t.Parallel()

	repo := &providerRepoStub{}
	actorID := uint(7)
	useCase := &providerUseCase{
		repo:              repo,
		cfg:               &config.Config{Environment: "development", MasterEncryptionKey: "0123456789abcdef0123456789abcdef"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	ctx := domain.WithAuditContext(context.Background(), domain.AuditContext{
		ActorType:     "USER",
		ActorID:       &actorID,
		CorrelationID: "req-provider-save-001",
		Origin:        "POST /api/v1/user/providers",
		RequestMethod: "POST",
		RoutePath:     "/api/v1/user/providers",
		RemoteAddr:    "127.0.0.1",
	})

	err := useCase.SaveCredentials(ctx, 7, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "top-secret",
		"model":   "gpt-4o-mini",
	}, domain.ProviderActivation{EnabledForIndexing: true})
	if err != nil {
		t.Fatalf("expected provider save to succeed, got %v", err)
	}
	if len(repo.savedConfigAudits) != 1 {
		t.Fatalf("expected one provider audit event, got %d", len(repo.savedConfigAudits))
	}
	audit := repo.savedConfigAudits[0]
	if audit.Action != "PROVIDER_CONFIG_SAVE" || audit.Status != "SUCCESS" {
		t.Fatalf("unexpected provider audit event: %#v", audit)
	}
	if audit.CorrelationID != "req-provider-save-001" || audit.Origin != "POST /api/v1/user/providers" {
		t.Fatalf("expected correlation and origin metadata, got %#v", audit)
	}
	credentialFields, ok := audit.Metadata["credential_fields"].([]string)
	if !ok || len(credentialFields) != 2 {
		t.Fatalf("expected credential field names only, got %#v", audit.Metadata["credential_fields"])
	}
	publicMetadata, ok := audit.Metadata["public_metadata"].(map[string]string)
	if !ok {
		t.Fatalf("expected public metadata map, got %#v", audit.Metadata["public_metadata"])
	}
	if publicMetadata["model"] != "gpt-4o-mini" {
		t.Fatalf("expected model metadata, got %#v", publicMetadata)
	}
}

func TestTestConnectionClassifiesAuthenticationFailures(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"invalid api key"}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.TestConnection(context.Background(), 1, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "bad-key",
		"model":   "gpt-4o-mini",
	})
	if !errors.Is(err, domain.ErrProviderAuthenticationFailed) {
		t.Fatalf("expected authentication failure, got %v", err)
	}
}

func TestTestConnectionClassifiesModelFailures(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"model not found"}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.TestConnection(context.Background(), 1, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "secret",
		"model":   "missing-model",
	})
	if !errors.Is(err, domain.ErrProviderModelInvalid) {
		t.Fatalf("expected invalid model classification, got %v", err)
	}
}

func TestTestConnectionClassifiesTimeoutFailures(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.TestConnection(context.Background(), 1, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "secret",
		"model":   "gpt-4o-mini",
	})
	if !errors.Is(err, domain.ErrProviderConnectionTimeout) {
		t.Fatalf("expected timeout classification, got %v", err)
	}
}

func TestTestConnectionClassifiesEndpointFailures(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, &url.Error{
					Op:  "Post",
					URL: r.URL.String(),
					Err: errors.New("dial tcp: lookup api.openai.com: no such host"),
				}
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.TestConnection(context.Background(), 1, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "secret",
		"model":   "gpt-4o-mini",
	})
	if !errors.Is(err, domain.ErrProviderEndpointUnavailable) {
		t.Fatalf("expected endpoint failure classification, got %v", err)
	}
}

func TestTestConnectionClassifiesGenericProviderFailures(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unexpected upstream failure"}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.TestConnection(context.Background(), 1, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "secret",
		"model":   "gpt-4o-mini",
	})
	if !errors.Is(err, domain.ErrProviderConnectionFailed) {
		t.Fatalf("expected generic provider failure classification, got %v", err)
	}
}

func TestSetProviderEnabledPersistsActivationAudit(t *testing.T) {
	t.Parallel()

	repo := &providerRepoStub{
		configByProvider: map[string]*models.UserProviderConfig{
			domain.ProviderMinio: {
				ID:           11,
				UserID:       3,
				Family:       domain.FamilyStorage,
				ProviderType: domain.ProviderMinio,
				IsEnabled:    false,
			},
		},
	}
	actorID := uint(3)
	useCase := &providerUseCase{repo: repo}

	ctx := domain.WithAuditContext(context.Background(), domain.AuditContext{
		ActorType:     "USER",
		ActorID:       &actorID,
		CorrelationID: "req-provider-toggle-001",
		Origin:        "PATCH /api/v1/user/providers/:providerType",
		RequestMethod: "PATCH",
		RoutePath:     "/api/v1/user/providers/:providerType",
	})

	if err := useCase.SetProviderEnabled(ctx, 3, domain.FamilyStorage, domain.ProviderMinio, true); err != nil {
		t.Fatalf("expected storage activation update to succeed, got %v", err)
	}
	if len(repo.setEnabledCalls) != 1 || repo.setEnabledCalls[0].auditEvent == nil {
		t.Fatal("expected storage activation audit event to be passed to repository")
	}
	if repo.setEnabledCalls[0].auditEvent.Metadata["activation_target"] != "STORAGE" {
		t.Fatalf("expected storage activation metadata, got %#v", repo.setEnabledCalls[0].auditEvent.Metadata)
	}
}

func TestSetLLMUsageEnabledPersistsActivationAudit(t *testing.T) {
	t.Parallel()

	repo := &providerRepoStub{
		configByProvider: map[string]*models.UserProviderConfig{
			domain.ProviderOpenAI: {
				ID:                 14,
				UserID:             5,
				Family:             domain.FamilyLLM,
				ProviderType:       domain.ProviderOpenAI,
				EnabledForIndexing: false,
				EnabledForChat:     false,
			},
		},
	}
	useCase := &providerUseCase{repo: repo}

	if err := useCase.SetLLMUsageEnabled(context.Background(), 5, domain.ProviderOpenAI, domain.LLMUsageChat, true); err != nil {
		t.Fatalf("expected llm activation update to succeed, got %v", err)
	}
	if len(repo.setLLMUsageCalls) != 1 || repo.setLLMUsageCalls[0].auditEvent == nil {
		t.Fatal("expected llm activation audit event to be passed to repository")
	}
	if repo.setLLMUsageCalls[0].auditEvent.Metadata["activation_target"] != "chat" {
		t.Fatalf("expected chat activation metadata, got %#v", repo.setLLMUsageCalls[0].auditEvent.Metadata)
	}
}

func TestListModelsReturnsOllamaModels(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "http://127.0.0.1:11434/api/tags" {
					t.Fatalf("unexpected models endpoint %q", r.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":[{"name":"llama3.1:8b"},{"name":"qwen2.5:7b"}]}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	models, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"model":    "llama3.1:8b",
		"base_url": "http://127.0.0.1:11434/v1",
	})
	if err != nil {
		t.Fatalf("expected model discovery to succeed, got %v", err)
	}
	if len(models) != 2 || models[0] != "llama3.1:8b" || models[1] != "qwen2.5:7b" {
		t.Fatalf("unexpected models result: %#v", models)
	}
}

func TestListModelsReturnsOllamaModelsWithoutConfiguredModel(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "http://127.0.0.1:11434/api/tags" {
					t.Fatalf("unexpected models endpoint %q", r.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":[{"name":"llama3.1:8b"},{"name":"qwen2.5:7b"}]}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	models, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"base_url": "http://127.0.0.1:11434/v1",
	})
	if err != nil {
		t.Fatalf("expected model discovery without a configured model to succeed, got %v", err)
	}
	if len(models) != 2 || models[0] != "llama3.1:8b" || models[1] != "qwen2.5:7b" {
		t.Fatalf("unexpected models result: %#v", models)
	}
}

func TestListModelsRejectsNonLLMFamily(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	_, err := useCase.ListModels(context.Background(), 1, domain.FamilyStorage, domain.ProviderMinio, map[string]string{
		"endpoint":    "minio.example.com:9000",
		"access_key":  "access",
		"secret_key":  "secret",
		"bucket_name": "docs",
	})
	if err == nil {
		t.Fatal("expected non-llm family to be rejected")
	}
}

func TestListModelsReturnsCachedCatalogWithoutDiscoveryOverrides(t *testing.T) {
	t.Parallel()

	repo := &providerRepoStub{
		modelCatalog: map[string][]models.ProviderModelCatalog{
			"GLOBAL:OLLAMA": {
				{ModelID: "deepseek-v3.1:671b", DisplayName: "deepseek-v3.1:671b", IsActive: true},
				{ModelID: "qwen3-coder:480b", DisplayName: "qwen3-coder:480b", IsActive: true},
			},
		},
	}
	useCase := &providerUseCase{
		repo:              repo,
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	models, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOllama, map[string]string{})
	if err != nil {
		t.Fatalf("expected cached models to be returned, got %v", err)
	}
	if len(models) != 2 || models[0] != "deepseek-v3.1:671b" || models[1] != "qwen3-coder:480b" {
		t.Fatalf("unexpected cached models result: %#v", models)
	}
}

func TestListModelsPrefersRemoteLookupWhenDiscoveryOverridesExist(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "http://127.0.0.1:11434/api/tags" {
					t.Fatalf("unexpected models endpoint %q", r.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":[{"name":"llama3.2:latest"},{"name":"qwen2.5:14b"}]}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	repo := &providerRepoStub{
		modelCatalog: map[string][]models.ProviderModelCatalog{
			"GLOBAL:OLLAMA": {
				{ModelID: "deepseek-v3.1:671b", DisplayName: "deepseek-v3.1:671b", IsActive: true},
			},
		},
	}
	useCase := &providerUseCase{
		repo:              repo,
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	models, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"base_url": "http://127.0.0.1:11434/v1",
	})
	if err != nil {
		t.Fatalf("expected remote discovery to succeed, got %v", err)
	}
	if len(models) != 2 || models[0] != "llama3.2:latest" || models[1] != "qwen2.5:14b" {
		t.Fatalf("unexpected models result: %#v", models)
	}
}

func TestListModelsReportsUnsupportedProvider(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	_, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key": "secret",
	})
	if !errors.Is(err, domain.ErrProviderModelsUnsupported) {
		t.Fatalf("expected unsupported model discovery error, got %v", err)
	}
}

func TestListModelsTimesOutSlowDiscovery(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				<-r.Context().Done()
				return nil, r.Context().Err()
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	start := time.Now()
	_, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"model":    "llama3.1:8b",
		"base_url": "http://127.0.0.1:11434/v1",
	})
	if !errors.Is(err, domain.ErrProviderModelsTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatalf("expected discovery timeout to fail quickly, took %s", time.Since(start))
	}
}

func TestListModelsAllowsOllamaCloudWithoutAPIKey(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://ollama.com/api/tags" {
					t.Fatalf("unexpected models endpoint %q", r.URL.String())
				}
				if got := r.Header.Get("Authorization"); got != "" {
					t.Fatalf("expected no authorization header, got %q", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":[{"name":"deepseek-v3.1:671b"}]}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	models, err := useCase.ListModels(context.Background(), 1, domain.FamilyLLM, domain.ProviderOllama, map[string]string{
		"model":    "deepseek-v3.1:671b",
		"base_url": "https://ollama.com/api",
	})
	if err != nil {
		t.Fatalf("expected ollama cloud discovery without api key to succeed, got %v", err)
	}
	if len(models) != 1 || models[0] != "deepseek-v3.1:671b" {
		t.Fatalf("unexpected models result: %#v", models)
	}
}

func TestSyncModelCatalogsStoresGlobalOllamaCatalog(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://ollama.com/api/tags" {
					t.Fatalf("unexpected sync endpoint %q", r.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":[{"name":"deepseek-v3.1:671b"},{"name":"qwen3-coder:480b"}]}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	repo := &providerRepoStub{}
	useCase := &providerUseCase{
		repo:              repo,
		cfg:               &config.Config{Environment: "development"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	if err := useCase.SyncModelCatalogs(context.Background()); err != nil {
		t.Fatalf("expected sync to succeed, got %v", err)
	}

	items := repo.modelCatalog["GLOBAL:OLLAMA"]
	if len(items) != 2 {
		t.Fatalf("expected synced catalog entries, got %#v", items)
	}
	if items[0].ModelID != "deepseek-v3.1:671b" || items[1].ModelID != "qwen3-coder:480b" {
		t.Fatalf("unexpected synced catalog entries: %#v", items)
	}
}

func TestSaveCredentialsPersistsLLMWhenConnectionValidationSucceeds(t *testing.T) {
	originalFactory := provider.NewLLMHTTPClientForTests(nil)
	defer provider.NewLLMHTTPClientForTests(originalFactory)

	provider.NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost {
					t.Fatalf("unexpected method %s", r.Method)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer valid-key" {
					t.Fatalf("unexpected authorization header %q", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"OK"}}]}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	})

	repo := &providerRepoStub{}
	useCase := &providerUseCase{
		repo:              repo,
		cfg:               &config.Config{Environment: "development", MasterEncryptionKey: "0123456789abcdef0123456789abcdef"},
		endpointValidator: netsec.NewStorageEndpointValidator(true),
	}

	err := useCase.SaveCredentials(context.Background(), 7, domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "valid-key",
		"model":    "gpt-4o-mini",
		"base_url": "http://8.8.8.8/v1",
	}, domain.ProviderActivation{EnabledForChat: true})
	if err != nil {
		t.Fatalf("expected credentials to be saved, got %v", err)
	}
	if len(repo.savedConfigs) != 1 {
		t.Fatalf("expected one saved config, got %d", len(repo.savedConfigs))
	}
	if !repo.savedConfigs[0].EnabledForChat {
		t.Fatal("expected saved config to keep chat activation")
	}
}
