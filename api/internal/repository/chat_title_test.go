package repository

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Mesmo defeito de buildConversationTitle, em outro lugar: cortar a string em
// bytes parte um code point UTF-8 no meio. Aqui o titulo derivado aparece
// direto na lista de conversas, entao o resultado e um caractere corrompido
// visivel na tela.
func TestDeriveConversationTitleNaoQuebraUTF8NaFronteira(t *testing.T) {
	t.Parallel()

	// "ç" ocupa 2 bytes. Com 79 caracteres ASCII antes, o byte 80 cai no meio
	// do code point.
	conteudo := strings.Repeat("a", 79) + "ç" + strings.Repeat("b", 30)

	resultado := deriveConversationTitle(conteudo)

	if !utf8.ValidString(resultado) {
		t.Fatalf("o titulo derivado nao e UTF-8 valido: %q", resultado)
	}

	semReticencias := strings.TrimSuffix(resultado, "...")
	if runes := utf8.RuneCountInString(semReticencias); runes > 80 {
		t.Fatalf("esperava no maximo 80 runes antes das reticencias, veio %d", runes)
	}
	if !strings.HasSuffix(semReticencias, "ç") {
		t.Fatalf("o caractere da fronteira deveria estar inteiro, veio %q", resultado)
	}
	if !strings.HasSuffix(resultado, "...") {
		t.Fatalf("titulo truncado deveria terminar em reticencias, veio %q", resultado)
	}
}

func TestDeriveConversationTitleMantemMensagemCurta(t *testing.T) {
	t.Parallel()

	if got := deriveConversationTitle("  Qual   é o prazo?  "); got != "Qual é o prazo?" {
		t.Fatalf("esperava espacos colapsados sem truncar, veio %q", got)
	}
}
