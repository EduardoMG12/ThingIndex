package indexing

import (
	"context"
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
