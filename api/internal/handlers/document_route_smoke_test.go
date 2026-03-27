//go:build smoke

package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"thing-index/internal/api/middleware"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type routeSmokeAuthRepoStub struct {
	user *models.User
	err  error
}

func (s *routeSmokeAuthRepoStub) GetUserByID(context.Context, uint) (*models.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func (s *routeSmokeAuthRepoStub) GetUserByEmail(context.Context, string) (*models.User, error) {
	panic("unexpected call")
}

func (s *routeSmokeAuthRepoStub) CreateUser(context.Context, *models.User) error {
	panic("unexpected call")
}

func (s *routeSmokeAuthRepoStub) UpdateUser(context.Context, *models.User) error {
	panic("unexpected call")
}

func (s *routeSmokeAuthRepoStub) GetUserByResetTokenHash(context.Context, string) (*models.User, error) {
	panic("unexpected call")
}

func issueDocumentRouteToken(t *testing.T, secret string, userID uint, issuedAt time.Time) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"iat":     issuedAt.Unix(),
		"exp":     issuedAt.Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}

func buildProtectedDocumentRouter(t *testing.T, useCase *documentUseCaseAuditCaptureStub) (*gin.Engine, *config.Config, *routeSmokeAuthRepoStub) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "route-smoke-secret"}
	authRepo := &routeSmokeAuthRepoStub{
		user: &models.User{
			ID: 77,
		},
	}

	handler := NewDocumentHandler(useCase, 10<<20)
	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(middleware.I18nMiddleware())

	documents := router.Group("/api/v1/documents")
	documents.Use(middleware.AuthMiddleware(cfg, authRepo))
	{
		documents.POST("/upload", handler.Upload)
		documents.GET("/:id/content", handler.GetDocumentContent)
	}

	return router, cfg, authRepo
}

func TestDocumentRouteSmokeRejectsUnauthenticatedContentRequest(t *testing.T) {
	t.Parallel()

	router, _, _ := buildProtectedDocumentRouter(t, &documentUseCaseAuditCaptureStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/7/content", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 response, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Não autorizado") {
		t.Fatalf("expected localized unauthorized message, got %q", rec.Body.String())
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request id header to be present")
	}
}

func TestDocumentRouteSmokeUploadsWithBearerToken(t *testing.T) {
	t.Parallel()

	useCase := &documentUseCaseAuditCaptureStub{}
	router, cfg, _ := buildProtectedDocumentRouter(t, useCase)

	token := issueDocumentRouteToken(t, cfg.JWTSecret, 77, time.Now().UTC())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nArquivo\r\n--x\r\nContent-Disposition: form-data; name=\"collection\"\r\n\r\nColecao\r\n--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"arquivo.pdf\"\r\nContent-Type: application/pdf\r\n\r\n%PDF-1.7\r\n--x--\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("X-Request-ID", "route-smoke-upload-001")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", rec.Code)
	}
	if !useCase.uploadCalled {
		t.Fatal("expected upload use case to be called")
	}
	if !strings.Contains(rec.Body.String(), "Document uploaded successfully") {
		t.Fatalf("expected english upload success message, got %q", rec.Body.String())
	}
	if rec.Header().Get("X-Request-ID") != "route-smoke-upload-001" {
		t.Fatalf("expected request id to round-trip, got %q", rec.Header().Get("X-Request-ID"))
	}
}

func TestDocumentRouteSmokeStreamsContentWithCookieToken(t *testing.T) {
	t.Parallel()

	useCase := &documentUseCaseAuditCaptureStub{
		documentContent: &domain.DocumentContent{
			Document: &models.Document{
				ID:       3,
				Title:    "Termo.pdf",
				MimeType: "application/pdf",
				Size:     11,
			},
			Reader: io.NopCloser(strings.NewReader("pdf-body")),
		},
	}
	router, cfg, _ := buildProtectedDocumentRouter(t, useCase)

	token := issueDocumentRouteToken(t, cfg.JWTSecret, 77, time.Now().UTC())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/3/content", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	req.Header.Set("Accept-Language", "en-US")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "inline") || !strings.Contains(got, "Termo.pdf") {
		t.Fatalf("expected inline content disposition, got %q", got)
	}
	if rec.Body.String() != "pdf-body" {
		t.Fatalf("unexpected streamed body: %q", rec.Body.String())
	}
}
