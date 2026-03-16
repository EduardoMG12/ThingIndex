package usecase

import (
	"testing"
	"time"

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

func TestBuildIndexMonitoringBuildsTelemetrySnapshot(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.March, 16, 11, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(2 * time.Second)
	doc := &models.Document{
		Status:          "INDEXED",
		ProcessingStage: "COMPLETED",
		StatusDetail:    "Indexacao concluida e pronta para consulta.",
		IndexStartedAt:  &startedAt,
		IndexedAt:       &completedAt,
	}
	metadata := []models.MetadataPair{
		{Key: "enrichment_status", Value: "applied"},
		{Key: "index_pipeline_version", Value: "page-tree-v3"},
		{Key: "prompt_profile_version", Value: "llm-index-v5"},
		{Key: "chunk_strategy_version", Value: "section-window-v2"},
		{Key: "tree_builder", Value: "llm-first"},
		{Key: "tree_verification_passed", Value: "true"},
		{Key: "llm_provider_type", Value: "OPENAI"},
		{Key: "llm_model", Value: "gpt-4o-mini"},
		{Key: "llm_request_count", Value: "5"},
		{Key: "llm_input_tokens_estimate", Value: "2400"},
		{Key: "llm_output_tokens_estimate", Value: "900"},
		{Key: "llm_total_tokens_estimate", Value: "3300"},
		{Key: "llm_pricing_available", Value: "true"},
		{Key: "llm_total_cost_usd", Value: "0.001230"},
	}
	evaluation := buildIndexEvaluation([]models.MetadataPair{
		{Key: "tree_coverage_ratio", Value: "0.91"},
		{Key: "tree_total_nodes", Value: "8"},
		{Key: "tree_low_confidence_nodes", Value: "1"},
		{Key: "tree_oversized_nodes", Value: "0"},
		{Key: "tree_max_depth", Value: "4"},
	}, []models.DocumentOutlineNode{{NodeID: "root", Title: "Contrato"}})

	monitoring := buildIndexMonitoring(doc, metadata, evaluation)
	if monitoring == nil {
		t.Fatal("expected monitoring snapshot")
	}
	if monitoring.DurationMs != 2000 {
		t.Fatalf("unexpected duration: %d", monitoring.DurationMs)
	}
	if monitoring.LLM == nil || monitoring.LLM.RequestCount != 5 {
		t.Fatalf("unexpected llm monitoring: %#v", monitoring.LLM)
	}
	if monitoring.EstimatedQualityScore <= 0 {
		t.Fatalf("expected positive quality score, got %f", monitoring.EstimatedQualityScore)
	}
}
