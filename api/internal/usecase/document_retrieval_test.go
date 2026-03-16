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
	if len(result.TraversalSteps) == 0 || result.TraversalSteps[0].NodeID != "node-002-payment" {
		t.Fatalf("unexpected traversal steps: %#v", result.TraversalSteps)
	}
	if !result.UsedTree || result.Strategy != "tree-iterative" {
		t.Fatalf("unexpected retrieval strategy: %#v", result)
	}
}

func BenchmarkBuildTreeRetrieval(b *testing.B) {
	startPageOne := 1
	endPageFive := 5
	nodes := []models.DocumentOutlineNode{
		{
			NodeID:        "node-root",
			SequenceOrder: 0,
			Title:         "Payment Terms",
			Summary:       "Prazo, invoice, cobranca e multa.",
			StartPage:     &startPageOne,
			EndPage:       &endPageFive,
			Metadata: map[string]any{
				"confidence_score": 88,
			},
		},
	}
	pages := []models.DocumentPage{
		{PageNumber: 1, Content: "Payment terms and invoice issuance."},
		{PageNumber: 2, Content: "Deadline and penalties for late payment."},
		{PageNumber: 3, Content: "Operational details."},
		{PageNumber: 4, Content: "Escalation and contacts."},
		{PageNumber: 5, Content: "Appendix."},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildTreeRetrieval("payment deadline invoice", nodes, pages)
	}
}

func BenchmarkBuildIndexEvaluation(b *testing.B) {
	metadata := []models.MetadataPair{
		{Key: "tree_coverage_ratio", Value: "0.91"},
		{Key: "tree_total_nodes", Value: "8"},
		{Key: "tree_low_confidence_nodes", Value: "1"},
		{Key: "tree_oversized_nodes", Value: "0"},
		{Key: "tree_max_depth", Value: "4"},
	}
	nodes := []models.DocumentOutlineNode{
		{NodeID: "root", Title: "Contrato"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildIndexEvaluation(metadata, nodes)
	}
}
