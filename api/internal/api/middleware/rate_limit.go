package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

type fixedWindowRateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
	limit   int
	window  time.Duration
}

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

func NewFixedWindowRateLimiter(limit int, window time.Duration) *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		entries: make(map[string]rateLimitEntry),
		limit:   limit,
		window:  window,
	}
}

func (l *fixedWindowRateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= l.window {
		l.entries[key] = rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		l.cleanup(now)
		return true
	}

	if entry.count >= l.limit {
		l.cleanup(now)
		return false
	}

	entry.count++
	l.entries[key] = entry
	l.cleanup(now)
	return true
}

func (l *fixedWindowRateLimiter) cleanup(now time.Time) {
	if len(l.entries) <= 1024 {
		return
	}

	cutoff := now.Add(-2 * l.window)
	for key, entry := range l.entries {
		if entry.windowStart.Before(cutoff) {
			delete(l.entries, key)
		}
	}
}

// RateLimitMiddleware limita por IP + rota, para as rotas anonimas (/auth).
// Em rota autenticada use UserRateLimitMiddleware: o frontend chama a API
// server-side e o ClientIP seria o mesmo para todos.
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewFixedWindowRateLimiter(limit, window)

	return func(c *gin.Context) {
		key := c.ClientIP() + ":" + c.Request.Method + ":" + c.FullPath()
		if !limiter.Allow(key, time.Now()) {
			lang := c.GetString("language")
			response.Error(c, http.StatusTooManyRequests, "error", i18n.T(lang, "error.rate_limited"), nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimitMiddleware limita por usuario + rota. Tem de vir depois do
// AuthMiddleware; sem user_id no contexto o pedido passa sem limite, para que
// um erro de ordem nao derrube a rota inteira.
func UserRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewFixedWindowRateLimiter(limit, window)

	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		if userID == 0 {
			c.Next()
			return
		}

		key := "user:" + strconv.FormatUint(uint64(userID), 10) + ":" + c.Request.Method + ":" + c.FullPath()
		if !limiter.Allow(key, time.Now()) {
			lang := c.GetString("language")
			response.Error(c, http.StatusTooManyRequests, "error", i18n.T(lang, "error.rate_limited"), nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// EmailRateLimitMiddleware limita por e-mail do corpo JSON, contra forca bruta
// distribuida em varios IPs. O e-mail e normalizado como no caso de uso; corpo
// sem e-mail passa (o handler responde 400). limit ou window <= 0 desliga.
func EmailRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 || window <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	limiter := NewFixedWindowRateLimiter(limit, window)

	return func(c *gin.Context) {
		email := readNormalizedEmail(c)
		if email == "" {
			c.Next()
			return
		}

		key := "email:" + email + ":" + c.Request.Method + ":" + c.FullPath()
		if !limiter.Allow(key, time.Now()) {
			lang := c.GetString("language")
			response.Error(c, http.StatusTooManyRequests, "error", i18n.T(lang, "error.rate_limited"), nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// readNormalizedEmail le o campo "email" do corpo JSON sem consumi-lo.
func readNormalizedEmail(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}

	body, err := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil || len(body) == 0 {
		return ""
	}

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	return NormalizeEmail(payload.Email)
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
