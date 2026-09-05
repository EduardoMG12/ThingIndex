package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"thing-index/internal/pkg/response"
)

// pingadorDeBanco e a dependencia minima do health check.
//
// Interface pequena em vez de *sql.DB: o handler so precisa saber se o banco
// responde, e depender do tipo concreto obrigaria o teste a subir Postgres para
// exercitar um if.
type pingadorDeBanco interface {
	PingContext(ctx context.Context) error
}

// healthCheckTimeout limita a checagem.
//
// Sem prazo, um banco que aceita conexao mas nao responde deixaria o health
// pendurado -- e um health que nao responde e tratado como falha por uns
// orquestradores e ignorado por outros, o pior dos dois mundos.
const healthCheckTimeout = 2 * time.Second

// healthHandler responde se a API consegue TRABALHAR, e nao apenas se o
// processo esta vivo.
//
// A versao anterior devolvia texto fixo com 200. Medido derrubando o Postgres
// com a API no ar: as consultas falhavam e o health seguia dizendo 200. Um
// orquestrador manteria a instancia no balanceador, com o painel verde
// enquanto o usuario recebe erro.
func healthHandler(pingador pingadorDeBanco) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handler registrado sem dependencia e erro de montagem: assumir
		// indisponibilidade e mais seguro do que entrar em panico e derrubar o
		// processo, ou do que responder OK sem ter verificado nada.
		if pingador == nil {
			response.Error(c, http.StatusServiceUnavailable, "error",
				"API indisponivel: dependencia de banco nao configurada",
				gin.H{"database": "not_configured"})
			return
		}

		ctx, cancelar := context.WithTimeout(c.Request.Context(), healthCheckTimeout)
		defer cancelar()

		if err := pingador.PingContext(ctx); err != nil {
			// O detalhe fica no log: a mensagem do driver carrega IP interno,
			// porta e nome do banco, e esta rota e publica.
			log.Printf("health check: banco indisponivel: %v", err)

			response.Error(c, http.StatusServiceUnavailable, "error",
				"API indisponivel: dependencia critica fora do ar",
				gin.H{"database": "unavailable"})
			return
		}

		response.Success(c, http.StatusOK, "success", "API Thing-Index no ar! 🚀",
			gin.H{"database": "ok"})
	}
}

// sqlDBParaHealth extrai o *sql.DB de dentro do GORM.
//
// Devolve nil quando a extracao falha, e o handler trata nil como
// indisponibilidade: preferivel a entrar em panico na montagem das rotas, que
// derrubaria a API inteira por causa do health check.
func sqlDBParaHealth(db *gorm.DB) pingadorDeBanco {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("health check: nao foi possivel obter o *sql.DB do gorm: %v", err)
		return nil
	}
	return sqlDB
}
