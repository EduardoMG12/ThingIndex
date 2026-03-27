package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type benchmarkProviderStub struct {
	report *domain.IndexingBenchmarkReport
	err    error
}

func (s *benchmarkProviderStub) GetPublicIndexingBenchmark(context.Context) (*domain.IndexingBenchmarkReport, error) {
	return s.report, s.err
}

func TestMetaHandlerGetRuntimeProfile(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewMetaHandler(&benchmarkProviderStub{})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/meta/runtime-profile", nil)

	handler.GetRuntimeProfile(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected response data map, got %#v", payload.Data)
	}

	if data["product_name"] != "ThingIndex" {
		t.Fatalf("expected product_name ThingIndex, got %#v", data["product_name"])
	}

	targetUsers, ok := data["target_users"].([]any)
	if !ok || len(targetUsers) < 2 {
		t.Fatalf("expected target_users payload, got %#v", data["target_users"])
	}

	artifacts, ok := data["artifact_refs"].([]any)
	if !ok || len(artifacts) < 2 {
		t.Fatalf("expected artifact refs payload, got %#v", data["artifact_refs"])
	}
}

func TestMetaHandlerGetIndexingBenchmark(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewMetaHandler(&benchmarkProviderStub{
		report: &domain.IndexingBenchmarkReport{
			DatasetVersion:       "2026-03-relational-baseline-v1",
			CorpusSourcePath:     "internal/usecase/testdata/indexing-benchmark/corpus.json",
			LastReviewedOn:       "2026-03-22",
			Summary:              "Controlled benchmark.",
			Methodology:          []string{"Run the deterministic indexer."},
			PassCriteria:         []string{"Top-1 exact match."},
			Limitations:          []string{"Small corpus."},
			HeuristicQualityNote: "Heuristic is operational only.",
			Corpus: []domain.IndexingBenchmarkCorpusDocument{
				{
					ID:                    "service-agreement",
					Title:                 "Master Service Agreement",
					StructurePassed:       true,
					RetrievalReady:        true,
					HeuristicQualityScore: 98,
				},
			},
			Metrics: domain.IndexingBenchmarkMetrics{
				CorpusDocuments:    1,
				DocumentQueries:    1,
				ExactMatchAccuracy: 100,
			},
			StructuralSignals: []domain.IndexingBenchmarkStructuralSignal{
				{
					ID:        "expected_query_node_match",
					Direction: "higher_is_better",
					Status:    "pass",
					Rate:      100,
					Count:     1,
					Total:     1,
				},
			},
			CurrentSnapshot: &domain.IndexingBenchmarkSnapshot{
				ID:                   "2026-03-relational-baseline-v1",
				Label:                "Current runtime",
				RecordedOn:           "2026-03-22",
				DatasetVersion:       "2026-03-relational-baseline-v1",
				PipelineVersion:      "page-tree-v3",
				PromptProfileVersion: "llm-index-v5",
				ChunkStrategyVersion: "section-window-v2",
				Metrics: domain.IndexingBenchmarkMetrics{
					CorpusDocuments:    1,
					DocumentQueries:    1,
					ExactMatchAccuracy: 100,
				},
			},
			LatestComparison: &domain.IndexingBenchmarkComparison{
				ID:         "baseline",
				Title:      "Baseline -> Current",
				RecordedOn: "2026-03-22",
				Before: domain.IndexingBenchmarkSnapshot{
					ID:             "older",
					Label:          "Older",
					RecordedOn:     "2026-03-20",
					DatasetVersion: "2026-03-relational-baseline-v0",
				},
				After: domain.IndexingBenchmarkSnapshot{
					ID:             "current",
					Label:          "Current",
					RecordedOn:     "2026-03-22",
					DatasetVersion: "2026-03-relational-baseline-v1",
				},
				KeyMetricDeltas: []domain.IndexingBenchmarkMetricDelta{
					{Metric: "exact_match_accuracy", PreviousValue: 20, CurrentValue: 100, Delta: 80},
				},
			},
			ReferenceComparison: &domain.IndexingBenchmarkReferenceComparison{
				SourcePath:     "pageindex/results",
				LastReviewedOn: "2026-03-27",
				Summary:        "Reference comparison summary.",
				Artifacts: []domain.IndexingBenchmarkReferenceArtifact{
					{
						ID:           "earthmover",
						Label:        "Research paper reference",
						SourcePath:   "pageindex/results/earthmover_structure.json",
						DocumentName: "earthmover.pdf",
						TotalNodes:   21,
						MaxDepth:     3,
					},
				},
				Checklist: []domain.IndexingBenchmarkReferenceChecklistItem{
					{
						ID:             "depth",
						Dimension:      "depth",
						Label:          "Useful structural depth",
						Status:         "watch",
						CurrentValue:   "max_depth=2",
						ReferenceValue: "max_depth=3",
						Summary:        "Still shallower than the reference.",
					},
				},
			},
			RetrievalComparison: &domain.IndexingBenchmarkRetrievalComparison{
				Summary:    "Retrieval comparison summary.",
				BaselineID: "relational-tree-baseline",
				Notes:      []string{"Experimental controls only."},
				Variants: []domain.IndexingBenchmarkRetrievalVariant{
					{
						ID:       "relational-tree-baseline",
						Label:    "Current relational baseline",
						Category: "baseline",
						Status:   "pass",
						Summary:  "Production path.",
						Metrics: domain.IndexingBenchmarkRetrievalVariantMetrics{
							DocumentQueries:                1,
							CollectionQueries:              1,
							TreeNodeTop1Accuracy:           100,
							TreePageTop1Accuracy:           100,
							CollectionDocumentTop1Accuracy: 100,
							ExactMatchAccuracy:             100,
						},
						KeyMetricDeltas: []domain.IndexingBenchmarkMetricDelta{
							{Metric: "exact_match_accuracy", PreviousValue: 100, CurrentValue: 100, Delta: 0},
						},
					},
				},
			},
			Cases: []domain.IndexingBenchmarkCaseResult{
				{
					ID:                "service-payment-deadline",
					Scope:             "DOCUMENT_TREE",
					Query:             "payment deadline invoice receipt",
					DocumentTitle:     "Master Service Agreement",
					ExpectedNodeTitle: "2 Payment Terms",
					ActualNodeTitle:   "2 Payment Terms",
					NodeMatched:       true,
					PageMatched:       true,
					Passed:            true,
				},
			},
		},
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/meta/indexing-benchmark", nil)

	handler.GetIndexingBenchmark(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected response data map, got %#v", payload.Data)
	}

	if data["dataset_version"] != "2026-03-relational-baseline-v1" {
		t.Fatalf("expected dataset_version, got %#v", data["dataset_version"])
	}

	metrics, ok := data["metrics"].(map[string]any)
	if !ok || metrics["exact_match_accuracy"] != float64(100) {
		t.Fatalf("expected metrics payload, got %#v", data["metrics"])
	}

	structuralSignals, ok := data["structural_signals"].([]any)
	if !ok || len(structuralSignals) != 1 {
		t.Fatalf("expected structural signals payload, got %#v", data["structural_signals"])
	}

	currentSnapshot, ok := data["current_snapshot"].(map[string]any)
	if !ok || currentSnapshot["chunk_strategy_version"] != "section-window-v2" {
		t.Fatalf("expected current snapshot payload, got %#v", data["current_snapshot"])
	}

	latestComparison, ok := data["latest_comparison"].(map[string]any)
	if !ok || latestComparison["title"] != "Baseline -> Current" {
		t.Fatalf("expected latest comparison payload, got %#v", data["latest_comparison"])
	}

	referenceComparison, ok := data["reference_comparison"].(map[string]any)
	if !ok || referenceComparison["source_path"] != "pageindex/results" {
		t.Fatalf("expected reference comparison payload, got %#v", data["reference_comparison"])
	}

	retrievalComparison, ok := data["retrieval_comparison"].(map[string]any)
	if !ok || retrievalComparison["baseline_id"] != "relational-tree-baseline" {
		t.Fatalf("expected retrieval comparison payload, got %#v", data["retrieval_comparison"])
	}
}
