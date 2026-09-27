package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newEmailLimitedRouter(limit int, window time.Duration, bodies *[]string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		*bodies = append(*bodies, string(body))
		c.Status(http.StatusOK)
	}
	emailLimit := EmailRateLimitMiddleware(limit, window)
	router.POST("/auth/login", emailLimit, handler)
	router.POST("/auth/forgot-password", emailLimit, handler)
	return router
}

func postJSON(router *gin.Engine, path, body, remoteAddr string) int {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}

// Forca bruta distribuida: cada tentativa vem de um IP diferente, entao o
// limite por IP nunca dispara. O limite por e-mail precisa barrar a partir da
// tentativa limit+1 contra a mesma conta, mesmo com variacao de caixa e espaco.
func TestEmailRateLimitBarraMesmaContaDeIPsDiferentes(t *testing.T) {
	var bodies []string
	router := newEmailLimitedRouter(3, time.Minute, &bodies)

	variantes := []string{"alice@example.com", " Alice@Example.com", "ALICE@EXAMPLE.COM ", "alice@example.com"}
	codes := make([]int, 0, len(variantes))
	for i, email := range variantes {
		codes = append(codes, postJSON(router, "/auth/login", `{"email":"`+email+`","password":"x"}`, "198.51.100."+string(rune('1'+i))+":1234"))
	}

	want := []int{200, 200, 200, 429}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("tentativa %d: status %d, esperava %d (todas: %v)", i+1, codes[i], want[i], codes)
		}
	}

	// Outra conta tem balde proprio.
	if code := postJSON(router, "/auth/login", `{"email":"bob@example.com","password":"x"}`, "198.51.100.9:1"); code != 200 {
		t.Fatalf("outra conta foi barrada pelo balde da primeira: %d", code)
	}

	// Rotas diferentes tem baldes diferentes: esgotar o login nao impede o
	// pedido de recuperacao de senha.
	if code := postJSON(router, "/auth/forgot-password", `{"email":"alice@example.com"}`, "198.51.100.9:1"); code != 200 {
		t.Fatalf("forgot-password barrado pelo balde do login: %d", code)
	}
}

// O handler precisa receber o corpo intacto depois de o middleware le-lo.
func TestEmailRateLimitPreservaOCorpoParaOHandler(t *testing.T) {
	var bodies []string
	router := newEmailLimitedRouter(5, time.Minute, &bodies)

	body := `{"email":"carol@example.com","password":"segredo"}`
	if code := postJSON(router, "/auth/login", body, "203.0.113.1:1"); code != 200 {
		t.Fatalf("status %d", code)
	}
	if len(bodies) != 1 || bodies[0] != body {
		t.Fatalf("handler recebeu corpo alterado: %q", bodies)
	}
}

// Corpo sem e-mail passa: quem responde e a validacao do handler (400).
func TestEmailRateLimitIgnoraCorpoSemEmail(t *testing.T) {
	var bodies []string
	router := newEmailLimitedRouter(1, time.Minute, &bodies)

	for i := 0; i < 3; i++ {
		if code := postJSON(router, "/auth/login", `nao e json`, "203.0.113.1:1"); code != 200 {
			t.Fatalf("tentativa %d sem e-mail foi barrada: %d", i+1, code)
		}
	}
}

func TestEmailRateLimitDesligadoComLimiteZero(t *testing.T) {
	var bodies []string
	router := newEmailLimitedRouter(0, time.Minute, &bodies)

	for i := 0; i < 5; i++ {
		if code := postJSON(router, "/auth/login", `{"email":"dave@example.com"}`, "203.0.113.1:1"); code != 200 {
			t.Fatalf("limite zero deveria desligar, tentativa %d veio %d", i+1, code)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM \n"); got != "alice@example.com" {
		t.Fatalf("NormalizeEmail = %q", got)
	}
}
