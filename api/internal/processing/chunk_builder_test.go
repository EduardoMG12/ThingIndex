package processing

import (
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestBuildDocumentChunksAddsParagraphAndTableChunks(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		ParagraphNodes: []models.ParagraphNode{
			{DocumentID: 1, SequenceOrder: 0, Content: "Introduction to the agreement."},
			{DocumentID: 1, SequenceOrder: 1, Content: "Payment must happen every month."},
			{DocumentID: 1, SequenceOrder: 2, Content: "Termination requires written notice."},
		},
		TableSections: []models.TableSection{
			{
				DocumentID:    1,
				SequenceOrder: 0,
				Title:         "Fees",
				Headers:       []string{"Item", "Value"},
				Rows: [][]string{
					{"Monthly Fee", "1000"},
					{"Penalty", "200"},
				},
			},
		},
		OutlineNodes: []models.DocumentOutlineNode{
			{DocumentID: 1, SequenceOrder: 0, Level: 1, Title: "Overview"},
			{DocumentID: 1, SequenceOrder: 1, Level: 1, Title: "Commercial Terms"},
		},
	}

	chunks := buildDocumentChunks(1, result)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].ChunkType != documentChunkTypeParagraphWindow {
		t.Fatalf("unexpected first chunk type: %s", chunks[0].ChunkType)
	}
	if chunks[0].Title == "" {
		t.Fatal("expected paragraph chunk to include an outline hint title")
	}
	if chunks[0].Metadata["chunk_strategy_version"] != chunkStrategyVersion {
		t.Fatalf("expected paragraph chunk strategy version %q, got %#v", chunkStrategyVersion, chunks[0].Metadata["chunk_strategy_version"])
	}

	if chunks[1].ChunkType != documentChunkTypeTable {
		t.Fatalf("unexpected second chunk type: %s", chunks[1].ChunkType)
	}
	if chunks[1].Title != "Fees" {
		t.Fatalf("unexpected table chunk title: %q", chunks[1].Title)
	}
	if chunks[1].Metadata["row_count"] != 2 {
		t.Fatalf("expected table chunk row_count 2, got %#v", chunks[1].Metadata["row_count"])
	}
}

func TestAppendPipelineMetadataAddsVersionMarkers(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		MetadataPairs: []models.MetadataPair{
			{DocumentID: 9, SequenceOrder: 0, Key: "paragraph_count", Value: "3"},
		},
	}

	appendPipelineMetadata(&result, 9, "applied")

	if len(result.MetadataPairs) != 5 {
		t.Fatalf("expected 5 metadata pairs, got %d", len(result.MetadataPairs))
	}

	keys := map[string]string{}
	for _, pair := range result.MetadataPairs {
		keys[pair.Key] = pair.Value
	}

	if keys["index_pipeline_version"] != indexPipelineVersion {
		t.Fatalf("unexpected index pipeline version: %q", keys["index_pipeline_version"])
	}
	if keys["chunk_strategy_version"] != chunkStrategyVersion {
		t.Fatalf("unexpected chunk strategy version: %q", keys["chunk_strategy_version"])
	}
	if keys["prompt_profile_version"] != promptProfileVersion {
		t.Fatalf("unexpected prompt profile version: %q", keys["prompt_profile_version"])
	}
	if keys["enrichment_status"] != "applied" {
		t.Fatalf("unexpected enrichment status: %q", keys["enrichment_status"])
	}
}

func TestAppendLLMUsageMetadataAddsTelemetry(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{}
	usage := llmUsageSnapshot{
		ProviderType:          "OPENAI",
		Model:                 "gpt-4o-mini",
		RequestCount:          4,
		EstimatedInputTokens:  1200,
		EstimatedOutputTokens: 340,
		EstimatedTotalTokens:  1540,
		PricingAvailable:      true,
		InputCostUSD:          0.000180,
		OutputCostUSD:         0.000204,
		TotalCostUSD:          0.000384,
	}

	appendLLMUsageMetadata(&result, 2, usage)

	keys := map[string]string{}
	for _, pair := range result.MetadataPairs {
		keys[pair.Key] = pair.Value
	}

	if keys["llm_provider_type"] != "OPENAI" {
		t.Fatalf("unexpected llm provider type: %q", keys["llm_provider_type"])
	}
	if keys["llm_model"] != "gpt-4o-mini" {
		t.Fatalf("unexpected llm model: %q", keys["llm_model"])
	}
	if keys["llm_request_count"] != "4" {
		t.Fatalf("unexpected llm request count: %q", keys["llm_request_count"])
	}
	if keys["llm_total_cost_usd"] == "" {
		t.Fatal("expected total cost metadata")
	}
}

func TestBuildDocumentChunksRespectsSectionBoundaries(t *testing.T) {
	t.Parallel()

	pageOne := 1
	pageTwo := 2
	result := domain.IndexingResult{
		ParagraphNodes: []models.ParagraphNode{
			{
				DocumentID:    2,
				SequenceOrder: 0,
				PageNumber:    &pageOne,
				Content:       "Opening paragraph.",
				Metadata: map[string]any{
					"section_key":   "intro",
					"section_title": "Introduction",
				},
			},
			{
				DocumentID:    2,
				SequenceOrder: 1,
				PageNumber:    &pageOne,
				Content:       "Still part of the intro.",
				Metadata: map[string]any{
					"section_key":   "intro",
					"section_title": "Introduction",
				},
			},
			{
				DocumentID:    2,
				SequenceOrder: 2,
				PageNumber:    &pageTwo,
				Content:       "Payment due in 15 days.",
				Metadata: map[string]any{
					"section_key":   "payment-terms",
					"section_title": "Payment Terms",
				},
			},
		},
	}

	chunks := buildDocumentChunks(2, result)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Title != "Introduction" {
		t.Fatalf("unexpected first chunk title: %q", chunks[0].Title)
	}
	if chunks[1].Title != "Payment Terms" {
		t.Fatalf("unexpected second chunk title: %q", chunks[1].Title)
	}
	if chunks[0].Metadata["page_start"] != 1 || chunks[0].Metadata["page_end"] != 1 {
		t.Fatalf("unexpected first chunk page range: %#v", chunks[0].Metadata)
	}
	if chunks[1].Metadata["section_key"] != "payment-terms" {
		t.Fatalf("unexpected second chunk section key: %#v", chunks[1].Metadata["section_key"])
	}
}
