package repository

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// deriveConversationTitle produz o titulo exibido na lista de conversas. E
// logica pura, roda em toda criacao de conversa e nao tinha teste nenhum.

func TestTituloDeConversaNaoQuebraUTF8(t *testing.T) {
	// O corte e por RUNE e nao por byte de proposito: cortar em byte parte o
	// code point no meio e o titulo aparece com caractere corrompido na tela.
	// Acentuacao em portugues ocupa 2 bytes, entao 80 runes acentuadas passam
	// de 80 bytes e o defeito apareceria em qualquer titulo real.
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
	// A maioria das conversas comeca com uma pergunta curta, e reescrever isso
	// seria pior que nao fazer nada.
	titulo := deriveConversationTitle("Qual o prazo de pagamento?")

	if titulo != "Qual o prazo de pagamento?" {
		t.Fatalf("texto curto foi alterado: %q", titulo)
	}
}

func TestTituloDeConversaVaziaRecebeCarimboDeData(t *testing.T) {
	// Conversa sem conteudo acontece quando o usuario abre o chat e nao envia
	// nada. Titulo vazio deixaria uma linha em branco na lista, indistinguivel
	// de erro de carregamento.
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
	// Mensagem colada de PDF vem cheia de quebra de linha. Sem colapsar, o
	// titulo quebraria o layout da lista lateral.
	titulo := deriveConversationTitle("Primeira linha\n\n\tSegunda   linha\n")

	if titulo != "Primeira linha Segunda linha" {
		t.Fatalf("espacos nao foram colapsados: %q", titulo)
	}
	if strings.Contains(titulo, "\n") || strings.Contains(titulo, "\t") {
		t.Fatalf("titulo carrega quebra de linha: %q", titulo)
	}
}

func TestNormalizacaoDeEspacoEhIdempotente(t *testing.T) {
	// Aplicar duas vezes precisa dar o mesmo resultado: o valor passa por aqui
	// em caminhos diferentes e uma normalizacao que muda a cada passagem
	// produziria titulos diferentes para a mesma mensagem.
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
	// strings.Fields opera sobre runes; um erro aqui corromperia todo titulo em
	// portugues.
	obtido := normalizeWhitespace("  ação   coordenação  ")

	if obtido != "ação coordenação" {
		t.Fatalf("acentuacao corrompida: %q", obtido)
	}
	if !utf8.ValidString(obtido) {
		t.Fatalf("UTF-8 invalido: %q", obtido)
	}
}
