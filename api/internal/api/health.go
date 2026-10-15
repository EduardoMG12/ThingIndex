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

type pingadorDeBanco interface {
	PingContext(ctx context.Context) error
}

// Sem prazo, um banco que aceita conexao mas nao responde pendura o health.
const healthCheckTimeout = 2 * time.Second

// healthHandler pinga o banco: responder 200 fixo deixava a instancia no
// balanceador com o Postgres fora.
func healthHandler(pingador pingadorDeBanco) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pingador == nil {
			response.Error(c, http.StatusServiceUnavailable, "error",
				"API indisponivel: dependencia de banco nao configurada",
				gin.H{"database": "not_configured"})
			return
		}

		ctx, cancelar := context.WithTimeout(c.Request.Context(), healthCheckTimeout)
		defer cancelar()

		if err := pingador.PingContext(ctx); err != nil {
			// O detalhe fica no log: a mensagem do driver tem IP e porta, e a rota e publica.
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

// sqlDBParaHealth devolve nil se a extracao falhar; o handler trata nil como indisponivel.
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
