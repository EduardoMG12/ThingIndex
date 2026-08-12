package indeximport

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"thing-index/internal/dto"
	"thing-index/internal/indexio"
)

func TestDiscoverExportFilesFindsJSONRecursively(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write txt file: %v", err)
	}
	exportPath := filepath.Join(nested, "document-index.json")
	if err := os.WriteFile(exportPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write export file: %v", err)
	}

	files, err := DiscoverExportFiles(root)
	if err != nil {
		t.Fatalf("discover export files: %v", err)
	}
	if len(files) != 1 || files[0] != exportPath {
		t.Fatalf("unexpected discovered files: %#v", files)
	}
}

func TestParseExportPayloadRejectsUnexpectedContract(t *testing.T) {
	t.Parallel()

	if _, err := ParseExportPayload([]byte(`{"kind":"wrong","version":"v1"}`)); err == nil {
		t.Fatal("expected invalid export payload to fail")
	}
}

func TestBuildImportRecordsMapsStructuredPayload(t *testing.T) {
	t.Parallel()

	exported := &indexio.DocumentIndexExportEnvelope{
		Kind:       indexio.DocumentIndexExportKind,
		Version:    indexio.DocumentIndexExportVersion,
		ExportedAt: time.Date(2026, time.March, 23, 10, 0, 0, 0, time.UTC),
		Source: indexio.DocumentIndexExportSource{
			Product:       "ThingIndex",
			Origin:        "http://127.0.0.1:3000",
			SourceKey:     "thingindex:http%3A%2F%2F127.0.0.1%3A3000:document:7",
			DocumentID:    7,
			DocumentTitle: "Contrato.pdf",
			MimeType:      "application/pdf",
			ExportRoute:   "/api/v1/documents/7/index/export",
		},
		DocumentIndex: &dto.DocumentIndexResponse{
			Document: &dto.DocumentResponse{
				ID:       7,
				Title:    "Contrato.pdf",
				MimeType: "application/pdf",
				Status:   "INDEXED",
				Collection: &dto.CollectionResponse{
					Name: "Contracts",
				},
				Category: &dto.CategoryResponse{
					Name: "Legal",
				},
			},
			DocumentPages: []dto.DocumentPageResponse{
				{PageNumber: 1, Content: "Page 1", Summary: "Summary", TokenEstimate: 42},
			},
			ParagraphNodes: []dto.ParagraphNodeResponse{
				{SequenceOrder: 1, Content: "Paragraph A"},
			},
			TableSections: []dto.TableSectionResponse{
				{SequenceOrder: 2, Title: "Clauses", Headers: []string{"A"}, Rows: [][]string{{"1"}}},
			},
			MetadataPairs: []dto.MetadataPairResponse{
				{SequenceOrder: 1, Key: "language", Value: "pt-BR"},
			},
			Chunks: []dto.DocumentChunkResponse{
				{SequenceOrder: 1, ChunkType: "paragraph", Content: "Chunk A"},
			},
			OutlineNodes: []dto.OutlineNodeResponse{
				{ID: 10, SequenceOrder: 1, NodeID: "overview", Level: 1, Title: "Overview"},
				{ID: 11, SequenceOrder: 2, NodeID: "payment", ParentID: uintPtr(10), Level: 2, Title: "Payment"},
			},
			DocumentSummary: "Key clauses",
			Monitoring: &dto.IndexMonitoringResponse{
				Status: "DONE",
				Stage:  "POST_PROCESSING",
			},
			IndexEvaluation: &dto.IndexEvaluationResponse{
				CoverageRatio: 100,
			},
		},
	}

	documentRecord, pages, paragraphs, tables, metadataPairs, chunks, outlineNodes, err := buildImportRecords(exported, []byte(`{"kind":"document_index_export"}`), time.Date(2026, time.March, 23, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build import records: %v", err)
	}

	if documentRecord.SourceKey != exported.Source.SourceKey {
		t.Fatalf("unexpected source key %q", documentRecord.SourceKey)
	}
	if documentRecord.CollectionName != "Contracts" || documentRecord.CategoryName != "Legal" {
		t.Fatalf("unexpected imported names: collection=%q category=%q", documentRecord.CollectionName, documentRecord.CategoryName)
	}
	if len(pages) != 1 || pages[0].PageNumber != 1 {
		t.Fatalf("unexpected pages %#v", pages)
	}
	if len(paragraphs) != 1 || paragraphs[0].SequenceOrder != 1 {
		t.Fatalf("unexpected paragraphs %#v", paragraphs)
	}
	if len(tables) != 1 || tables[0].Title != "Clauses" {
		t.Fatalf("unexpected tables %#v", tables)
	}
	if len(metadataPairs) != 1 || metadataPairs[0].Key != "language" {
		t.Fatalf("unexpected metadata pairs %#v", metadataPairs)
	}
	if len(chunks) != 1 || chunks[0].ChunkType != "paragraph" {
		t.Fatalf("unexpected chunks %#v", chunks)
	}
	if len(outlineNodes) != 2 || outlineNodes[1].ParentNodeID != "overview" {
		t.Fatalf("unexpected outline nodes %#v", outlineNodes)
	}
}

func uintPtr(value uint) *uint {
	return &value
}
