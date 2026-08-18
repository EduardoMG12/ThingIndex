package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// buildUserRateLimitRouter monta uma rota protegida por limite por usuario.
// O userID vem por header apenas para o teste conseguir simular usuarios
// diferentes sem depender do AuthMiddleware real.
func buildUserRateLimitRouter(limit int, window time.Duration) *gin.Engine {
	router := gin.New()
	router.POST("/upload", func(c *gin.Context) {
		if raw := c.GetHeader("X-Test-User"); raw != "" {
			switch raw {
			case "1":
				c.Set("user_id", uint(1))
			case "2":
				c.Set("user_id", uint(2))
			}
		}
		c.Next()
	}, UserRateLimitMiddleware(limit, window), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func doRequest(router *gin.Engine, path string, userID string) int {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if userID != "" {
		req.Header.Set("X-Test-User", userID)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}

// O ponto central do item: um usuario esgotando a cota NAO pode derrubar outro.
// Com o limite por IP essa separacao nao existia -- o frontend chama a API
// server-side, entao o ClientIP e o mesmo para todo mundo e o balde era global.
func TestUserRateLimitIsolaUsuariosDiferentes(t *testing.T) {
	t.Parallel()

	router := buildUserRateLimitRouter(2, time.Minute)

	if code := doRequest(router, "/upload", "1"); code != http.StatusOK {
		t.Fatalf("1a requisicao do usuario 1: esperava 200, veio %d", code)
	}
	if code := doRequest(router, "/upload", "1"); code != http.StatusOK {
		t.Fatalf("2a requisicao do usuario 1: esperava 200, veio %d", code)
	}
	if code := doRequest(router, "/upload", "1"); code != http.StatusTooManyRequests {
		t.Fatalf("3a requisicao do usuario 1 deveria estourar a cota, veio %d", code)
	}

	// Usuario 2 nunca enviou nada. Se levar 429 aqui, o limite continua
	// global e o bug nao foi corrigido.
	if code := doRequest(router, "/upload", "2"); code != http.StatusOK {
		t.Fatalf("usuario 2 foi punido pela cota do usuario 1 (veio %d); o limite ainda e global", code)
	}
}

func TestUserRateLimitLiberaAposAJanela(t *testing.T) {
	t.Parallel()

	router := buildUserRateLimitRouter(1, 40*time.Millisecond)

	if code := doRequest(router, "/upload", "1"); code != http.StatusOK {
		t.Fatalf("1a requisicao: esperava 200, veio %d", code)
	}
	if code := doRequest(router, "/upload", "1"); code != http.StatusTooManyRequests {
		t.Fatalf("2a requisicao deveria estourar, veio %d", code)
	}

	time.Sleep(60 * time.Millisecond)

	if code := doRequest(router, "/upload", "1"); code != http.StatusOK {
		t.Fatalf("apos a janela a cota deveria zerar, veio %d", code)
	}
}

// Sem user_id no contexto (middleware fora de ordem), o pedido passa em vez de
// ser barrado. Rejeitar transformaria um erro de configuracao em
// indisponibilidade total da rota -- pior que a falta de limite.
func TestUserRateLimitDeixaPassarQuandoNaoHaUsuarioNoContexto(t *testing.T) {
	t.Parallel()

	router := buildUserRateLimitRouter(1, time.Minute)

	for i := 0; i < 5; i++ {
		if code := doRequest(router, "/upload", ""); code != http.StatusOK {
			t.Fatalf("requisicao %d sem user_id: esperava 200, veio %d", i+1, code)
		}
	}
}

// Baldes separados por rota: enviar documentos nao pode consumir a cota de
// conversar sobre eles.
func TestUserRateLimitSeparaBaldesPorRota(t *testing.T) {
	t.Parallel()

	router := gin.New()
	limitar := func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	}
	router.POST("/upload", limitar, UserRateLimitMiddleware(1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.POST("/chat", limitar, UserRateLimitMiddleware(1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	if code := doRequest(router, "/upload", "1"); code != http.StatusOK {
		t.Fatalf("upload: esperava 200, veio %d", code)
	}
	if code := doRequest(router, "/upload", "1"); code != http.StatusTooManyRequests {
		t.Fatalf("upload deveria estourar na 2a, veio %d", code)
	}

	if code := doRequest(router, "/chat", "1"); code != http.StatusOK {
		t.Fatalf("o chat foi bloqueado pela cota de upload (veio %d); os baldes estao compartilhados", code)
	}
}
