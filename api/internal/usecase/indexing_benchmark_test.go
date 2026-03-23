package usecase

import (
	"context"
	"math"
	"testing"
)

func TestIndexingBenchmarkProviderBuildsPublicReport(t *testing.T) {
	t.Parallel()

	provider := NewIndexingBenchmarkProvider()
	report, err := provider.GetPublicIndexingBenchmark(context.Background())
	if err != nil {
		t.Fatalf("GetPublicIndexingBenchmark returned error: %v", err)
	}

	if report.DatasetVersion != "2026-03-relational-baseline-v1" {
		t.Fatalf("unexpected dataset version: %#v", report.DatasetVersion)
	}
	if report.CorpusSourcePath != indexingBenchmarkCorpusPath {
		t.Fatalf("unexpected corpus source path: %#v", report.CorpusSourcePath)
	}
	if len(report.Corpus) != 3 {
		t.Fatalf("expected 3 benchmark documents, got %d", len(report.Corpus))
	}
	if len(report.Cases) != 7 {
		t.Fatalf("expected 7 benchmark cases, got %d", len(report.Cases))
	}
	if report.Metrics.CorpusDocuments != 3 || report.Metrics.DocumentQueries != 4 || report.Metrics.CollectionQueries != 3 {
		t.Fatalf("unexpected benchmark counts: %#v", report.Metrics)
	}
	if math.Abs(report.Metrics.StructurePassRate-33.33333333333333) > 0.0001 {
		t.Fatalf("expected current structure pass rate, got %f", report.Metrics.StructurePassRate)
	}
	if math.Abs(report.Metrics.TreeNodeTop1Accuracy-0) > 0.0001 {
		t.Fatalf("expected current node top-1 accuracy, got %f", report.Metrics.TreeNodeTop1Accuracy)
	}
	if math.Abs(report.Metrics.CollectionDocumentTop1Accuracy-66.66666666666666) > 0.0001 {
		t.Fatalf("expected current collection top-1 accuracy, got %f", report.Metrics.CollectionDocumentTop1Accuracy)
	}
	if math.Abs(report.Metrics.ExactMatchAccuracy-28.57142857142857) > 0.0001 {
		t.Fatalf("expected current exact match accuracy, got %f", report.Metrics.ExactMatchAccuracy)
	}
	if report.Metrics.AverageHeuristicScore < 90 {
		t.Fatalf("expected healthy heuristic baseline, got %f", report.Metrics.AverageHeuristicScore)
	}
}
