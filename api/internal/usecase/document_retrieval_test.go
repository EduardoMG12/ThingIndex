package usecase

import (
	"testing"

	"thing-index/internal/models"
)

func TestBuildTreeRetrievalPrioritizesMatchingNodesAndPages(t *testing.T) {
	t.Parallel()

	startPageOne := 1
	endPageOne := 2
	startPageThree := 3
	endPageThree := 3

	result := buildTreeRetrieval(
		"payment deadline",
		[]models.DocumentOutlineNode{
			{
				NodeID:        "node-001-intro",
				SequenceOrder: 0,
				Title:         "Introduction",
				Summary:       "Visao geral do documento.",
				StartPage:     &startPageOne,
				EndPage:       &endPageOne,
			},
			{
				NodeID:        "node-002-payment",
				SequenceOrder: 1,
				Title:         "Payment Terms",
				Summary:       "Prazos de pagamento e deadline contratual.",
				StartPage:     &startPageThree,
				EndPage:       &endPageThree,
			},
		},
		[]models.DocumentPage{
			{PageNumber: 1, Summary: "Resumo introdutorio.", Content: "Introducao e escopo."},
			{PageNumber: 3, Summary: "Pagamento em 15 dias.", Content: "The payment deadline is 15 days after invoice."},
		},
	)
	if result == nil {
		t.Fatal("expected retrieval result")
	}

	if len(result.NodeHits) == 0 || result.NodeHits[0].NodeID != "node-002-payment" {
		t.Fatalf("unexpected node hits: %#v", result.NodeHits)
	}
	if len(result.PageHits) == 0 || result.PageHits[0].PageNumber != 3 {
		t.Fatalf("unexpected page hits: %#v", result.PageHits)
	}
	if !result.UsedTree || result.Strategy != "tree-first" {
		t.Fatalf("unexpected retrieval strategy: %#v", result)
	}
}
