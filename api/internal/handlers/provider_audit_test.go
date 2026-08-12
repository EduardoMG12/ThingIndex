package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"thing-index/internal/api/middleware"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/gin-gonic/gin"
)

type providerUseCaseAuditCaptureStub struct {
	lastSaveAuditContext   domain.AuditContext
	lastToggleAuditContext domain.AuditContext
	lastAuditFilter        domain.ProviderAuditFilter
	testConnectionErr      error
	testConnectionReport   *domain.ProviderReadinessReport
}

func (s *providerUseCaseAuditCaptureStub) SaveCredentials(ctx context.Context, _ uint, _ string, _ string, _ map[string]string, _ domain.ProviderActivation) error {
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastSaveAuditContext = auditCtx
	}
	return nil
}

func (s *providerUseCaseAuditCaptureStub) TestConnection(context.Context, uint, string, string, map[string]string) (*domain.ProviderReadinessReport, error) {
	return s.testConnectionReport, s.testConnectionErr
}

func (s *providerUseCaseAuditCaptureStub) ListModels(context.Context, uint, string, string, map[string]string) ([]string, error) {
	return nil, nil
}

func (s *providerUseCaseAuditCaptureStub) ListProviderSupportMatrix(context.Context, string) ([]domain.ProviderSupportEntry, error) {
	return []domain.ProviderSupportEntry{
		{
			Family:       domain.FamilyStorage,
			ProviderType: domain.ProviderGCPStorage,
			DisplayName:  "Google Cloud Storage",
			SupportLevel: domain.ProviderSupportLevelReady,
			Transport:    domain.ProviderSupportTransportNativeGCS,
			Summary:      "Ready for the main runtime.",
			SuggestedSetup: map[string]string{
				"project_id":  "thingindex-dev",
				"bucket_name": "thingindex-docs",
			},
			Capabilities: domain.ProviderSupportCapabilities{
				SupportsCredentialValidation: true,
				SupportsConnectionTest:       true,
				SupportsIndexing:             true,
				SupportsOptionalLiveTests:    true,
			},
			Notes: []string{"Supports service-account JSON or emulator_host."},
		},
		{
			Family:       domain.FamilyLLM,
			ProviderType: domain.ProviderOpenAI,
			DisplayName:  "OpenAI",
			SupportLevel: domain.ProviderSupportLevelReady,
			Transport:    domain.ProviderSupportTransportNativeOpenAI,
			Summary:      "Ready for the main runtime.",
			Capabilities: domain.ProviderSupportCapabilities{
				SupportsCredentialValidation: true,
				SupportsConnectionTest:       true,
				SupportsModelDiscovery:       true,
				SupportsStreaming:            true,
				SupportsIndexing:             true,
				SupportsChat:                 true,
				SupportsOptionalLiveTests:    true,
			},
			Notes: []string{"Direct OpenAI path."},
		},
	}, nil
}

func (s *providerUseCaseAuditCaptureStub) GetDecryptedCredentials(context.Context, uint, string) (map[string]string, error) {
	return nil, nil
}

func (s *providerUseCaseAuditCaptureStub) ListUserProviders(context.Context, uint) ([]models.UserProviderConfig, error) {
	return nil, nil
}

func (s *providerUseCaseAuditCaptureStub) GetProviderRuntimeOverview(context.Context, uint) (*domain.ProviderRuntimeOverview, error) {
	return &domain.ProviderRuntimeOverview{
		Indexing: &domain.ProviderRuntimeAssignment{
			Family:           domain.FamilyLLM,
			ActivationTarget: domain.LLMUsageIndexing,
			ProviderType:     domain.ProviderOpenAI,
			PublicMetadata: map[string]string{
				"model": "gpt-4o-mini",
			},
		},
		Chat: &domain.ProviderRuntimeAssignment{
			Family:           domain.FamilyLLM,
			ActivationTarget: domain.LLMUsageChat,
			ProviderType:     domain.ProviderOllama,
			PublicMetadata: map[string]string{
				"model": "llama3.1:8b",
			},
		},
	}, nil
}

func (s *providerUseCaseAuditCaptureStub) ListProviderAuditEvents(context.Context, uint, domain.ProviderAuditFilter) ([]models.ProviderAuditEvent, error) {
	return []models.ProviderAuditEvent{
		{
			ID:           1,
			Family:       domain.FamilyLLM,
			ProviderType: domain.ProviderOpenAI,
			Action:       "PROVIDER_CONFIG_SAVE",
			Status:       "SUCCESS",
		},
	}, nil
}

