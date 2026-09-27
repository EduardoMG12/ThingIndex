package api

import (
	"strings"
	"testing"
)

// Parecer B3: POST /documents/:id/paragraph-nodes, /table-sections e
// /metadata-pairs deixavam o dono do documento reescrever o indice (texto,
// paginas, tabelas) sem evento de auditoria, e o chat passava a citar
// "proveniencia" que nao corresponde ao PDF. O frontend nunca as usou; foram
// removidas. Este teste impede que voltem sem uma decisao explicita (papel
// administrativo + DocumentAuditEvent). Le o fonte pelo mesmo motivo de
// expensive_routes_test.go: registrar rotas exige Postgres.
func TestRotasQueReescreviamOIndiceSemAuditoriaNaoExistem(t *testing.T) {
	fonte := lerFonteDeRotas(t)

	for _, rota := range []string{"/paragraph-nodes", "/table-sections", "/metadata-pairs"} {
		if strings.Contains(fonte, rota) {
			t.Errorf("rota %q voltou a ser registrada: reescreve o indice sem trilha de auditoria", rota)
		}
	}
}
