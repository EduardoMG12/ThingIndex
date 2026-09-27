package api

import (
	"regexp"
	"testing"
)

// Toda rota de /auth que recebe e-mail no corpo passa pelo limite por conta.
// O limite por IP sozinho nao segura tentativas distribuidas contra a mesma
// conta, e /register tambem aceita e-mail (enumeracao de contas por cadastro).
// Le o fonte pelo mesmo motivo de removed_routes_test.go: registrar rotas
// exige Postgres.
func TestRotasDeAuthComEmailPassamPeloLimitePorConta(t *testing.T) {
	fonte := lerFonteDeRotas(t)

	for _, rota := range []string{"/register", "/login", "/forgot-password"} {
		padrao := regexp.MustCompile(`auth\.POST\("` + regexp.QuoteMeta(rota) + `",\s*emailLimit,`)
		if !padrao.MatchString(fonte) {
			t.Errorf("rota %q sem emailLimit: o limite por conta nao se aplica a ela", rota)
		}
	}
}
