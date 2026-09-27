package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Com TRUSTED_PROXIES apontando para a sub-rede do frontend (e um proxy
// reverso que sobrescreve X-Forwarded-For na frente dele), o limite por IP de
// /auth separa os clientes. Com a lista vazia, o padrao seguro, o cabecalho e
// ignorado e todos caem no balde de quem conecta: um cliente nao consegue
// forjar um IP novo a cada tentativa.
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
