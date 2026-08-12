//go:build integration

package indeximport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"thing-index/internal/dto"
	"thing-index/internal/indexio"
	"thing-index/internal/testutil"
)

func TestServiceImportPathCreatesUpdatesAndSkipsExports(t *testing.T) {
	t.Parallel()

	db := testutil.OpenIsolatedPostgresDB(t)
	service := NewService(db)
	service.now = func() time.Time {
		return time.Date(2026, time.March, 23, 12, 0, 0, 0, time.UTC)
	}

	root := t.TempDir()
	exportPath := filepath.Join(root, "contract-index.json")

	writeExport := func(summary string) {
		t.Helper()

		exported := indexio.NewDocumentIndexExport("http://127.0.0.1:3000", &dto.DocumentIndexResponse{
			Document: &dto.DocumentResponse{
				ID:       7,
				Title:    "Contrato.pdf",
				MimeType: "application/pdf",
				Status:   "INDEXED",
			},
			DocumentPages: []dto.DocumentPageResponse{
				{PageNumber: 1, Content: "Page 1"},
			},
			ParagraphNodes: []dto.ParagraphNodeResponse{
				{SequenceOrder: 1, Content: "Paragraph A"},
			},
			DocumentSummary: summary,
		}, time.Date(2026, time.March, 23, 10, 0, 0, 0, time.UTC))

		payload, err := jsonMarshal(exported)
		if err != nil {
			t.Fatalf("marshal export payload: %v", err)
		}
		if err := os.WriteFile(exportPath, payload, 0o644); err != nil {
			t.Fatalf("write export file: %v", err)
		}
	}

	ctx := context.Background()

	writeExport("first summary")
	report, err := service.ImportPath(ctx, root)
	if err != nil {
		t.Fatalf("first import path: %v", err)
	}
	if report.CreatedDocuments != 1 || report.UpdatedDocuments != 0 || report.SkippedFiles != 0 {
		t.Fatalf("unexpected first report %#v", report)
	}

	writeExport("updated summary")
	report, err = service.ImportPath(ctx, root)
	if err != nil {
		t.Fatalf("second import path: %v", err)
	}
	if report.CreatedDocuments != 0 || report.UpdatedDocuments != 1 || report.SkippedFiles != 0 {
		t.Fatalf("unexpected second report %#v", report)
	}

	report, err = service.ImportPath(ctx, root)
	if err != nil {
		t.Fatalf("third import path: %v", err)
	}
	if report.CreatedDocuments != 0 || report.UpdatedDocuments != 0 || report.SkippedFiles != 1 {
		t.Fatalf("unexpected third report %#v", report)
	}

	var imported ImportedDocument
	if err := db.First(&imported).Error; err != nil {
		t.Fatalf("load imported document: %v", err)
	}
	if imported.DocumentSummary != "updated summary" {
		t.Fatalf("expected updated summary, got %q", imported.DocumentSummary)
	}

	var pageCount int64
	if err := db.Model(&ImportedDocumentPage{}).Count(&pageCount).Error; err != nil {
		t.Fatalf("count imported pages: %v", err)
	}
	if pageCount != 1 {
		t.Fatalf("expected 1 imported page, got %d", pageCount)
	}
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}