func (s *providerUseCaseAuditCaptureStub) QueryProviderAuditEvents(_ context.Context, _ uint, filter domain.ProviderAuditFilter) (*domain.ProviderAuditQueryResult, error) {
	s.lastAuditFilter = filter
	if filter.Cursor == "invalid" {
		return nil, domain.ErrAuditCursorInvalid
	}
	return &domain.ProviderAuditQueryResult{
		Items: []models.ProviderAuditEvent{
			{
				ID:            1,
				Family:        domain.FamilyLLM,
				ProviderType:  domain.ProviderOpenAI,
				Action:        "PROVIDER_CONFIG_SAVE",
				Status:        "SUCCESS",
				CorrelationID: "req-http-provider-save-001",
			},
		},
		Summary: domain.ProviderAuditSummary{
			TotalEvents: 1,
			ActionBreakdown: []domain.AuditCountBucket{
				{Key: "PROVIDER_CONFIG_SAVE", Count: 1},
			},
			StatusBreakdown: []domain.AuditCountBucket{
				{Key: "SUCCESS", Count: 1},
			},
		},
		PageInfo: domain.AuditPageInfo{
			Limit:    1,
			Returned: 1,
		},
	}, nil
}

func (s *providerUseCaseAuditCaptureStub) SyncModelCatalogs(context.Context) error {
	return nil
}

func (s *providerUseCaseAuditCaptureStub) SetProviderEnabled(ctx context.Context, _ uint, _ string, _ string, _ bool) error {
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastToggleAuditContext = auditCtx
	}
	return nil
}

func (s *providerUseCaseAuditCaptureStub) SetLLMUsageEnabled(ctx context.Context, _ uint, _ string, _ string, _ bool) error {
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastToggleAuditContext = auditCtx
	}
	return nil
}

func TestSaveCredentialsEndpointAddsAuditContextFromRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.POST("/api/v1/user/providers", handler.SaveCredentials)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/providers", strings.NewReader(`{"family":"LLM","provider_type":"OPENAI","credentials":{"api_key":"secret","model":"gpt-4o-mini"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-http-provider-save-001")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if useCase.lastSaveAuditContext.CorrelationID != "req-http-provider-save-001" {
		t.Fatalf("expected request id to propagate, got %#v", useCase.lastSaveAuditContext)
	}
	if useCase.lastSaveAuditContext.Origin != "POST /api/v1/user/providers" {
		t.Fatalf("expected save origin to include route, got %#v", useCase.lastSaveAuditContext)
	}
}

func TestListSupportMatrixEndpointReturnsSupportPayload(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Next()
	})
	router.GET("/api/v1/providers/support", handler.ListSupportMatrix)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/support?family=LLM", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider_type":"OPENAI"`) {
		t.Fatalf("expected openai support entry, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"support_level":"READY"`) {
		t.Fatalf("expected ready support level, got %q", rec.Body.String())
	}
}

func TestListSupportMatrixEndpointRejectsUnsupportedFamilies(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Next()
	})
	router.GET("/api/v1/providers/support", handler.ListSupportMatrix)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/support?family=UNKNOWN", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListSupportMatrixEndpointAcceptsStorageFamily(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Next()
	})
	router.GET("/api/v1/providers/support", handler.ListSupportMatrix)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/support?family=STORAGE", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider_type":"GCP_STORAGE"`) {
		t.Fatalf("expected gcp storage support entry, got %q", rec.Body.String())
	}
}

func TestUpdateProviderStateEndpointAddsAuditContextFromRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(52))
		c.Next()
	})
	router.PATCH("/api/v1/user/providers/:providerType", handler.UpdateProviderState)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/providers/OPENAI", strings.NewReader(`{"activation_target":"CHAT","enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-http-provider-toggle-001")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if useCase.lastToggleAuditContext.CorrelationID != "req-http-provider-toggle-001" {
		t.Fatalf("expected request id to propagate, got %#v", useCase.lastToggleAuditContext)
	}
	if useCase.lastToggleAuditContext.Origin != "PATCH /api/v1/user/providers/:providerType" {
		t.Fatalf("expected toggle origin to include route, got %#v", useCase.lastToggleAuditContext)
	}
}

