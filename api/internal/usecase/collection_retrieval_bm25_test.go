package usecase

import (
	"strings"
	"testing"

	"thing-index/internal/models"
)

// O caso real da colecao da faculdade: a pergunta carrega "em", "que", "de",
// e uma pagina longa do PPC contem todas as palavras da pergunta como
// substring, enquanto a linha curta da matriz que tem a resposta contem so as
// raras. Com contagem de substring a pagina longa vencia; com BM25 (IDF e
// normalizacao por comprimento) a matriz vence. Acento tambem nao pode
// separar "periodo" de "período".
func TestBuildCollectionRetrievalPrefersRareTermsOverLongStopwordHeavyPages(t *testing.T) {
	t.Parallel()
	longa := strings.Repeat("em que o curso de sistemas de informacao esta a disciplina e seu pre requisito do projeto pedagogico banco ", 40)
	pages := []models.DocumentPage{
		{DocumentID: 260, PageNumber: 40, Content: longa},
		{DocumentID: 265, PageNumber: 3, Content: "3º Período: Banco de Dados I – 67 h – pré-requisito: Algoritmos e Lógica de Programação"},
	}
	// Paginas de enchimento com as palavras comuns da pergunta: e o que da IDF
	// baixo a "curso", "disciplina", "requisito" e alto a "dados", "periodo".
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
	// O ranking de DOCUMENTOS soma os scores de todas as paginas (politica
	// anterior, mantida): um documento longo com muitas paginas fracas ainda
	// pode liderar. O chat usa os hits de pagina e de secao, nao o de documento.
	if result.Strategy != "collection-bm25" {
		t.Fatalf("estrategia: %q", result.Strategy)
	}
}
