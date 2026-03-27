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

	currentSnapshot, ok := data["current_snapshot"].(map[string]any)
	if !ok || currentSnapshot["chunk_strategy_version"] != "section-window-v2" {
		t.Fatalf("expected current snapshot payload, got %#v", data["current_snapshot"])
	}

	latestComparison, ok := data["latest_comparison"].(map[string]any)
	if !ok || latestComparison["title"] != "Baseline -> Current" {
		t.Fatalf("expected latest comparison payload, got %#v", data["latest_comparison"])
	}
}
