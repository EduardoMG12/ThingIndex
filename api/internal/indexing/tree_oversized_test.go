package indexing

import (
	"context"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// LLM que devolve uma arvore "bonita" de um no so cobrindo 40 paginas: e o
// que aconteceu com o PPC de 209 paginas (7 nos de ate 130 paginas, com
// start_page na pagina do sumario). Cobertura por intervalo passa; a
// estrutura e chute.
type stubOversizedTreeLLM struct{}

func (stubOversizedTreeLLM) GenerateText(_ context.Context, _ string, userPrompt string) (string, error) {
	if strings.Contains(userPrompt, "expand") || strings.Contains(userPrompt, "Expand") {
		return `{"nodes":[]}`, nil
	}
	return `{"nodes":[{"node_id":"1","parent_node_id":"","title":"Documento inteiro","level":1,"start_page":1,"end_page":40,"summary":"tudo"}]}`, nil
}

func TestArvoreDaLLMComNoGrandeDemaisNaoSubstituiOSumario(t *testing.T) {
	t.Parallel()
	pages := make([]models.DocumentPage, 0, 40)
	for n := 1; n <= 40; n++ {
		pages = append(pages, models.DocumentPage{DocumentID: 1, PageNumber: n, Content: strings.Repeat("texto da pagina com conteudo suficiente ", 20)})
	}
	start := 1
	result := domain.IndexingResult{
		DocumentPages: pages,
		OutlineNodes:  []models.DocumentOutlineNode{{DocumentID: 1, NodeID: "det-1", Title: "1 Introducao", Level: 1, StartPage: &start, EndPage: &start}},
	}
	updated, err := NewLLMEnricher().buildTree(context.Background(), stubOversizedTreeLLM{}, &models.Document{ID: 1, Title: "Doc"}, result)
	if err == nil || !strings.Contains(err.Error(), "tree not grounded") {
		t.Fatalf("esperava recusa por no grande demais, veio err=%v", err)
	}
	if len(updated.OutlineNodes) != 1 || updated.OutlineNodes[0].NodeID != "det-1" {
		t.Fatalf("o sumario deterministico deveria ficar intacto: %+v", updated.OutlineNodes)
	}
}
