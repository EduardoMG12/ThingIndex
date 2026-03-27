package usecase

import (
	"context"
	"math"
	"testing"

	"thing-index/internal/domain"
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
	if len(report.StructuralSignals) != 4 {
		t.Fatalf("expected four structural signals, got %#v", report.StructuralSignals)
	}
	structuralSignalByID := make(map[string]struct {
		Status string
		Rate   float64
		Count  int
		Total  int
	}, len(report.StructuralSignals))
	for _, item := range report.StructuralSignals {
		structuralSignalByID[item.ID] = struct {
			Status string
			Rate   float64
			Count  int
			Total  int
		}{
			Status: item.Status,
			Rate:   item.Rate,
			Count:  item.Count,
			Total:  item.Total,
		}
	}
	if item, ok := structuralSignalByID["expected_query_node_match"]; !ok || item.Status != "pass" || math.Abs(item.Rate-100) > 0.0001 || item.Count != 6 || item.Total != 6 {
		t.Fatalf("unexpected expected_query_node_match signal: %#v", report.StructuralSignals)
	}
	if item, ok := structuralSignalByID["useful_depth"]; !ok || item.Status != "pass" || math.Abs(item.Rate-100) > 0.0001 || item.Count != 5 || item.Total != 5 {
		t.Fatalf("unexpected useful_depth signal: %#v", report.StructuralSignals)
	}
	if item, ok := structuralSignalByID["outline_coverage"]; !ok || item.Status != "pass" || math.Abs(item.Rate-100) > 0.0001 || item.Count != 19 || item.Total != 19 {
		t.Fatalf("unexpected outline_coverage signal: %#v", report.StructuralSignals)
	}
	if item, ok := structuralSignalByID["heading_false_positive"]; !ok || item.Status != "pass" || math.Abs(item.Rate) > 0.0001 || item.Count != 0 || item.Total != 19 {
		t.Fatalf("unexpected heading_false_positive signal: %#v", report.StructuralSignals)
	}
	if report.CurrentSnapshot == nil {
		t.Fatalf("expected current benchmark snapshot, got nil")
	}
	if report.CurrentSnapshot.DatasetVersion != report.DatasetVersion {
		t.Fatalf("expected current snapshot dataset %q, got %#v", report.DatasetVersion, report.CurrentSnapshot)
	}
	if report.CurrentSnapshot.ChunkStrategyVersion != "section-window-v3" {
		t.Fatalf("expected current snapshot chunk strategy, got %#v", report.CurrentSnapshot)
	}
	if len(report.ComparisonHistory) != 1 {
		t.Fatalf("expected one recorded benchmark comparison, got %#v", report.ComparisonHistory)
	}
	if report.LatestComparison == nil {
		t.Fatalf("expected latest benchmark comparison, got nil")
	}
	if report.LatestComparison.Title != "Current runtime vs last recorded snapshot" {
		t.Fatalf("unexpected latest comparison title: %#v", report.LatestComparison)
	}
	if report.LatestComparison.Before.DatasetVersion != "2026-03-relational-baseline-v2" {
		t.Fatalf("unexpected latest comparison before snapshot: %#v", report.LatestComparison.Before)
	}
	if report.LatestComparison.After.DatasetVersion != "2026-03-relational-baseline-v2" {
		t.Fatalf("unexpected latest comparison after snapshot: %#v", report.LatestComparison.After)
	}
	if len(report.LatestComparison.KeyMetricDeltas) != 6 {
		t.Fatalf("expected six key metric deltas, got %#v", report.LatestComparison.KeyMetricDeltas)
	}
	if report.ReferenceComparison == nil {
		t.Fatalf("expected workspace reference comparison, got nil")
	}
	if report.ReferenceComparison.SourcePath != "pageindex/results" {
		t.Fatalf("unexpected reference comparison source path: %#v", report.ReferenceComparison)
	}
	if len(report.ReferenceComparison.Artifacts) != 3 {
		t.Fatalf("expected three reference artifacts, got %#v", report.ReferenceComparison.Artifacts)
	}
	if len(report.ReferenceComparison.Checklist) != 4 {
		t.Fatalf("expected four checklist items, got %#v", report.ReferenceComparison.Checklist)
	}

	checklistByID := make(map[string]string, len(report.ReferenceComparison.Checklist))
	for _, item := range report.ReferenceComparison.Checklist {
		checklistByID[item.ID] = item.Status
	}
	if checklistByID["depth"] != "watch" {
		t.Fatalf("expected depth checklist to remain below the reference bar, got %#v", report.ReferenceComparison.Checklist)
	}
	if checklistByID["coherence"] != "watch" || checklistByID["coverage"] != "fail" || checklistByID["legibility"] != "pass" {
		t.Fatalf("unexpected reference checklist statuses: %#v", report.ReferenceComparison.Checklist)
	}
	if report.RetrievalComparison == nil {
		t.Fatalf("expected retrieval comparison payload, got nil")
	}
	if report.RetrievalComparison.BaselineID != "relational-tree-baseline" {
		t.Fatalf("unexpected retrieval comparison baseline: %#v", report.RetrievalComparison)
	}
	if len(report.RetrievalComparison.Variants) != 3 {
		t.Fatalf("expected three retrieval variants, got %#v", report.RetrievalComparison)
	}
	variantByID := make(map[string]domain.IndexingBenchmarkRetrievalVariant, len(report.RetrievalComparison.Variants))
	for _, item := range report.RetrievalComparison.Variants {
		variantByID[item.ID] = item
	}
	if item, ok := variantByID["relational-tree-baseline"]; !ok || item.Status != "pass" || math.Abs(item.Metrics.ExactMatchAccuracy-100) > 0.0001 {
		t.Fatalf("unexpected relational baseline variant: %#v", report.RetrievalComparison.Variants)
	}
	if item, ok := variantByID["flat-hybrid-control"]; !ok || item.Status != "watch" || math.Abs(item.Metrics.ExactMatchAccuracy-90.9090909090909) > 0.0001 {
		t.Fatalf("unexpected flat hybrid variant: %#v", report.RetrievalComparison.Variants)
	}
	if item, ok := variantByID["flat-page-control"]; !ok || item.Status != "fail" || math.Abs(item.Metrics.TreeNodeTop1Accuracy-33.33333333333333) > 0.0001 {
		t.Fatalf("unexpected flat page variant: %#v", report.RetrievalComparison.Variants)
	}
}
