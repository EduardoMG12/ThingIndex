package api

import (
	"os"
	"strings"
	"testing"
)

// Garante que o limite por usuario continua montado nas rotas caras. Le o fonte
// porque o gin nao expoe a cadeia de middlewares por rota e montar o roteador
// exigiria Postgres.

func TestRotasCarasRegistramLimitePorUsuario(t *testing.T) {
	fonte := lerFonteDeRotas(t)

	casos := []struct {
		nome   string
		trecho string
		motivo string
	}{
		{
			nome:   "upload",
			trecho: `documents.POST("/upload",`,
			motivo: "storage + fila + LLM no worker",
		},
		{
			nome:   "chat stream",
			trecho: `chat.POST("/stream",`,
			motivo: "cada mensagem e chamada paga de LLM",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			inicio := strings.Index(fonte, caso.trecho)
			if inicio < 0 {
				t.Fatalf("rota %q nao encontrada em routes.go", caso.nome)
			}

			// Janela do registro da rota: do nome ate o fechamento da chamada.
			fim := strings.Index(fonte[inicio:], "\n\t\t\t)")
			if fim < 0 {
				t.Fatalf("registro da rota %q nao segue o formato multilinha esperado; "+
					"se virou uma linha so, o middleware pode ter sido removido junto", caso.nome)
			}

			registro := fonte[inicio : inicio+fim]
			if !strings.Contains(registro, "UserRateLimitMiddleware") {
				t.Fatalf("rota %q (%s) perdeu o limite por usuario:\n%s",
					caso.nome, caso.motivo, registro)
			}
		})
	}
}

// O frontend chama a API server-side: limite por IP viraria cota global.
func TestLimiteEhPorUsuarioNaoPorIP(t *testing.T) {
	fonte := lerFonteDeRotas(t)

	if strings.Contains(fonte, "IPRateLimitMiddleware(") &&
		strings.Contains(fonte, `documents.POST("/upload"`) {
		trecho := fonte[strings.Index(fonte, `documents.POST("/upload"`):]
		if fim := strings.Index(trecho, ")"); fim > 0 && strings.Contains(trecho[:fim], "IPRateLimit") {
			t.Fatal("upload passou a usar limite por IP: com o frontend chamando server-side " +
				"isso vira cota global compartilhada entre todos os usuarios")
		}
	}
}

func lerFonteDeRotas(t *testing.T) string {
	t.Helper()

	conteudo, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatalf("nao foi possivel ler routes.go: %v", err)
	}
	return string(conteudo)
}
