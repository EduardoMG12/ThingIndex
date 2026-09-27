package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func routerComLimiteDeCorpo(limite int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JSONBodyLimitMiddleware(limite))
	router.POST("/json", func(c *gin.Context) {
		var corpo map[string]any
		if err := c.ShouldBindJSON(&corpo); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})
	router.POST("/multipart", func(c *gin.Context) {
		if _, err := c.FormFile("file"); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})
	return router
}

// Regressao P2-5: rotas JSON decodificavam corpo de qualquer tamanho.
func TestJSONBodyLimitRecusaCorpoJSONAcimaDoTeto(t *testing.T) {
	router := routerComLimiteDeCorpo(64)

	grande := `{"campo":"` + strings.Repeat("a", 200) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(grande))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("corpo acima do teto passou: status %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(`{"campo":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("corpo pequeno recusado: status %d", rec.Code)
	}
}

func TestJSONBodyLimitNaoSeAplicaAMultipart(t *testing.T) {
	router := routerComLimiteDeCorpo(64)

	var corpo bytes.Buffer
	escritor := multipart.NewWriter(&corpo)
	parte, err := escritor.CreateFormFile("file", "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = parte.Write(bytes.Repeat([]byte("x"), 4096))
	_ = escritor.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/multipart", &corpo)
	req.Header.Set("Content-Type", escritor.FormDataContentType())
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("multipart foi limitado pelo teto de JSON: status %d", rec.Code)
	}
}

func TestJSONBodyLimitBytesEUmMiB(t *testing.T) {
	if JSONBodyLimitBytes != 1<<20 {
		t.Fatalf("teto de corpo JSON mudou para %d; revisar docs e frontend", JSONBodyLimitBytes)
	}
}
