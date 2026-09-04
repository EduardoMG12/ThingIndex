package handlers

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"thing-index/internal/api/middleware"
	"thing-index/internal/domain"

	"github.com/gin-gonic/gin"
)

// Antes desta correcao, todo caminho 5xx do DocumentHandler respondia um 500
// generico e DESCARTAVA a causa: `respondInternalError(c, lang)` nao recebia o
// erro. Na pratica, um upload que falhava por storage fora do ar aparecia no
// log apenas como `500 | POST /api/v1/documents/upload`, sem uma linha
// explicando o motivo -- exatamente o cenario que este teste trava.
//
// Verificado RED -> GREEN: revertendo a assinatura de respondInternalError
// para descartar o erro, este teste falha com "o log do servidor ficou vazio".
func TestUploadDocumentLogaCausaDoErroInterno(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// A causa real que precisa aparecer no log do operador.
	causaReal := errors.New("minio: connection refused em minio:9000")
	useCase := &documentUseCaseAuditCaptureStub{
		uploadErr: errors.Join(domain.ErrUploadFailed, causaReal),
	}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.POST("/api/v1/documents/upload", handler.Upload)

	// Captura o log padrao do servidor durante a requisicao.
	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logBuffer)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	})

	body := "--x\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nArquivo\r\n" +
		"--x\r\nContent-Disposition: form-data; name=\"collection\"\r\n\r\nColecao\r\n" +
		"--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"arquivo.pdf\"\r\n" +
		"Content-Type: application/pdf\r\n\r\n%PDF-1.4\r\n--x--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, veio %d", rec.Code)
	}

	logged := logBuffer.String()
	if strings.TrimSpace(logged) == "" {
		t.Fatal("o log do servidor ficou vazio: a causa do 500 foi engolida e o operador fica cego")
	}
	if !strings.Contains(logged, causaReal.Error()) {
		t.Fatalf("o log precisa conter a causa real %q, veio: %q", causaReal.Error(), logged)
	}
	if !strings.Contains(logged, "/api/v1/documents/upload") {
		t.Fatalf("o log precisa identificar a rota que falhou, veio: %q", logged)
	}

	// O detalhe de infraestrutura NAO pode vazar para o cliente.
	if strings.Contains(rec.Body.String(), causaReal.Error()) {
		t.Fatalf("a causa interna vazou na resposta HTTP: %s", rec.Body.String())
	}
}
