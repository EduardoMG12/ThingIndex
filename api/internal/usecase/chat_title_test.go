package usecase

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Cortar em bytes parte o code point quando o caractere na fronteira e
// multibyte. O teste so exige UTF-8 valido e caber na coluna.
func TestBuildConversationTitleNaoQuebraUTF8NaFronteira(t *testing.T) {
	t.Parallel()

	// "ã" ocupa 2 bytes; com 254 ASCII antes, o byte 255 cai no meio dele.
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
