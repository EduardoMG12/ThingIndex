package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Com TRUSTED_PROXIES apontando para a rede interna, o limite por IP de /auth
// passa a separar clientes pelo X-Forwarded-For que o Next repassa. Sem isso
// (lista vazia) todos caem no balde do container do frontend -- o defeito do
// parecer A3.
func TestRateLimitPorIPUsaXForwardedForDeProxyConfiavel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	build := func(trusted []string) *gin.Engine {
		router := gin.New()
		if err := router.SetTrustedProxies(trusted); err != nil {
			t.Fatalf("SetTrustedProxies: %v", err)
		}
		router.POST("/auth/login", RateLimitMiddleware(1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })
		return router
	}

	send := func(router *gin.Engine, clientIP string) int {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "172.29.0.5:40000" // container do frontend
		req.Header.Set("X-Forwarded-For", clientIP)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	confiavel := build([]string{"172.16.0.0/12", "192.168.0.0/16"})
	if a, b := send(confiavel, "203.0.113.10"), send(confiavel, "203.0.113.11"); a != 200 || b != 200 {
		t.Fatalf("clientes diferentes atras do proxy confiavel dividiram o balde: %d, %d", a, b)
	}
	if again := send(confiavel, "203.0.113.10"); again != http.StatusTooManyRequests {
		t.Fatalf("o mesmo cliente deveria esgotar o proprio balde, veio %d", again)
	}

	semConfianca := build(nil)
	if a, b := send(semConfianca, "203.0.113.10"), send(semConfianca, "203.0.113.11"); a != 200 || b != http.StatusTooManyRequests {
		t.Fatalf("controle: sem proxy confiavel todos caem no mesmo balde; veio %d, %d", a, b)
	}
}
