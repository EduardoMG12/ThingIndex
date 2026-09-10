package usecase

import (
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// A fonte citada precisa dizer QUAL documento e, quando vem de um no do
// sumario, QUAL no -- e o que o workspace usa para ir ate o trecho. Antes so
// havia o ID "doc:page", que o frontend teria de parsear.
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
	if sources[0].DocumentID != 42 || sources[0].NodeID != "" || sources[0].Page != 3 {
		t.Fatalf("fonte de pagina errada: %+v", sources[0])
	}
	if sources[1].DocumentID != 43 || sources[1].NodeID != "n-12" || sources[1].Page != 7 {
		t.Fatalf("fonte de no errada: %+v", sources[1])
	}
}
