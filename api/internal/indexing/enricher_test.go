package indexing

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestParseLLMJSONExtractsFencedPayload(t *testing.T) {
	t.Parallel()

	raw := "```json\n{\"document_summary\":\"Resumo\",\"keywords\":[\"contrato\"],\"document_purpose\":\"Registrar obrigacoes.\"}\n```"

	var payload documentSynopsisResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		t.Fatalf("parseLLMJSON returned error: %v", err)
	}

	if payload.DocumentSummary != "Resumo" {
		t.Fatalf("unexpected document summary: %q", payload.DocumentSummary)
	}
	if len(payload.Keywords) != 1 || payload.Keywords[0] != "contrato" {
		t.Fatalf("unexpected keywords: %#v", payload.Keywords)
	}
}

func TestParseLLMJSONExtractsJSONAmidWrapperText(t *testing.T) {
	t.Parallel()

	raw := "Aqui esta o resultado solicitado:\n{\"pages\":[{\"page_number\":1,\"summary\":\"Resumo da pagina.\"}]}\nUse apenas o objeto."

	var payload pageSummaryResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		t.Fatalf("parseLLMJSON returned error: %v", err)
	}

	if len(payload.Pages) != 1 {
		t.Fatalf("expected one page summary, got %d", len(payload.Pages))
	}
	if payload.Pages[0].PageNumber != 1 {
		t.Fatalf("unexpected page number: %d", payload.Pages[0].PageNumber)
	}
	if payload.Pages[0].Summary != "Resumo da pagina." {
		t.Fatalf("unexpected page summary: %q", payload.Pages[0].Summary)
	}
}

func TestFillMissingPageSummariesUsesContentFallback(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{PageNumber: 1, Content: "Contrato de manutencao predial com atendimento emergencial em ate 4 horas."},
			{PageNumber: 2, Content: "Pagina com resumo existente.", Summary: "Ja preenchido."},
		},
	}

	fillMissingPageSummaries(&result)

	if result.DocumentPages[0].Summary == "" {
		t.Fatal("expected fallback summary for page without llm summary")
	}
	if result.DocumentPages[1].Summary != "Ja preenchido." {
		t.Fatalf("expected existing summary to be preserved, got %q", result.DocumentPages[1].Summary)
	}
}

func TestBuildFallbackDocumentSynopsisUsesIndexedContent(t *testing.T) {
	t.Parallel()

	synopsis := buildFallbackDocumentSynopsis(&models.Document{Title: "contrato-servicos.html"}, domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{PageNumber: 1, Content: "Contrato de prestacao de servicos com manutencao preventiva mensal e atendimento emergencial em ate 4 horas."},
		},
		OutlineNodes: []models.DocumentOutlineNode{
			{Title: "Escopo"},
		},
	})

	if synopsis.DocumentSummary == "" {
		t.Fatal("expected fallback document summary")
	}
	if synopsis.DocumentPurpose == "" {
		t.Fatal("expected fallback document purpose")
	}
	if len(synopsis.Keywords) == 0 {
		t.Fatal("expected fallback keywords")
	}
}

