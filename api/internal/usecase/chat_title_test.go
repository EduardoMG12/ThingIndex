package usecase

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// O corte de string em bytes (`s[:255]`) parte um code point UTF-8 no meio
// quando o caractere na fronteira e multibyte. Em portugues isso e o caso
// comum, nao a excecao: qualquer acento na posicao errada corrompe o titulo.
//
// O teste ancora no que importa de fato -- a string resultante e UTF-8 valida e
// cabe na coluna -- em vez de fixar um recorte especifico.
func TestBuildConversationTitleNaoQuebraUTF8NaFronteira(t *testing.T) {
	t.Parallel()

	// "ã" ocupa 2 bytes. Com 254 caracteres ASCII antes dele, o byte 255 cai
	// exatamente no meio do code point -- a fronteira que quebra.
	titulo := strings.Repeat("a", 254) + "ã" + strings.Repeat("b", 50)

	resultado := buildConversationTitle(titulo)

	if !utf8.ValidString(resultado) {
		t.Fatalf("o titulo cortado nao e UTF-8 valido: %q", resultado)
	}
	if runes := utf8.RuneCountInString(resultado); runes > 255 {
		t.Fatalf("esperava no maximo 255 runes, veio %d", runes)
	}
	if !strings.HasSuffix(resultado, "ã") {
		t.Fatalf("o caractere da fronteira deveria estar inteiro no fim, veio %q", resultado[len(resultado)-8:])
	}
}

func TestBuildConversationTitlePreservaTituloCurto(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"Relatório de análise":     "Relatório de análise",
		"  espaços   colapsados  ": "espaços colapsados",
	}

	for entrada, esperado := range casos {
		if got := buildConversationTitle(entrada); got != esperado {
			t.Errorf("buildConversationTitle(%q) = %q, esperava %q", entrada, got, esperado)
		}
	}
}

func TestBuildConversationTitleGeraFallbackQuandoVazio(t *testing.T) {
	t.Parallel()

	if got := buildConversationTitle("   "); !strings.HasPrefix(got, "Chat ") {
		t.Fatalf("esperava fallback com prefixo 'Chat ', veio %q", got)
	}
}
