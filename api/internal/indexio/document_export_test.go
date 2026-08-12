package indexio

import (
	"testing"
	"time"

	"thing-index/internal/dto"
)

func TestNormalizeExportOriginPrefersSchemeAndHost(t *testing.T) {
	t.Parallel()

	got := NormalizeExportOrigin("https://thingindex.dev/documents/7")
	want := "https://thingindex.dev"

	if got != want {
		t.Fatalf("expected normalized origin %q, got %q", want, got)
	}
}

func TestBuildDocumentExportSourceKeyFallsBackWithoutOrigin(t *testing.T) {
	t.Parallel()

	got := BuildDocumentExportSourceKey("", 42)
	want := "thingindex:document:42"

	if got != want {
		t.Fatalf("expected source key %q, got %q", want, got)
	}
}

func TestDocumentIndexExportValidateRejectsMismatchedDocumentID(t *testing.T) {
	t.Parallel()

	exported := &DocumentIndexExportEnvelope{
		Kind:       DocumentIndexExportKind,
		Version:    DocumentIndexExportVersion,
		ExportedAt: time.Now().UTC(),
		Source: DocumentIndexExportSource{
			SourceKey:  "thingindex:document:7",
			DocumentID: 7,
		},
		DocumentIndex: &dto.DocumentIndexResponse{
			Document: &dto.DocumentResponse{ID: 9, Title: "Contrato.pdf"},
		},
	}

	if err := exported.Validate(); err == nil {
		t.Fatal("expected mismatched document id to fail validation")
	}
}

func TestNewDocumentIndexExportBuildsStableMetadata(t *testing.T) {
	t.Parallel()

	index := &dto.DocumentIndexResponse{
		Document: &dto.DocumentResponse{
			ID:       12,
			Title:    "Contrato final.pdf",
			MimeType: "application/pdf",
		},
	}

	exported := NewDocumentIndexExport("http://127.0.0.1:3000/documents", index, time.Date(2026, time.March, 23, 10, 0, 0, 0, time.UTC))
	if exported == nil {
		t.Fatal("expected export payload")
	}
	if err := exported.Validate(); err != nil {
		t.Fatalf("expected valid export payload, got %v", err)
	}
	if exported.Source.Origin != "http://127.0.0.1:3000" {
		t.Fatalf("expected normalized origin, got %q", exported.Source.Origin)
	}
	if exported.Source.SourceKey != "thingindex:http%3A%2F%2F127.0.0.1%3A3000:document:12" {
		t.Fatalf("unexpected source key %q", exported.Source.SourceKey)
	}
	if exported.Source.ExportRoute != "/api/v1/documents/12/index/export" {
		t.Fatalf("unexpected export route %q", exported.Source.ExportRoute)
	}
}
