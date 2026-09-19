package indexing

import (
	"strings"
	"testing"

	"thing-index/internal/models"
)

// Estes testes existem por uma razao especifica e vale registrar: a TPC medida
// sobre o corpus C1 NAO consegue provar atribuicao correta de pagina, porque
// C1 e texto puro e `buildDocumentPages` joga todo bloco sem pagina na pagina 1
// (outline.go). Ali, "a pagina certa" e "a constante 1" sao indistinguiveis.
//
// A prova de que o chunk recebe a pagina CERTA, e nao apenas alguma pagina,
// tem que vir daqui: paragrafos espalhados por paginas diferentes, e cada
// chunk conferido contra a pagina do seu proprio conteudo.

func paragrafoNaPagina(seq int, pagina *int, conteudo string) models.ParagraphNode {
	return models.ParagraphNode{
		DocumentID:    1,
		SequenceOrder: seq,
		PageNumber:    pagina,
		Content:       conteudo,
	}
}

// noDoSumario monta um no cobrindo o intervalo [start, end] de paragrafos.
// O intervalo vive no Metadata porque e de la que `outlineParagraphRange` le.
func noDoSumario(seq int, titulo string, start, end int, startPage *int) models.DocumentOutlineNode {
	return models.DocumentOutlineNode{
		DocumentID:    1,
		SequenceOrder: seq,
		NodeID:        titulo,
		Level:         1,
		Title:         titulo,
		StartPage:     startPage,
		Metadata: map[string]any{
			"paragraph_start": start,
			"paragraph_end":   end,
		},
	}
}

// texto longo o bastante para passar de minChunkContentLength (80 caracteres).
func textoLongo(prefixo string) string {
	return prefixo + ": " + strings.Repeat("conteudo suficiente para formar um chunk valido. ", 3)
}

func TestChunkRecebeAPaginaDoSeuPrimeiroParagrafo(t *testing.T) {
	t.Parallel()

	p1, p2, p3 := 1, 2, 3
	paragrafos := []models.ParagraphNode{
		paragrafoNaPagina(0, &p1, textoLongo("abertura na pagina um")),
		paragrafoNaPagina(1, &p2, textoLongo("desenvolvimento na pagina dois")),
		paragrafoNaPagina(2, &p3, textoLongo("fechamento na pagina tres")),
	}
	nos := []models.DocumentOutlineNode{
		noDoSumario(0, "Secao A", 0, 0, &p1),
		noDoSumario(1, "Secao B", 1, 1, &p2),
		noDoSumario(2, "Secao C", 2, 2, &p3),
	}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)

	if len(chunks) != 3 {
		t.Fatalf("esperava 3 chunks, um por secao, veio %d", len(chunks))
	}
	esperado := []int{1, 2, 3}
	for i, chunk := range chunks {
		if chunk.PageNumber == nil {
			t.Fatalf("chunk %d (%q) ficou sem pagina", i, chunk.Title)
		}
		if *chunk.PageNumber != esperado[i] {
			t.Errorf("chunk %d (%q): pagina %d, esperava %d",
				i, chunk.Title, *chunk.PageNumber, esperado[i])
		}
	}
}

// Se a pagina viesse do StartPage do no em vez do paragrafo, este teste
// passaria errado: aqui os dois DIVERGEM de proposito. O paragrafo e a origem
// mais proxima da evidencia, entao ele ganha.
func TestPaginaDoParagrafoVenceOStartPageDoNo(t *testing.T) {
	t.Parallel()

	paginaDoParagrafo, paginaDoNo := 7, 99
	paragrafos := []models.ParagraphNode{
		paragrafoNaPagina(0, &paginaDoParagrafo, textoLongo("o texto mesmo esta na pagina sete")),
	}
	nos := []models.DocumentOutlineNode{noDoSumario(0, "Secao", 0, 0, &paginaDoNo)}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)

	if len(chunks) == 0 {
		t.Fatal("nenhum chunk gerado")
	}
	if chunks[0].PageNumber == nil || *chunks[0].PageNumber != paginaDoParagrafo {
		t.Fatalf("esperava a pagina do paragrafo (%d), veio %v",
			paginaDoParagrafo, chunks[0].PageNumber)
	}
}

// Paragrafo vazio nao conta como origem: a pagina tem que vir do primeiro
// paragrafo com conteudo de verdade.
func TestParagrafoVazioNaoDefineAPaginaDoChunk(t *testing.T) {
	t.Parallel()

	paginaVazia, paginaComTexto := 4, 5
	paragrafos := []models.ParagraphNode{
		paragrafoNaPagina(0, &paginaVazia, "   "),
		paragrafoNaPagina(1, &paginaComTexto, textoLongo("aqui esta o conteudo real")),
	}
	nos := []models.DocumentOutlineNode{noDoSumario(0, "Secao", 0, 1, nil)}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)

	if len(chunks) == 0 {
		t.Fatal("nenhum chunk gerado")
	}
	if chunks[0].PageNumber == nil || *chunks[0].PageNumber != paginaComTexto {
		t.Fatalf("esperava a pagina do paragrafo com texto (%d), veio %v",
			paginaComTexto, chunks[0].PageNumber)
	}
}

