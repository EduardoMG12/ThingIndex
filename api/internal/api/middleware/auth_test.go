package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type stubAuthRepository struct {
	user *models.User
	err  error
}

func (s *stubAuthRepository) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	return s.user, s.err
}

func (s *stubAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	panic("unexpected call")
}

func (s *stubAuthRepository) CreateUser(ctx context.Context, user *models.User) error {
	panic("unexpected call")
}

func (s *stubAuthRepository) UpdateUser(ctx context.Context, user *models.User) error {
	panic("unexpected call")
}

func (s *stubAuthRepository) GetUserByResetTokenHash(ctx context.Context, tokenHash string) (*models.User, error) {
	panic("unexpected call")
}

func TestAuthMiddlewareRejectsTokenIssuedBeforeUserUpdate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret"}
	invalidAfter := time.Now().UTC()
	user := &models.User{
		ID:                  1,
		SessionInvalidAfter: &invalidAfter,
	}

	token := issueTestToken(t, cfg.JWTSecret, user.ID, invalidAfter.Add(-10*time.Second))

	router := gin.New()
	router.Use(AuthMiddleware(cfg, &stubAuthRepository{user: user}))
	router.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddlewareAcceptsFreshToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret"}
	invalidAfter := time.Now().UTC()
	user := &models.User{ID: 1, SessionInvalidAfter: &invalidAfter}

	token := issueTestToken(t, cfg.JWTSecret, user.ID, invalidAfter.Add(1*time.Second))

	router := gin.New()
	router.Use(AuthMiddleware(cfg, &stubAuthRepository{user: user}))
	router.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func issueTestToken(t *testing.T, secret string, userID uint, issuedAt time.Time) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
		},
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}
