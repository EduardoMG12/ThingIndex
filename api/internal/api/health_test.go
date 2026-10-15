package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Responder HTTP prova que o processo esta vivo; so a checagem do banco prova que
// ele consegue trabalhar.

type pingadorFalso struct {
	err error
}

func (p *pingadorFalso) PingContext(context.Context) error {
	return p.err
}

func lerCorpo(t *testing.T, gravador *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var corpo map[string]any
	if err := json.Unmarshal(gravador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta nao e JSON valido: %v (corpo=%q)", err, gravador.Body.String())
	}
	return corpo
}

func TestHealthReportaFalhaQuandoOBancoEstaFora(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", healthHandler(&pingadorFalso{
		err: errors.New("dial tcp 172.29.0.2:5432: connect: connection refused"),
	}))

	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, httptest.NewRequest(http.MethodGet, "/health", nil))

	// 503 e o codigo que orquestradores entendem como "nao me mande trafego".
	if gravador.Code != http.StatusServiceUnavailable {
		t.Fatalf("com o banco fora o health deveria devolver 503, devolveu %d", gravador.Code)
	}

	corpo := lerCorpo(t, gravador)
	detalhe, _ := corpo["errors"].(map[string]any)
	if detalhe == nil {
		t.Fatalf("resposta sem detalhe da falha: o operador nao consegue ver qual dependencia caiu (%v)", corpo)
	}
	if detalhe["database"] == "ok" {
		t.Fatal("o health afirmou que o banco esta ok com ele fora do ar")
	}
}

func TestHealthReportaSucessoComOBancoNoAr(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", healthHandler(&pingadorFalso{}))

	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, httptest.NewRequest(http.MethodGet, "/health", nil))

	if gravador.Code != http.StatusOK {
		t.Fatalf("com tudo no ar o health deveria devolver 200, devolveu %d", gravador.Code)
	}

	corpo := lerCorpo(t, gravador)
	dados, _ := corpo["data"].(map[string]any)
	if dados == nil || dados["database"] != "ok" {
		t.Fatalf("health nao reportou o estado do banco: %v", corpo["data"])
	}
}

// A mensagem do driver tem IP e porta; o endpoint e publico.
func TestHealthNaoVazaDetalheDaInfraestrutura(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", healthHandler(&pingadorFalso{
		err: errors.New("dial tcp 172.29.0.2:5432: connect: connection refused"),
	}))

	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, httptest.NewRequest(http.MethodGet, "/health", nil))

	corpoTexto := gravador.Body.String()
	for _, segredo := range []string{"172.29.0.2", "5432", "connection refused"} {
		if contemTexto(corpoTexto, segredo) {
			t.Fatalf("o health vazou detalhe de infraestrutura (%q) numa rota publica: %s",
				segredo, corpoTexto)
		}
	}
}

// Registrado sem dependencia, o handler nao pode dar panico.
func TestHealthRespondeMesmoSemPingador(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", healthHandler(nil))

	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, httptest.NewRequest(http.MethodGet, "/health", nil))

	if gravador.Code != http.StatusServiceUnavailable {
		t.Fatalf("sem pingador o health deveria assumir indisponibilidade, devolveu %d", gravador.Code)
	}
}

func contemTexto(texto, sub string) bool {
	return len(sub) > 0 && len(texto) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(texto); i++ {
			if texto[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