// Sem pagina em lugar nenhum, o chunk fica NULO e nao zero. Zero seria uma
// pagina inexistente apresentada como se fosse coordenada valida, que e pior
// que a ausencia: a metrica contaria como completo um item que nao da para
// conferir no documento.
func TestChunkSemOrigemDePaginaFicaNuloENaoZero(t *testing.T) {
	t.Parallel()

	paragrafos := []models.ParagraphNode{
		paragrafoNaPagina(0, nil, textoLongo("documento sem paginacao, como HTML")),
	}
	nos := []models.DocumentOutlineNode{noDoSumario(0, "Secao", 0, 0, nil)}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)

	if len(chunks) == 0 {
		t.Fatal("nenhum chunk gerado")
	}
	if chunks[0].PageNumber != nil {
		t.Fatalf("esperava pagina nula, veio %d", *chunks[0].PageNumber)
	}
}

// O caminho de fallback (documento sem sumario) tambem precisa carregar a
// pagina, senao um documento sem estrutura vira um indice inauditavel inteiro.
func TestChunkDeFallbackRecebeAPaginaDaPropriaPagina(t *testing.T) {
	t.Parallel()

	paginas := []models.DocumentPage{
		{DocumentID: 1, PageNumber: 1, Content: textoLongo("primeira pagina")},
		{DocumentID: 1, PageNumber: 2, Content: textoLongo("segunda pagina")},
	}

	chunks := newChunkBuilder(1).buildFromOutline(nil, nil, nil, paginas)

	if len(chunks) != 2 {
		t.Fatalf("esperava 2 chunks de fallback, veio %d", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.PageNumber == nil || *chunk.PageNumber != paginas[i].PageNumber {
			t.Errorf("chunk de fallback %d: pagina %v, esperava %d",
				i, chunk.PageNumber, paginas[i].PageNumber)
		}
	}
}

// Um no grande demais vira varias partes. Todas tem que apontar a mesma pagina
// de inicio: partes do mesmo trecho divergindo entre si seria proveniencia
// inconsistente dentro do proprio documento.
func TestPartesDeUmChunkGrandeCompartilhamAPagina(t *testing.T) {
	t.Parallel()

	pagina := 12
	longo := strings.Repeat("Frase completa com tamanho suficiente para forcar a divisao do chunk. ", 120)
	paragrafos := []models.ParagraphNode{paragrafoNaPagina(0, &pagina, longo)}
	nos := []models.DocumentOutlineNode{noDoSumario(0, "Secao longa", 0, 0, &pagina)}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)

	if len(chunks) < 2 {
		t.Fatalf("esperava divisao em varias partes, veio %d chunk(s)", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.PageNumber == nil || *chunk.PageNumber != pagina {
			t.Errorf("parte %d: pagina %v, esperava %d", i, chunk.PageNumber, pagina)
		}
	}
}

// O chunk nao pode compartilhar o ponteiro de pagina com o paragrafo que o
// gerou. Se compartilhasse, alterar o paragrafo depois mudaria a proveniencia
// ja publicada do chunk, sem ninguem ter escrito nele. Proveniencia que muda
// sozinha contradiz o principio II (mesma entrada, mesma saida).
func TestChunkNaoCompartilhaPonteiroDePaginaComOParagrafo(t *testing.T) {
	t.Parallel()

	pagina := 3
	paragrafos := []models.ParagraphNode{
		paragrafoNaPagina(0, &pagina, textoLongo("trecho na pagina tres")),
	}
	nos := []models.DocumentOutlineNode{noDoSumario(0, "Secao", 0, 0, nil)}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)
	if len(chunks) == 0 || chunks[0].PageNumber == nil {
		t.Fatal("esperava um chunk com pagina")
	}

	pagina = 99 // mexe na origem DEPOIS de o chunk ter sido construido

	if *chunks[0].PageNumber != 3 {
		t.Fatalf("a pagina do chunk mudou junto com a do paragrafo: %d", *chunks[0].PageNumber)
	}
}

// Mesmo raciocinio para o StartPage do no, que e a segunda opcao de origem.
func TestChunkNaoCompartilhaPonteiroDePaginaComONo(t *testing.T) {
	t.Parallel()

	paginaDoNo := 8
	paragrafos := []models.ParagraphNode{
		paragrafoNaPagina(0, nil, textoLongo("paragrafo sem pagina propria")),
	}
	nos := []models.DocumentOutlineNode{noDoSumario(0, "Secao", 0, 0, &paginaDoNo)}

	chunks := newChunkBuilder(1).buildFromOutline(nos, paragrafos, nil, nil)
	if len(chunks) == 0 || chunks[0].PageNumber == nil {
		t.Fatal("esperava um chunk com a pagina do no")
	}

	paginaDoNo = 99

	if *chunks[0].PageNumber != 8 {
		t.Fatalf("a pagina do chunk mudou junto com a do no: %d", *chunks[0].PageNumber)
	}
}