func TestLLMEnricherFallsBackToPageSegmentsWhenTreeBuildFails(t *testing.T) {
	t.Parallel()

	startPage := 1
	endPage := 3
	result, err := NewLLMEnricher().Enrich(
		context.Background(),
		stubFailingLLM{},
		&models.Document{ID: 1, Title: "contract.pdf"},
		domain.IndexingResult{
			DocumentPages: []models.DocumentPage{
				{DocumentID: 1, PageNumber: 1, Content: "First page explains the contract overview."},
				{DocumentID: 1, PageNumber: 2, Content: "Second page adds operational details."},
				{DocumentID: 1, PageNumber: 3, Content: "Third page closes with appendix notes."},
			},
			OutlineNodes: []models.DocumentOutlineNode{
				{
					DocumentID:    1,
					SequenceOrder: 0,
					NodeID:        "root",
					Level:         1,
					Title:         "Contract",
					Summary:       "General overview",
					StartPage:     &startPage,
					EndPage:       &endPage,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Enrich() returned error: %v", err)
	}

	if len(result.OutlineNodes) != 3 {
		t.Fatalf("expected fallback outline nodes per page, got %#v", result.OutlineNodes)
	}
	if result.OutlineNodes[0].Title != "Page 1" || result.OutlineNodes[1].Title != "Page 2" {
		t.Fatalf("unexpected fallback titles: %#v", result.OutlineNodes)
	}
	if got := lastMetadataValue(result.MetadataPairs, "tree_builder"); got != conservativeTreeFallbackBuilder {
		t.Fatalf("expected fallback builder metadata, got %q", got)
	}
	if got := lastMetadataValue(result.MetadataPairs, "tree_fallback_reason"); got != "shallow_tree" {
		t.Fatalf("expected fallback reason shallow_tree, got %q", got)
	}
	if got := lastMetadataValue(result.MetadataPairs, "tree_fallback_applied"); got != "true" {
		t.Fatalf("expected fallback applied metadata, got %q", got)
	}
}

func lastMetadataValue(pairs []models.MetadataPair, key string) string {
	value := ""
	for _, pair := range pairs {
		if pair.Key == key {
			value = pair.Value
		}
	}
	return value
}

// stubBatchLLM responde resumos para todas as paginas e secoes pedidas no
// prompt, e conta as chamadas: prova que o enriquecimento cobre o documento
// inteiro em lotes, nao so as 8 primeiras paginas e 12 primeiras secoes.
type stubBatchLLM struct {
	calls *int
}

func (s stubBatchLLM) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	*s.calls++
	if strings.Contains(userPrompt, "Pages:\n") {
		var pages []string
		for _, line := range strings.Split(userPrompt, "\n") {
			if n, ok := strings.CutPrefix(line, "- page_number="); ok {
				num, _, _ := strings.Cut(n, ",")
				pages = append(pages, `{"page_number":`+num+`,"summary":"Resumo da pagina `+num+` com os prazos e nomes citados nela.","topics":["prazo","matricula"]}`)
			}
		}
		return `{"pages":[` + strings.Join(pages, ",") + `]}`, nil
	}
	if strings.Contains(userPrompt, "Sections:\n") {
		var nodes []string
		for _, line := range strings.Split(userPrompt, "\n") {
			if n, ok := strings.CutPrefix(line, "- sequence_order="); ok {
				num, _, _ := strings.Cut(n, ",")
				nodes = append(nodes, `{"sequence_order":`+num+`,"summary":"Resumo da secao `+num+`."}`)
			}
		}
		return `{"nodes":[` + strings.Join(nodes, ",") + `]}`, nil
	}
	return "", errors.New("prompt nao suportado pelo stub")
}

func TestEnriquecimentoResumeTodasAsPaginasESecoesEmLotes(t *testing.T) {
	t.Parallel()

	var pages []models.DocumentPage
	var nodes []models.DocumentOutlineNode
	for i := 1; i <= 20; i++ {
		content := strings.Repeat("Texto da pagina com prazos, nomes e regras suficientes para resumir. ", 4)
		if i == 20 {
			content = "curta" // abaixo do minimo: fica no fallback
		}
		pages = append(pages, models.DocumentPage{DocumentID: 1, PageNumber: i, Content: content})
	}
	for i := 0; i < 15; i++ {
		start := i + 1
		nodes = append(nodes, models.DocumentOutlineNode{DocumentID: 1, SequenceOrder: i, NodeID: "n" + strconv.Itoa(i), Level: 1, Title: "Secao " + strconv.Itoa(i), StartPage: &start, EndPage: &start})
	}
	calls := 0
	result, err := NewLLMEnricher().Enrich(context.Background(), stubBatchLLM{calls: &calls}, &models.Document{ID: 1, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages, OutlineNodes: nodes})
	if err != nil {
		t.Fatalf("Enrich() erro: %v", err)
	}

	// 19 paginas com texto em lotes de 6 = 4 chamadas; 15 secoes em lotes de 12 = 2.
	if got := lastMetadataValue(result.MetadataPairs, "enrichment_pages_summarized"); got != "19" {
		t.Fatalf("paginas resumidas = %q, esperava 19", got)
	}
	if got := lastMetadataValue(result.MetadataPairs, "enrichment_sections_summarized"); got != "15" {
		t.Fatalf("secoes resumidas = %q, esperava 15", got)
	}
	for _, page := range result.DocumentPages {
		if page.PageNumber == 20 {
			if strings.HasPrefix(page.Summary, "Resumo da pagina") {
				t.Fatalf("pagina curta nao devia ir a LLM: %q", page.Summary)
			}
			continue
		}
		if !strings.HasPrefix(page.Summary, "Resumo da pagina "+strconv.Itoa(page.PageNumber)) {
			t.Fatalf("pagina %d sem resumo da LLM: %q", page.PageNumber, page.Summary)
		}
		if got, _ := page.Metadata["topics"].(string); got != "prazo, matricula" {
			t.Fatalf("pagina %d sem topicos: %#v", page.PageNumber, page.Metadata)
		}
	}
	for _, node := range result.OutlineNodes {
		if node.Summary != "Resumo da secao "+strconv.Itoa(node.SequenceOrder)+"." {
			t.Fatalf("secao %d sem resumo da LLM: %q", node.SequenceOrder, node.Summary)
		}
	}
	if calls > 12 {
		t.Fatalf("chamadas = %d, esperava poucas (sinopse, sinais, arvore e 6 lotes)", calls)
	}
}
