package api

import (
	"strings"
	"testing"
)

// As rotas POST de paragraph-nodes, table-sections e metadata-pairs reescreviam o
// indice sem auditoria e foram removidas; isto impede que voltem. Le o fonte
// porque registrar rotas exige Postgres.
func TestRotasQueReescreviamOIndiceSemAuditoriaNaoExistem(t *testing.T) {
	fonte := lerFonteDeRotas(t)

	for _, rota := range []string{"/paragraph-nodes", "/table-sections", "/metadata-pairs"} {
		if strings.Contains(fonte, rota) {
			t.Errorf("rota %q voltou a ser registrada: reescreve o indice sem trilha de auditoria", rota)
		}
	}
}
