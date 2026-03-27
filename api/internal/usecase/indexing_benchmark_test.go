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

	if report.DatasetVersion != "2026-03-relational-baseline-v2" {
		t.Fatalf("unexpected dataset version: %#v", report.DatasetVersion)
	}
	if report.CorpusSourcePath != indexingBenchmarkCorpusPath {
		t.Fatalf("unexpected corpus source path: %#v", report.CorpusSourcePath)
	}
	if len(report.Corpus) != 5 {
		t.Fatalf("expected 5 benchmark documents, got %d", len(report.Corpus))
	}
	if len(report.Cases) != 11 {
		t.Fatalf("expected 11 benchmark cases, got %d", len(report.Cases))
	}
	if report.Metrics.CorpusDocuments != 5 || report.Metrics.DocumentQueries != 6 || report.Metrics.CollectionQueries != 5 {
		t.Fatalf("unexpected benchmark counts: %#v", report.Metrics)
	}
	if math.Abs(report.Metrics.StructurePassRate-100) > 0.0001 {
		t.Fatalf("expected improved structure pass rate, got %f", report.Metrics.StructurePassRate)
	}
	if math.Abs(report.Metrics.TreeNodeTop1Accuracy-100) > 0.0001 {
		t.Fatalf("expected improved node top-1 accuracy, got %f", report.Metrics.TreeNodeTop1Accuracy)
	}
	if math.Abs(report.Metrics.TreePageTop1Accuracy-100) > 0.0001 {
		t.Fatalf("expected improved page top-1 accuracy, got %f", report.Metrics.TreePageTop1Accuracy)
	}
	if math.Abs(report.Metrics.CollectionDocumentTop1Accuracy-100) > 0.0001 {
		t.Fatalf("expected improved collection top-1 accuracy, got %f", report.Metrics.CollectionDocumentTop1Accuracy)
	}
	if math.Abs(report.Metrics.ExactMatchAccuracy-100) > 0.0001 {
		t.Fatalf("expected improved exact match accuracy, got %f", report.Metrics.ExactMatchAccuracy)
	}
	if report.Metrics.AverageHeuristicScore >= 90 || report.Metrics.AverageHeuristicScore <= 60 {
		t.Fatalf("expected recalibrated heuristic score, got %f", report.Metrics.AverageHeuristicScore)
	}
}
