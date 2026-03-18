package usecase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/provider"
)

type providerRepoStub struct {
	savedConfigs []*models.UserProviderConfig
	modelCatalog map[string][]models.ProviderModelCatalog
	allConfigs   []models.UserProviderConfig
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (r *providerRepoStub) SaveConfig(_ context.Context, config *models.UserProviderConfig) error {
	r.savedConfigs = append(r.savedConfigs, config)
	return nil
}

func (r *providerRepoStub) GetConfig(context.Context, uint, string) (*models.UserProviderConfig, error) {
	return nil, nil
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

func (r *providerRepoStub) SetConfigEnabled(context.Context, uint, string, string, bool) error {
	return nil
}

func (r *providerRepoStub) SetLLMConfigUsage(context.Context, uint, string, string, bool) error {
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
