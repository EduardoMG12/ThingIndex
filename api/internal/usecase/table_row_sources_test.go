package usecase

import (
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func matrizDeTeste() []models.TableSection {
	page := 1
	return []models.TableSection{{
		ID: 9, DocumentID: 265, PageNumber: &page,
		Headers: []string{"Período", "Disciplina", "Categoria", "Hrs", "Pré-requisito"},
		Rows: [][]string{
			{"01", "SIS07101 - Algoritmos e Lógica de Programação", "Natureza - Obrigatória", "100", ""},
			{"03", "SIS07302 - Estrutura de Dados I", "Natureza - Obrigatória", "67", "SIS07206"},
			{"", "SIS07305 - Sistemas Operacionais", "Natureza - Obrigatória", "67", "SIS07201"},
		},
	}}
}

// A linha da matriz vence a pagina de prosa do PPC e vira hit de linha com
// pagina; a tabela tambem sustenta o ranking do documento.
func TestLinhaDeTabelaVenceAPaginaDeProsaEViraHitComPagina(t *testing.T) {
	t.Parallel()
	pages := []models.DocumentPage{
		{DocumentID: 260, PageNumber: 40, Content: strings.Repeat("a disciplina de estrutura de dados e o pre requisito do curso de sistemas ", 30)},
		{DocumentID: 265, PageNumber: 1, Content: "GRADE CURRICULAR SIS07302 Estrutura de Dados I 67 SIS07206"},
	}
	result := buildCollectionRetrieval(
		"Qual é o pré-requisito de Estrutura de Dados I?",
		[]models.Document{{ID: 260, Title: "PPC"}, {ID: 265, Title: "Matriz"}},
		pages, nil, nil, matrizDeTeste(),
	)
	if result == nil || len(result.TableRowHits) == 0 {
		t.Fatal("esperava hits de linha de tabela")
	}
	top := result.TableRowHits[0]
	if top.DocumentID != 265 || top.PageNumber != 1 || top.RowIndex != 1 || !strings.Contains(top.Text, "Pré-requisito: SIS07206") {
		t.Fatalf("a linha de Estrutura de Dados I deveria ser o primeiro hit: %+v", top)
	}
	if result.DocumentHits[0].DocumentID != 265 {
		t.Fatalf("a matriz deveria liderar o ranking de documentos: %+v", result.DocumentHits)
	}
}

func TestLinhaSemPaginaPontuaODocumentoMasNaoViraFonte(t *testing.T) {
	t.Parallel()
	tables := matrizDeTeste()
	tables[0].PageNumber = nil // HTML/planilha
	result := buildCollectionRetrieval("pré-requisito de Estrutura de Dados I",
		[]models.Document{{ID: 265, Title: "Matriz"}}, nil, nil, nil, tables)
	if result == nil || len(result.TableRowHits) != 0 {
		t.Fatalf("linha sem pagina nao pode ser fonte: %+v", result)
	}
	if len(result.DocumentHits) == 0 || result.DocumentHits[0].DocumentID != 265 {
		t.Fatal("a tabela ainda deve pontuar o documento")
	}
}

// A linha entra primeiro nas fontes, com pagina, e o prompt leva a linha (com
// o titulo da tabela), nao a pagina inteira.
func TestFonteDeLinhaVemPrimeiroEVaiAoPromptComoLinha(t *testing.T) {
	t.Parallel()
	page := 7
	context := &domain.CollectionContext{
		Collection: &models.Collection{Name: "SI"},
		Documents:  []models.Document{{ID: 265, Title: "Matriz"}, {ID: 260, Title: "PPC"}},
		DocumentPages: []models.DocumentPage{
			{DocumentID: 265, PageNumber: 1, Content: "PAGINA INTEIRA DA MATRIZ " + strings.Repeat("x ", 500)},
		},
	}
	retrieval := &domain.CollectionRetrievalResult{
		TableRowHits: []domain.CollectionTableRowHit{{DocumentID: 265, TableID: 9, RowIndex: 1, PageNumber: 1, Title: "Grade 07", Text: "Disciplina: SIS07302 - Estrutura de Dados I; Pré-requisito: SIS07206"}},
		NodeHits:     []domain.CollectionNodeHit{{DocumentID: 260, NodeID: "n-1", Title: "Matriz do PPC", StartPage: &page}},
		PageHits:     []domain.CollectionPageHit{{DocumentID: 265, PageNumber: 1, Snippet: "grade"}},
	}
	sources := buildChatSources(context, retrieval)
	if len(sources) != 3 {
		t.Fatalf("esperava linha, secao e pagina: %+v", sources)
	}
	row := sources[0]
	if row.ID != "265:1:table-9-row-1" || row.Page != 1 || row.DocumentID != 265 || !strings.Contains(row.Body, "Grade 07 | Disciplina: SIS07302") {
		t.Fatalf("fonte de linha errada: %+v", row)
	}
	if sources[1].NodeID != "n-1" || sources[2].Page != 1 || sources[2].Body != "" {
		t.Fatalf("ordem das fontes: %+v", sources)
	}
	_, userPrompt := buildCollectionChatPrompt(context, retrieval, domain.ChatRequest{Prompt: "pré-requisito?", Language: "pt-BR"}, sources)
	if !strings.Contains(userPrompt, "1. Matriz (page 1):\nGrade 07 | Disciplina: SIS07302 - Estrutura de Dados I; Pré-requisito: SIS07206") {
		t.Fatalf("a primeira fonte do prompt deveria ser a linha, nao a pagina:\n%s", userPrompt)
	}
	if !strings.Contains(userPrompt, "3. Matriz (page 1):\nPAGINA INTEIRA DA MATRIZ") {
		t.Fatalf("a pagina inteira ainda entra como terceira fonte:\n%s", userPrompt)
	}
}
