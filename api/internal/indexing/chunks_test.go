package indexing

import (
	"strings"
	"testing"

	"thing-index/internal/models"
)

// Paragrafos espalhados por paginas diferentes, cada chunk conferido contra a
// pagina do seu conteudo (em texto puro toda pagina e 1 e nao prova nada).

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

// Paragrafo e StartPage do no divergem de proposito: o paragrafo ganha.
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

// Sem pagina em lugar nenhum, o chunk fica nil, nao zero.
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

// O fallback (documento sem sumario) tambem carrega a pagina.
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

// As partes de um no grande demais apontam todas a mesma pagina de inicio.
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

// O chunk nao compartilha o ponteiro de pagina com o paragrafo que o gerou.
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

// Idem para o StartPage do no.
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
