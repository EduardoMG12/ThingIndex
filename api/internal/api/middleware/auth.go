package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type authClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

const tokenIssuedAtSkew = 2 * time.Second

func AuthMiddleware(cfg *config.Config, authRepo domain.AuthRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetString("language")
		authHeader := c.GetHeader("Authorization")
		tokenString := ""

		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else if cookieToken, err := c.Cookie("token"); err == nil {
			tokenString = strings.TrimSpace(cookieToken)
		}

		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}

		claims := &authClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}

		if claims.UserID == 0 {
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}
		if authRepo == nil {
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}
		if claims.IssuedAt == nil {
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}

		user, err := authRepo.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
				c.Abort()
				return
			}
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}
		if user.SessionInvalidAfter != nil && claims.IssuedAt.Time.Add(tokenIssuedAtSkew).Before(*user.SessionInvalidAfter) {
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "error.unauthorized"), nil)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
