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

// RateLimitMiddleware limita por IP + rota. Serve para as rotas ANONIMAS
// (/auth), onde nao existe usuario autenticado para chavear.
//
// Nao use em rota autenticada: o frontend chama a API server-side, entao o
// ClientIP e sempre o do container do Next. Todos os usuarios cairiam no mesmo
// balde e um unico usuario ativo derrubaria os demais. Para essas rotas use
// UserRateLimitMiddleware.
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

// UserRateLimitMiddleware limita por usuario autenticado, e nao por IP.
//
// Motivo: o frontend chama a API server-side (Next em container), entao o
// ClientIP e sempre o mesmo endereco para TODOS os usuarios. Um limite por IP
// numa rota autenticada e, na pratica, um limite global compartilhado -- o
// usuario A esgota a cota e o usuario B leva 429 sem ter feito nada.
//
// Precisa vir DEPOIS do AuthMiddleware na cadeia, que e quem popula user_id.
// Se o user_id nao estiver no contexto (middleware fora de ordem), o pedido
// passa sem limite em vez de ser barrado: rejeitar aqui transformaria um erro
// de configuracao em indisponibilidade total da rota.
//
// A rota entra na chave para que upload e chat tenham baldes separados. Sem
// isso, enviar documentos consumiria a cota de conversar sobre eles.
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

// EmailRateLimitMiddleware limita por e-mail informado no corpo JSON, alem do
// limite por IP de RateLimitMiddleware.
//
// Motivo (parecer A3): o limite por IP nao protege uma conta contra forca bruta
// distribuida -- cada IP de uma botnet tem a propria cota. Chavear pelo e-mail
// alvo limita as tentativas contra a MESMA conta, venham de onde vierem. Usado
// em /auth/register, /auth/login e /auth/forgot-password.
//
// O e-mail e normalizado como no caso de uso (trim + minusculas), senao
// "Alice@x.com" e "alice@x.com " teriam baldes separados para a mesma conta.
// Corpo sem e-mail (ou invalido) passa: o handler devolve 400 na validacao, e
// o limite por IP continua valendo. O corpo e restaurado para o handler.
//
// Assim como os demais limitadores, e um mapa em memoria por processo.
// limit ou window <= 0 desliga este limite (o por IP continua valendo); assim
// um Config montado a mao, sem os campos novos, nao bloqueia todo login.
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

// NormalizeEmail aplica a mesma normalizacao do caso de uso de autenticacao.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
