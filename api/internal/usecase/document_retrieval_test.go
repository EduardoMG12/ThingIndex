package usecase

import (
	"testing"
	"time"

	"thing-index/internal/domain"
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

func TestBuildTreeRetrievalPrefersStrongChildMatchOverGenericAncestor(t *testing.T) {
	t.Parallel()

	pageOne := 1
	nodes := []models.DocumentOutlineNode{
		{
			NodeID:        "root",
			SequenceOrder: 0,
			Level:         1,
			Title:         "Contract",
			Summary:       "General contract information.",
			StartPage:     &pageOne,
			EndPage:       &pageOne,
		},
		{
			NodeID:        "payment",
			SequenceOrder: 1,
			Level:         2,
			Title:         "Payment Terms",
			Summary:       "Payment deadline after invoice receipt.",
			StartPage:     &pageOne,
			EndPage:       &pageOne,
			Metadata: map[string]any{
				"parent_node_id": "root",
			},
		},
	}

	result := buildTreeRetrieval("payment deadline invoice", nodes, []models.DocumentPage{
		{PageNumber: 1, Content: "Payment deadline after invoice receipt."},
	})
	if result == nil {
		t.Fatal("expected retrieval result")
	}
	if len(result.NodeHits) == 0 || result.NodeHits[0].NodeID != "payment" {
		t.Fatalf("expected child node to rank first, got %#v", result.NodeHits)
	}
	if len(result.TraversalSteps) != 2 || result.TraversalSteps[1].NodeID != "payment" {
		t.Fatalf("expected traversal path to end at child node, got %#v", result.TraversalSteps)
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

func TestBuildIndexEvaluationFallsBackToNodeCoverageWhenTreeMetadataIsMissing(t *testing.T) {
	t.Parallel()

	pageOne := 1
	pageTwo := 2
	evaluation := buildIndexEvaluation(nil, []models.DocumentOutlineNode{
		{
			NodeID:    "root",
			Title:     "Contrato",
			Level:     1,
			Summary:   "Resumo geral",
			StartPage: &pageOne,
			EndPage:   &pageTwo,
		},
		{
			NodeID:    "payment",
			Title:     "Payment Terms",
			Level:     2,
			StartPage: &pageOne,
			EndPage:   &pageOne,
		},
	})

	if evaluation == nil {
		t.Fatal("expected evaluation result")
	}
	if evaluation.CoverageRatio != 1 {
		t.Fatalf("expected fallback coverage ratio 1, got %f", evaluation.CoverageRatio)
	}
	if evaluation.SummaryCoverageRatio != 0.5 {
		t.Fatalf("expected fallback summary coverage ratio 0.5, got %f", evaluation.SummaryCoverageRatio)
	}
	if evaluation.MaxDepth != 2 {
		t.Fatalf("expected fallback max depth 2, got %d", evaluation.MaxDepth)
	}
	if !evaluation.RetrievalReady {
		t.Fatalf("expected retrieval ready evaluation, got %#v", evaluation)
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
		{Key: "tree_summary_coverage_ratio", Value: "0.88"},
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

func TestAggregateDocumentMonitoringSummaryAggregatesTelemetry(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.March, 16, 11, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(2 * time.Second)
	completedAtTwo := startedAt.Add(5 * time.Second)

	summary := aggregateDocumentMonitoringSummary(
		[]models.Document{
			{
				ID:              1,
				Status:          "INDEXED",
				ProcessingStage: "COMPLETED",
				IndexStartedAt:  &startedAt,
				IndexedAt:       &completedAt,
			},
			{
				ID:              2,
				Status:          "FAILED",
				ProcessingStage: "MOVED_TO_DLQ",
				IndexStartedAt:  &startedAt,
				IndexedAt:       &completedAtTwo,
			},
			{
				ID:              3,
				Status:          "ORPHANED",
				ProcessingStage: "SOURCE_MISSING",
			},
			{
				ID:              4,
				Status:          "QUEUED",
				ProcessingStage: "RETRY_SCHEDULED",
			},
		},
		[]models.MetadataPair{
			{DocumentID: 1, Key: "tree_coverage_ratio", Value: "0.91"},
			{DocumentID: 1, Key: "tree_summary_coverage_ratio", Value: "0.88"},
			{DocumentID: 1, Key: "tree_total_nodes", Value: "8"},
			{DocumentID: 1, Key: "tree_low_confidence_nodes", Value: "1"},
			{DocumentID: 1, Key: "tree_oversized_nodes", Value: "0"},
			{DocumentID: 1, Key: "llm_provider_type", Value: "OPENAI"},
			{DocumentID: 1, Key: "llm_model", Value: "gpt-4o-mini"},
			{DocumentID: 1, Key: "llm_request_count", Value: "5"},
			{DocumentID: 1, Key: "llm_input_tokens_estimate", Value: "2400"},
			{DocumentID: 1, Key: "llm_output_tokens_estimate", Value: "900"},
			{DocumentID: 1, Key: "llm_total_tokens_estimate", Value: "3300"},
			{DocumentID: 1, Key: "llm_pricing_available", Value: "true"},
			{DocumentID: 1, Key: "llm_total_cost_usd", Value: "0.001230"},
			{DocumentID: 1, Key: "index_completed_at", Value: completedAt.Format(time.RFC3339)},
			{DocumentID: 2, Key: "index_completed_at", Value: completedAtTwo.Format(time.RFC3339)},
		},
	)

	if summary.TotalDocuments != 4 || summary.IndexedDocuments != 1 || summary.FailedDocuments != 1 {
		t.Fatalf("unexpected document counters: %#v", summary)
	}
	if summary.PermanentFailedDocuments != 0 {
		t.Fatalf("expected no permanent failures in this sample, got %#v", summary)
	}
	if summary.QueuedDocuments != 1 || summary.RetryingDocuments != 1 || summary.OrphanedDocuments != 1 || summary.DLQDocuments != 1 {
		t.Fatalf("unexpected operational counters: %#v", summary)
	}
	if summary.DocumentsWithLLM != 1 || summary.TotalLLMRequests != 5 || summary.TotalTokens != 3300 {
		t.Fatalf("unexpected llm totals: %#v", summary)
	}
	if summary.TotalEstimatedCostUSD != 0.001230 {
		t.Fatalf("unexpected cost total: %f", summary.TotalEstimatedCostUSD)
	}
	if summary.AverageDurationMs != 3500 {
		t.Fatalf("unexpected average duration: %d", summary.AverageDurationMs)
	}
	if summary.AverageQualityScore <= 0 {
		t.Fatalf("expected positive quality score: %#v", summary)
	}
	if summary.LastCompletedAt == nil || !summary.LastCompletedAt.Equal(completedAtTwo) {
		t.Fatalf("unexpected last completed at: %#v", summary.LastCompletedAt)
	}
	if len(summary.ModelBreakdown) != 1 || summary.ModelBreakdown[0].Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model breakdown: %#v", summary.ModelBreakdown)
	}
}

func TestBuildCollectionRetrievalRanksDocumentsAfterAggregatingAllSignals(t *testing.T) {
	t.Parallel()

	pageOne := 1
	pageTwo := 2
	result := buildCollectionRetrieval(
		"incident retention years",
		[]models.Document{
			{ID: 1, Title: "Contract Appendix"},
			{ID: 2, Title: "Incident Playbook"},
		},
		[]models.DocumentPage{
			{DocumentID: 1, PageNumber: 1, Content: "General appendix."},
			{DocumentID: 2, PageNumber: 2, Content: "Incident retention is tracked by policy."},
		},
		[]models.DocumentOutlineNode{
			{DocumentID: 1, NodeID: "d1-root", Title: "General Terms", Summary: "Retention", StartPage: &pageOne, EndPage: &pageOne},
			{DocumentID: 2, NodeID: "d2-root", Title: "Evidence", Summary: "Incident retention", StartPage: &pageTwo, EndPage: &pageTwo},
		},
		[]models.DocumentChunk{
			{DocumentID: 2, ChunkType: "paragraph_window", Title: "Evidence Retention", Content: "Incident evidence is retained for 7 years."},
		},
		nil,
	)

	if result == nil {
		t.Fatal("expected retrieval result")
	}
	if len(result.DocumentHits) == 0 || result.DocumentHits[0].DocumentID != 2 {
		t.Fatalf("expected aggregated document ranking to favor document 2, got %#v", result.DocumentHits)
	}
}

func TestBuildCollectionRetrievalUsesTableSignalsForDocumentRanking(t *testing.T) {
	t.Parallel()

	pageOne := 1
	result := buildCollectionRetrieval(
		"security incidents seven years",
		[]models.Document{
			{ID: 10, Title: "Retention Policy"},
			{ID: 11, Title: "Vendor Guide"},
		},
		nil,
		[]models.DocumentOutlineNode{
			{DocumentID: 10, NodeID: "retention", Title: "Retention Schedule", Summary: "Operational retention windows.", StartPage: &pageOne, EndPage: &pageOne},
		},
		nil,
		[]models.TableSection{
			{
				DocumentID: 10,
				Title:      "Retention Schedule",
				Headers:    []string{"Artifact", "Retention"},
				Rows:       [][]string{{"Security incidents", "7 years"}},
			},
		},
	)

	if result == nil {
		t.Fatal("expected retrieval result")
	}
	if len(result.DocumentHits) == 0 || result.DocumentHits[0].DocumentID != 10 {
		t.Fatalf("expected table-backed document to rank first, got %#v", result.DocumentHits)
	}
}

func TestEstimateIndexQualityScorePenalizesMissingSummaries(t *testing.T) {
	t.Parallel()

	withSummaries := estimateIndexQualityScore(&domain.IndexEvaluationResult{
		CoverageRatio:        1,
		SummaryCoverageRatio: 1,
		TotalNodes:           4,
		MaxDepth:             2,
		RetrievalReady:       true,
	})
	withoutSummaries := estimateIndexQualityScore(&domain.IndexEvaluationResult{
		CoverageRatio:        1,
		SummaryCoverageRatio: 0,
		TotalNodes:           4,
		MaxDepth:             2,
		RetrievalReady:       true,
	})

	if withoutSummaries >= withSummaries {
		t.Fatalf("expected missing summaries to reduce quality score, got with=%f without=%f", withSummaries, withoutSummaries)
	}
	if withoutSummaries >= 80 {
		t.Fatalf("expected missing summaries to avoid overly optimistic quality score, got %f", withoutSummaries)
	}
}

func TestTokenizeRetrievalTextExpandsNumericWordsAndDigits(t *testing.T) {
	t.Parallel()

	tokens := tokenizeRetrievalText("security incidents seven years 7")

	expected := map[string]struct{}{
		"security":  {},
		"incidents": {},
		"seven":     {},
		"7":         {},
		"years":     {},
	}
	for token := range expected {
		found := false
		for _, candidate := range tokens {
			if candidate == token {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected token %q in %#v", token, tokens)
		}
	}
}
