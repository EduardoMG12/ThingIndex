package middleware

import (
	"net/http"
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
