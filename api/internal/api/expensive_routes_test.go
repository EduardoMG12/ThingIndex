package api

import (
	"os"
	"strings"
	"testing"
)

// As rotas mais caras do sistema precisam de limite POR USUARIO. O middleware
// tem teste proprio, mas nada garantia que ele estivesse efetivamente montado
// nas rotas que importam -- e limite que existe e nao e aplicado da a mesma
// falsa seguranca de um teste que passa sem detectar nada.
//
// A sonda contra a API viva mostrou que o limite FUNCIONA (30 uploads passam,
// do 31 em diante vem 429). Este teste evita que uma refatoracao futura remova
// o middleware da cadeia sem ninguem perceber, o que so apareceria em producao
// como conta inesperada de provedor.
//
// Verifica o codigo-fonte porque o gin nao expoe a cadeia de middlewares por
// rota: Routes() devolve apenas o nome do handler final. Ler o registro da rota
// e a unica forma de afirmar que o limite esta ali sem subir Postgres.

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

// TestLimiteEhPorUsuarioNaoPorIP protege a decisao registrada no proprio
// codigo: o frontend chama a API server-side, entao o ClientIP e sempre o do
// container do Next. Um limite por IP viraria cota GLOBAL compartilhada entre
// todos os usuarios -- e o primeiro a enviar documentos bloquearia os demais.
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

// lerFonteDeRotas devolve o conteudo de routes.go.
//
// Ler o fonte e deliberado: o gin nao expoe a cadeia de middlewares por rota, e
// montar o roteador exigiria Postgres. O teste vale como guarda de regressao --
// a prova de que o limite FUNCIONA vem da sonda contra a API viva.
func lerFonteDeRotas(t *testing.T) string {
	t.Helper()

	conteudo, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatalf("nao foi possivel ler routes.go: %v", err)
	}
	return string(conteudo)
}
