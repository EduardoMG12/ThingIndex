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
	testConnectionErr      error
}

func (s *providerUseCaseAuditCaptureStub) SaveCredentials(ctx context.Context, _ uint, _ string, _ string, _ map[string]string, _ domain.ProviderActivation) error {
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastSaveAuditContext = auditCtx
	}
	return nil
}

func (s *providerUseCaseAuditCaptureStub) TestConnection(context.Context, uint, string, string, map[string]string) error {
	return s.testConnectionErr
}

func (s *providerUseCaseAuditCaptureStub) ListModels(context.Context, uint, string, string, map[string]string) ([]string, error) {
	return nil, nil
}

func (s *providerUseCaseAuditCaptureStub) GetDecryptedCredentials(context.Context, uint, string) (map[string]string, error) {
	return nil, nil
}

func (s *providerUseCaseAuditCaptureStub) ListUserProviders(context.Context, uint) ([]models.UserProviderConfig, error) {
	return nil, nil
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
