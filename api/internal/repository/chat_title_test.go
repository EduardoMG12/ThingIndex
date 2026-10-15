package repository

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTituloDeConversaNaoQuebraUTF8(t *testing.T) {
	// Corte por rune, nao por byte, para nao partir caractere acentuado.
	conteudo := strings.Repeat("ação", 40) // 160 runes, 240 bytes

	titulo := deriveConversationTitle(conteudo)

	if !utf8.ValidString(titulo) {
		t.Fatalf("titulo com UTF-8 invalido: %q", titulo)
	}
	if !strings.HasSuffix(titulo, "...") {
		t.Fatalf("titulo longo deveria terminar em reticencias: %q", titulo)
	}

	semReticencias := strings.TrimSuffix(titulo, "...")
	if runes := utf8.RuneCountInString(semReticencias); runes > 80 {
		t.Fatalf("titulo com %d runes: passou do limite de 80", runes)
	}
}

func TestTituloDeConversaPreservaTextoCurto(t *testing.T) {
	titulo := deriveConversationTitle("Qual o prazo de pagamento?")

	if titulo != "Qual o prazo de pagamento?" {
		t.Fatalf("texto curto foi alterado: %q", titulo)
	}
}

func TestTituloDeConversaVaziaRecebeCarimboDeData(t *testing.T) {
	for _, entrada := range []string{"", "   ", "\n\t  \n"} {
		titulo := deriveConversationTitle(entrada)

		if strings.TrimSpace(titulo) == "" {
			t.Fatalf("entrada %q gerou titulo vazio: a lista mostraria linha em branco", entrada)
		}
		if !strings.HasPrefix(titulo, "Chat ") {
			t.Fatalf("esperava carimbo de data, veio %q", titulo)
		}
	}
}

func TestTituloDeConversaColapsaQuebrasDeLinha(t *testing.T) {
	titulo := deriveConversationTitle("Primeira linha\n\n\tSegunda   linha\n")

	if titulo != "Primeira linha Segunda linha" {
		t.Fatalf("espacos nao foram colapsados: %q", titulo)
	}
	if strings.Contains(titulo, "\n") || strings.Contains(titulo, "\t") {
		t.Fatalf("titulo carrega quebra de linha: %q", titulo)
	}
}

func TestNormalizacaoDeEspacoEhIdempotente(t *testing.T) {
	entrada := "  texto   com    espacos  "

	uma := normalizeWhitespace(entrada)
	duas := normalizeWhitespace(uma)

	if uma != duas {
		t.Fatalf("normalizacao nao e idempotente: %q -> %q", uma, duas)
	}
	if uma != "texto com espacos" {
		t.Fatalf("resultado inesperado: %q", uma)
	}
}

func TestNormalizacaoDeEspacoPreservaAcentuacao(t *testing.T) {
	obtido := normalizeWhitespace("  ação   coordenação  ")

	if obtido != "ação coordenação" {
		t.Fatalf("acentuacao corrompida: %q", obtido)
	}
	if !utf8.ValidString(obtido) {
		t.Fatalf("UTF-8 invalido: %q", obtido)
	}
}
