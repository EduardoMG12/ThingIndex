package indexing

import (
	"context"
	"errors"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type stubFailingLLM struct{}

func (s stubFailingLLM) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "", errors.New("llm unavailable")
}

func TestNormalizeConstructedTreeRepairsInvalidParentRanges(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{DocumentID: 1, PageNumber: 1, Content: "Intro"},
			{DocumentID: 1, PageNumber: 2, Content: "Payment terms"},
			{DocumentID: 1, PageNumber: 3, Content: "Appendix"},
		},
	}

	payload := treeConstructionResponse{
		Nodes: []struct {
			NodeID       string `json:"node_id"`
			ParentNodeID string `json:"parent_node_id"`
			Title        string `json:"title"`
			Level        int    `json:"level"`
			StartPage    int    `json:"start_page"`
			EndPage      int    `json:"end_page"`
			Summary      string `json:"summary"`
		}{
			{NodeID: "root", Title: "Contrato", Level: 1, StartPage: 1, EndPage: 2, Summary: "Resumo"},
			{NodeID: "child", ParentNodeID: "root", Title: "Anexo", Level: 2, StartPage: 3, EndPage: 3, Summary: "Fora do range"},
		},
	}

	nodes := normalizeConstructedTree(result, payload)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if _, ok := nodes[1].Metadata["parent_node_id"]; ok {
		t.Fatalf("expected invalid parent to be removed: %#v", nodes[1].Metadata)
	}
	if nodes[1].Level != 1 {
		t.Fatalf("expected repaired node level 1, got %d", nodes[1].Level)
	}
	if nodes[1].NodePath != nodes[1].NodeID {
		t.Fatalf("expected repaired node path to reset, got %q", nodes[1].NodePath)
	}
}

func TestNormalizeConstructedTreeKeepsValidHierarchy(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{DocumentID: 1, PageNumber: 1, Content: "Intro"},
			{DocumentID: 1, PageNumber: 2, Content: "Payment terms"},
		},
	}

	payload := treeConstructionResponse{
		Nodes: []struct {
			NodeID       string `json:"node_id"`
			ParentNodeID string `json:"parent_node_id"`
			Title        string `json:"title"`
			Level        int    `json:"level"`
			StartPage    int    `json:"start_page"`
			EndPage      int    `json:"end_page"`
			Summary      string `json:"summary"`
		}{
			{NodeID: "root", Title: "Contrato", Level: 1, StartPage: 1, EndPage: 2, Summary: "Resumo"},
			{NodeID: "child", ParentNodeID: "root", Title: "Pagamento", Level: 2, StartPage: 2, EndPage: 2, Summary: "Prazo"},
		},
	}

	nodes := normalizeConstructedTree(result, payload)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if got := nodes[1].Metadata["parent_node_id"]; got != "root" {
		t.Fatalf("unexpected parent metadata: %#v", got)
	}
	if nodes[1].NodePath != "root/child" {
		t.Fatalf("unexpected node path: %q", nodes[1].NodePath)
	}
	if nodes[1].Level != 2 {
		t.Fatalf("unexpected child level: %d", nodes[1].Level)
	}
}

func TestVerifyAndScoreTreeComputesCoverageAndConfidence(t *testing.T) {
	t.Parallel()

	startOne := 1
	endTwo := 2
	startTwo := 2
	endTwoChild := 2
	nodes := []models.DocumentOutlineNode{
		{
			NodeID:    "root",
			Level:     1,
			Title:     "Payment Terms",
			Summary:   "Prazo de pagamento e invoice.",
			StartPage: &startOne,
			EndPage:   &endTwo,
			Metadata:  map[string]any{},
		},
		{
			NodeID:    "child",
			Level:     2,
			Title:     "Invoice Deadline",
			Summary:   "Pagamento em 15 dias.",
			StartPage: &startTwo,
			EndPage:   &endTwoChild,
			Metadata: map[string]any{
				"parent_node_id": "root",
			},
		},
	}

	verified, stats := verifyAndScoreTree([]models.DocumentPage{
		{PageNumber: 1, Content: "Payment terms overview and obligations."},
		{PageNumber: 2, Content: "Invoice deadline is 15 days after issuance."},
	}, nodes, "llm")

	if !stats.VerificationPassed {
		t.Fatalf("expected verification to pass: %#v", stats)
	}
	if stats.CoverageRatio < 1 {
		t.Fatalf("expected full coverage, got %.2f", stats.CoverageRatio)
	}
	if verified[0].Metadata["confidence_band"] == "low" {
		t.Fatalf("expected non-low confidence: %#v", verified[0].Metadata)
	}
	if verified[1].Metadata["verification_status"] != "verified" {
		t.Fatalf("expected verified status: %#v", verified[1].Metadata)
	}
}

func TestVerifyAndScoreTreeRejectsLowCoverage(t *testing.T) {
	t.Parallel()

	startOne := 1
	endOne := 1
	nodes := []models.DocumentOutlineNode{
		{
			NodeID:    "root",
			Level:     1,
			Title:     "Introduction",
			Summary:   "Resumo",
			StartPage: &startOne,
			EndPage:   &endOne,
			Metadata:  map[string]any{},
		},
	}

	_, stats := verifyAndScoreTree([]models.DocumentPage{
		{PageNumber: 1, Content: "Introduction content."},
		{PageNumber: 2, Content: "Uncovered page."},
		{PageNumber: 3, Content: "Another uncovered page."},
	}, nodes, "llm")

	if stats.VerificationPassed {
		t.Fatalf("expected verification to fail on low coverage: %#v", stats)
	}
	if stats.FallbackReason != "low_page_coverage" {
		t.Fatalf("unexpected fallback reason: %q", stats.FallbackReason)
	}
}

func TestIsOversizedNodeByPageSpan(t *testing.T) {
	t.Parallel()

	startPage := 1
	endPage := 12
	oversized := isOversizedNode(models.DocumentOutlineNode{
		Title:     "Large node",
		StartPage: &startPage,
		EndPage:   &endPage,
	}, map[int]models.DocumentPage{})

	if !oversized {
		t.Fatal("expected oversized node by page span")
	}
}

func TestFixTreeWithRetriesReturnsErrorWhenLLMFails(t *testing.T) {
	t.Parallel()

	startPage := 1
	endPage := 2
	nodes := []models.DocumentOutlineNode{
		{
			NodeID:    "root",
			Title:     "Contrato",
			Level:     1,
			StartPage: &startPage,
			EndPage:   &endPage,
			Metadata:  map[string]any{},
		},
	}

	enricher := NewLLMEnricher()
	_, _, err := enricher.fixTreeWithRetries(
		context.Background(),
		stubFailingLLM{},
		&models.Document{Title: "Contrato"},
		domain.IndexingResult{
			DocumentPages: []models.DocumentPage{
				{DocumentID: 1, PageNumber: 1, Content: "pagina um"},
				{DocumentID: 1, PageNumber: 2, Content: "pagina dois"},
			},
		},
		nodes,
		treeVerificationStats{FallbackReason: "low_page_coverage"},
	)
	if err == nil {
		t.Fatal("expected retry loop to return error when llm fails")
	}
}
