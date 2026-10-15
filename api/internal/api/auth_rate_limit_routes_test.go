package api

import (
	"regexp"
	"testing"
)

// Toda rota de /auth que recebe e-mail no corpo passa pelo limite por conta
// (inclusive /register, por enumeracao). Le o fonte porque registrar rotas exige Postgres.
func TestRotasDeAuthComEmailPassamPeloLimitePorConta(t *testing.T) {
	fonte := lerFonteDeRotas(t)

	for _, rota := range []string{"/register", "/login", "/forgot-password"} {
		padrao := regexp.MustCompile(`auth\.POST\("` + regexp.QuoteMeta(rota) + `",\s*emailLimit,`)
		if !padrao.MatchString(fonte) {
			t.Errorf("rota %q sem emailLimit: o limite por conta nao se aplica a ela", rota)
		}
	}
}