func TestTestConnectionEndpointReturnsGatewayTimeoutForProviderTimeout(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{testConnectionErr: domain.ErrProviderConnectionTimeout}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.POST("/api/v1/user/providers/test", handler.TestConnection)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/providers/test", strings.NewReader(`{"family":"LLM","provider_type":"OPENAI","credentials":{"api_key":"secret","model":"gpt-4o-mini"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 response, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "tempo limite") {
		t.Fatalf("expected timeout message, got %q", rec.Body.String())
	}
}

func TestTestConnectionEndpointReturnsReadinessPayloadOnSuccess(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{
		testConnectionReport: &domain.ProviderReadinessReport{
			Family:       domain.FamilyLLM,
			ProviderType: domain.ProviderOpenAI,
			Status:       domain.ProviderReadinessStatusReady,
			Checks: []domain.ProviderReadinessCheck{
				{Code: domain.ProviderReadinessCheckCredentials, Status: domain.ProviderReadinessCheckPassed},
				{Code: domain.ProviderReadinessCheckConnection, Status: domain.ProviderReadinessCheckPassed},
				{Code: domain.ProviderReadinessCheckModelDiscovery, Status: domain.ProviderReadinessCheckPassed},
			},
			DiscoveredModels: []string{"gpt-4o-mini", "gpt-4.1-mini"},
		},
	}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.POST("/api/v1/user/providers/test", handler.TestConnection)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/providers/test", strings.NewReader(`{"family":"LLM","provider_type":"OPENAI","credentials":{"api_key":"secret","model":"gpt-4o-mini"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"READY"`) {
		t.Fatalf("expected readiness status in response, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"discovered_models":["gpt-4o-mini","gpt-4.1-mini"]`) {
		t.Fatalf("expected discovered models in response, got %q", rec.Body.String())
	}
}

func TestTestConnectionEndpointReturnsBadGatewayForLoopbackOllamaEndpoint(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{testConnectionErr: domain.ErrProviderEndpointUnavailable}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.POST("/api/v1/user/providers/test", handler.TestConnection)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/providers/test", strings.NewReader(`{"family":"LLM","provider_type":"OLLAMA","credentials":{"model":"llama3.1:8b","base_url":"http://localhost:11434/v1"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 response, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "host.docker.internal") {
		t.Fatalf("expected localhost ollama hint, got %q", rec.Body.String())
	}
}

func TestTestConnectionEndpointReturnsBadRequestForInvalidModel(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{testConnectionErr: domain.ErrProviderModelInvalid}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.POST("/api/v1/user/providers/test", handler.TestConnection)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/providers/test", strings.NewReader(`{"family":"LLM","provider_type":"OPENAI","credentials":{"api_key":"secret","model":"missing-model"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "modelo") {
		t.Fatalf("expected invalid model message, got %q", rec.Body.String())
	}
}

func TestListProviderAuditEventsEndpointAcceptsFilters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.GET("/api/v1/user/providers/audit", handler.ListProviderAuditEvents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/providers/audit?family=llm&provider_type=openai&action=provider_config_save&status=success&actor_type=user&actor_id=52&correlation_id=req-provider-77&q=openai&created_from=2026-03-20T10:00:00Z&created_to=2026-03-20T12:00:00Z&limit=5", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PROVIDER_CONFIG_SAVE") {
		t.Fatalf("expected provider audit payload, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"page_info\"") {
		t.Fatalf("expected provider audit page info, got %q", rec.Body.String())
	}
	if useCase.lastAuditFilter.ProviderType != "openai" || useCase.lastAuditFilter.ActorID == nil || *useCase.lastAuditFilter.ActorID != 52 {
		t.Fatalf("expected provider audit filters to propagate, got %#v", useCase.lastAuditFilter)
	}
}

func TestListProviderAuditEventsEndpointRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.GET("/api/v1/user/providers/audit", handler.ListProviderAuditEvents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/providers/audit?cursor=invalid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "cursor") {
		t.Fatalf("expected invalid cursor message, got %q", rec.Body.String())
	}
}

func TestListProviderRuntimeOverviewEndpointReturnsActiveAssignments(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &providerUseCaseAuditCaptureStub{}
	handler := NewProviderHandler(useCase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(44))
		c.Next()
	})
	router.GET("/api/v1/user/providers/runtime", handler.ListRuntimeOverview)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/providers/runtime", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"activation_target":"INDEXING"`) {
		t.Fatalf("expected indexing runtime assignment, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider_type":"OLLAMA"`) {
		t.Fatalf("expected chat runtime provider payload, got %q", rec.Body.String())
	}
}
