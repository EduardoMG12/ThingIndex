package indexing

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Titulo longo cheio de letras acentuadas: qualquer ponto de corte por byte
// tem chance de cair no meio de uma runa.
func TestClampIdentifierWithHashNuncaParteRunaAcentuada(t *testing.T) {
	titulo := strings.Repeat("ção-", 100)
	for limit := 10; limit <= 300; limit++ {
		saida := clampIdentifierWithHash(titulo, limit)
		if !utf8.ValidString(saida) {
			t.Fatalf("limit=%d: saida com UTF-8 invalido: %q", limit, saida)
		}
		if n := utf8.RuneCountInString(saida); n > limit {
			t.Fatalf("limit=%d: %d caracteres, coluna varchar(%d) recusaria", limit, n, limit)
		}
	}
}

func TestBuildOutlineSlugRespeitaLimiteEmCaracteres(t *testing.T) {
	slug := buildOutlineSlug(strings.Repeat("Disposições Gerais ", 40))
	if !utf8.ValidString(slug) {
		t.Fatalf("slug com UTF-8 invalido: %q", slug)
	}
	if utf8.RuneCountInString(slug) > maxOutlineSlugLength {
		t.Fatalf("slug com %d caracteres, limite %d", utf8.RuneCountInString(slug), maxOutlineSlugLength)
	}
}
