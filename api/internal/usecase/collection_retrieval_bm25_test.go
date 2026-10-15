package usecase

import (
	"strings"
	"testing"

	"thing-index/internal/models"
)

// A pergunta carrega "em", "que", "de" e uma pagina longa contem todas;
// a linha curta da matriz so contem as raras. Com BM25 a matriz vence.
// Acento nao pode separar "periodo" de "período".
func TestBuildCollectionRetrievalPrefersRareTermsOverLongStopwordHeavyPages(t *testing.T) {
	t.Parallel()
	longa := strings.Repeat("em que o curso de sistemas de informacao esta a disciplina e seu pre requisito do projeto pedagogico banco ", 40)
	pages := []models.DocumentPage{
		{DocumentID: 260, PageNumber: 40, Content: longa},
		{DocumentID: 265, PageNumber: 3, Content: "3º Período: Banco de Dados I – 67 h – pré-requisito: Algoritmos e Lógica de Programação"},
	}
	// Paginas de enchimento para baixar o IDF das palavras comuns.
	for n := 1; n <= 12; n++ {
		pages = append(pages, models.DocumentPage{DocumentID: 260, PageNumber: 100 + n, Content: "Em que o curso de sistemas de informacao esta: a disciplina que o projeto pedagogico do curso exige como pre requisito para o seu estagio."})
	}
	result := buildCollectionRetrieval(
		"Em que periodo do curso esta a disciplina de Banco de Dados I e qual e o seu pre-requisito?",
		[]models.Document{{ID: 260, Title: "PPC"}, {ID: 265, Title: "Matriz"}},
		pages, nil, nil, nil,
	)
	if result == nil || len(result.PageHits) == 0 {
		t.Fatal("esperava hits de pagina")
	}
	if got := result.PageHits[0]; got.DocumentID != 265 || got.PageNumber != 3 {
		t.Fatalf("a matriz (265:3) deveria vencer a pagina longa: %#v", result.PageHits)
	}
	// O ranking de documentos soma as paginas, entao documento longo ainda pode
	// liderar; o chat usa os hits de pagina e secao.
	if result.Strategy != "collection-bm25" {
		t.Fatalf("estrategia: %q", result.Strategy)
	}
}
