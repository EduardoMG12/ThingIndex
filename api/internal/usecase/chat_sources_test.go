package usecase

import (
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// A fonte diz qual documento e qual no; o workspace usa isso para ir ao trecho.
func TestBuildChatSourcesCarriesDocumentAndNode(t *testing.T) {
	page := 7
	context := &domain.CollectionContext{
		Documents: []models.Document{{ID: 42, Title: "Edital 56"}, {ID: 43, Title: "PPC"}},
	}
	retrieval := &domain.CollectionRetrievalResult{
		PageHits: []domain.CollectionPageHit{{DocumentID: 42, PageNumber: 3, Snippet: "prazo de inscricao"}},
		NodeHits: []domain.CollectionNodeHit{{DocumentID: 43, NodeID: "n-12", Title: "Matriz", StartPage: &page}},
	}

	sources := buildChatSources(context, retrieval)
	if len(sources) != 2 {
		t.Fatalf("esperava 2 fontes, veio %d", len(sources))
	}
	if sources[0].DocumentID != 43 || sources[0].NodeID != "n-12" || sources[0].Page != 7 {
		t.Fatalf("fonte de no errada: %+v", sources[0])
	}
	if sources[1].DocumentID != 42 || sources[1].NodeID != "" || sources[1].Page != 3 {
		t.Fatalf("fonte de pagina errada: %+v", sources[1])
	}
}

// Secoes entram antes das paginas, no maximo ChatNodeSourceLimit, sem repetir
// a mesma pagina; o total fecha em ChatSourceLimit.
func TestBuildChatSourcesPutsSectionsFirstWithinLimits(t *testing.T) {
	if ChatSourceLimit != 6 || ChatNodeSourceLimit != 3 || ChatPageCharLimit != 4000 || ChatContextCharLimit != 24000 {
		t.Fatalf("constantes do contexto mudaram: revise o artigo e este teste")
	}
	p := func(n int) *int { return &n }
	context := &domain.CollectionContext{Documents: []models.Document{{ID: 1, Title: "Doc"}}}
	retrieval := &domain.CollectionRetrievalResult{
		NodeHits: []domain.CollectionNodeHit{
			{DocumentID: 1, NodeID: "a", StartPage: p(10)},
			{DocumentID: 1, NodeID: "sem-pagina"},
			{DocumentID: 1, NodeID: "b", StartPage: p(10)}, // mesma pagina de "a": ignorada
			{DocumentID: 1, NodeID: "c", StartPage: p(11)},
			{DocumentID: 1, NodeID: "d", StartPage: p(12)},
			{DocumentID: 1, NodeID: "e", StartPage: p(13)}, // quarto no: fica de fora
		},
		PageHits: []domain.CollectionPageHit{
			{DocumentID: 1, PageNumber: 11, Snippet: "repetida"}, // ja veio pela secao "c"
			{DocumentID: 1, PageNumber: 1}, {DocumentID: 1, PageNumber: 2}, {DocumentID: 1, PageNumber: 3}, {DocumentID: 1, PageNumber: 4},
		},
	}
	sources := buildChatSources(context, retrieval)
	got := make([]string, 0, len(sources))
	for _, s := range sources {
		got = append(got, s.ID)
	}
	want := "1:10:a 1:11:c 1:12:d 1:1 1:2 1:3"
	if strings.Join(got, " ") != want {
		t.Fatalf("ordem/limite das fontes: got %q want %q", strings.Join(got, " "), want)
	}
}

// O prompt leva o conteudo da pagina citada, com teto por pagina e teto total.
func TestBuildCollectionChatPromptSendsPageContentWithinLimits(t *testing.T) {
	longa := strings.Repeat("x", ChatPageCharLimit+50)
	pages := []models.DocumentPage{
		{DocumentID: 1, PageNumber: 9, Content: "Carga horária total do curso: 3.200 horas."},
		{DocumentID: 1, PageNumber: 10, Content: longa},
	}
	// Sete paginas de 4000 estouram o teto de 24000: as duas ultimas voltam ao trecho.
	for n := 11; n <= 16; n++ {
		pages = append(pages, models.DocumentPage{DocumentID: 1, PageNumber: n, Content: strings.Repeat("y", ChatPageCharLimit)})
	}
	context := &domain.CollectionContext{Collection: &models.Collection{ID: 1, Name: "SI"}, Documents: []models.Document{{ID: 1, Title: "PPC"}}, DocumentPages: pages}
	sources := []domain.ChatSource{{ID: "1:9", DocumentID: 1, Page: 9, DocName: "PPC", Snippet: "Carga horária total"}}
	for n := 10; n <= 16; n++ {
		sources = append(sources, domain.ChatSource{ID: "1:" + string(rune('0'+n-10)), DocumentID: 1, Page: n, DocName: "PPC", Snippet: "TRECHO-CURTO"})
	}

	system, user := buildCollectionChatPrompt(context, nil, domain.ChatRequest{Prompt: "Qual a carga horária?", Language: "pt-BR"}, sources)

	if !strings.Contains(user, "3.200 horas") {
		t.Fatalf("o conteudo da pagina nao entrou no prompt:\n%s", user)
	}
	if strings.Contains(user, longa) || !strings.Contains(user, strings.Repeat("x", ChatPageCharLimit)+"…") {
		t.Fatalf("a pagina longa nao foi cortada em %d runas", ChatPageCharLimit)
	}
	if got := strings.Count(user, "TRECHO-CURTO"); got != 2 {
		t.Fatalf("esperava 2 fontes de volta ao trecho por estourar o teto total, veio %d", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(user), "Responda em português do Brasil.") || !strings.HasSuffix(system, "Responda em português do Brasil.") {
		t.Fatalf("a instrucao de idioma nao esta no fim dos dois prompts")
	}
	if strings.Contains(user, "Relevant outline hits") {
		t.Fatalf("o bloco de outline hits deveria ter saido")
	}
}
