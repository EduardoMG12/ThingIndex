//go:build integration

package usecase

import (
	"context"
	"testing"
)

func TestIndexingBenchmarkIntegrationUsesEmbeddedCorpusWithRealIndexer(t *testing.T) {
	t.Parallel()

	provider := NewIndexingBenchmarkProvider()
	report, err := provider.GetPublicIndexingBenchmark(context.Background())
	if err != nil {
		t.Fatalf("GetPublicIndexingBenchmark returned error: %v", err)
	}

	if report.Metrics.CorpusDocuments != 5 || len(report.Cases) != 11 {
		t.Fatalf("expected populated benchmark report, got %#v", report.Metrics)
	}
	if report.Metrics.ExactMatchAccuracy <= 0 {
		t.Fatalf("expected some passing benchmark cases, got %#v", report.Metrics)
	}
	if report.CurrentSnapshot == nil || report.LatestComparison == nil {
		t.Fatalf("expected recorded benchmark snapshots, got snapshot=%#v latest=%#v", report.CurrentSnapshot, report.LatestComparison)
	}

	structurePasses := 0
	for _, item := range report.Corpus {
		if !item.RetrievalReady {
			t.Fatalf("expected retrieval ready corpus document %#v", item)
		}
		if item.StructurePassed {
			structurePasses++
		}
	}
	if structurePasses == 0 {
		t.Fatalf("expected at least one structure pass, got %#v", report.Corpus)
	}
}
