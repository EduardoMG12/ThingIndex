package indexing

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"thing-index/internal/models"

	"github.com/xuri/excelize/v2"
)

func TestDeterministicIndexerIndexHTML(t *testing.T) {
	t.Parallel()

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       42,
		MimeType: "text/html",
	}

	html := `
	<html>
		<body>
			<h1>Contract Overview</h1>
			<p>First paragraph.</p>
			<ul><li>Important bullet</li></ul>
			<table>
				<tr><th>Clause</th><th>Value</th></tr>
				<tr><td>Term</td><td>12 months</td></tr>
			</table>
		</body>
	</html>`

	result, err := indexer.Index(context.Background(), doc, bytes.NewBufferString(html))
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if len(result.OutlineNodes) != 1 {
		t.Fatalf("expected 1 outline node, got %d", len(result.OutlineNodes))
	}
	if result.OutlineNodes[0].Title != "Contract Overview" {
		t.Fatalf("unexpected outline title: %q", result.OutlineNodes[0].Title)
	}

	if len(result.ParagraphNodes) != 2 {
		t.Fatalf("expected 2 paragraph nodes, got %d", len(result.ParagraphNodes))
	}
	if result.ParagraphNodes[0].Content != "First paragraph." {
		t.Fatalf("unexpected first paragraph: %q", result.ParagraphNodes[0].Content)
	}
	if result.ParagraphNodes[1].Content != "Important bullet" {
		t.Fatalf("unexpected second paragraph: %q", result.ParagraphNodes[1].Content)
	}

	if len(result.TableSections) != 1 {
		t.Fatalf("expected 1 table section, got %d", len(result.TableSections))
	}
	if len(result.DocumentPages) != 1 || result.DocumentPages[0].PageNumber != 1 {
		t.Fatalf("unexpected document pages: %#v", result.DocumentPages)
	}
	if !slices.Equal(result.TableSections[0].Headers, []string{"Clause", "Value"}) {
		t.Fatalf("unexpected table headers: %#v", result.TableSections[0].Headers)
	}
	if len(result.TableSections[0].Rows) != 1 || !slices.Equal(result.TableSections[0].Rows[0], []string{"Term", "12 months"}) {
		t.Fatalf("unexpected table rows: %#v", result.TableSections[0].Rows)
	}

	assertMetadataValue(t, result.MetadataPairs, "paragraph_count", "2")
	assertMetadataValue(t, result.MetadataPairs, "table_count", "1")
	assertMetadataValue(t, result.MetadataPairs, "outline_count", "1")
}

func TestDeterministicIndexerIndexSpreadsheet(t *testing.T) {
	t.Parallel()

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       7,
		MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	workbook := excelize.NewFile()
	workbook.SetSheetName("Sheet1", "Vendors")
	if err := workbook.SetSheetRow("Vendors", "A1", &[]string{"Name", "Plan"}); err != nil {
		t.Fatalf("SetSheetRow() header failed: %v", err)
	}
	if err := workbook.SetSheetRow("Vendors", "A2", &[]string{"Acme", "Enterprise"}); err != nil {
		t.Fatalf("SetSheetRow() row failed: %v", err)
	}

	var payload bytes.Buffer
	if err := workbook.Write(&payload); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if err := workbook.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	result, err := indexer.Index(context.Background(), doc, bytes.NewReader(payload.Bytes()))
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if len(result.OutlineNodes) != 1 || result.OutlineNodes[0].Title != "Vendors" {
		t.Fatalf("unexpected outline nodes: %#v", result.OutlineNodes)
	}
	if len(result.DocumentPages) != 1 || result.DocumentPages[0].PageNumber != 1 {
		t.Fatalf("unexpected document pages: %#v", result.DocumentPages)
	}
	if len(result.ParagraphNodes) != 1 || result.ParagraphNodes[0].Content != "Planilha Vendors com 2 linhas." {
		t.Fatalf("unexpected paragraph nodes: %#v", result.ParagraphNodes)
	}
	if len(result.TableSections) != 1 {
		t.Fatalf("expected 1 table section, got %d", len(result.TableSections))
	}
	if !slices.Equal(result.TableSections[0].Headers, []string{"Name", "Plan"}) {
		t.Fatalf("unexpected sheet headers: %#v", result.TableSections[0].Headers)
	}
	if len(result.TableSections[0].Rows) != 1 || !slices.Equal(result.TableSections[0].Rows[0], []string{"Acme", "Enterprise"}) {
		t.Fatalf("unexpected sheet rows: %#v", result.TableSections[0].Rows)
	}

	assertMetadataValue(t, result.MetadataPairs, "paragraph_count", "1")
	assertMetadataValue(t, result.MetadataPairs, "table_count", "1")
	assertMetadataValue(t, result.MetadataPairs, "outline_count", "1")
}

func TestDeterministicIndexerFallsBackToPlainText(t *testing.T) {
	t.Parallel()

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       99,
		MimeType: "text/plain",
	}

	content := "MASTER SERVICE AGREEMENT\n\nThis agreement starts today.\n\nTermination"
	result, err := indexer.Index(context.Background(), doc, bytes.NewBufferString(content))
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if len(result.OutlineNodes) != 2 {
		t.Fatalf("expected 2 outline nodes, got %d", len(result.OutlineNodes))
	}
	if result.OutlineNodes[0].Title != "MASTER SERVICE AGREEMENT" {
		t.Fatalf("unexpected first outline title: %q", result.OutlineNodes[0].Title)
	}
	if result.OutlineNodes[1].Title != "Termination" {
		t.Fatalf("unexpected second outline title: %q", result.OutlineNodes[1].Title)
	}
	if len(result.ParagraphNodes) != 1 || result.ParagraphNodes[0].Content != "This agreement starts today." {
		t.Fatalf("unexpected paragraph nodes: %#v", result.ParagraphNodes)
	}
	if len(result.DocumentPages) != 1 || result.DocumentPages[0].PageNumber != 1 {
		t.Fatalf("unexpected document pages: %#v", result.DocumentPages)
	}

	assertMetadataValue(t, result.MetadataPairs, "paragraph_count", "1")
	assertMetadataValue(t, result.MetadataPairs, "table_count", "0")
	assertMetadataValue(t, result.MetadataPairs, "outline_count", "2")
}

func TestDeterministicIndexerAssignsSectionMetadataToParagraphs(t *testing.T) {
	t.Parallel()

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       100,
		MimeType: "text/plain",
	}

	content := "INTRODUCTION\n\nOpening paragraph.\n\nPAYMENT TERMS\n\nInvoice due in 15 days."
	result, err := indexer.Index(context.Background(), doc, bytes.NewBufferString(content))
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if len(result.ParagraphNodes) != 2 {
		t.Fatalf("expected 2 paragraph nodes, got %d", len(result.ParagraphNodes))
	}

	if got := result.ParagraphNodes[0].Metadata["section_title"]; got != "INTRODUCTION" {
		t.Fatalf("unexpected first section title: %#v", got)
	}
	if got := result.ParagraphNodes[1].Metadata["section_title"]; got != "PAYMENT TERMS" {
		t.Fatalf("unexpected second section title: %#v", got)
	}

	if got := result.OutlineNodes[1].Metadata["paragraph_start"]; got != 1 {
		t.Fatalf("unexpected outline paragraph_start: %#v", got)
	}
}

func assertMetadataValue(t *testing.T, pairs []models.MetadataPair, key, want string) {
	t.Helper()

	for _, pair := range pairs {
		if pair.Key == key {
			if pair.Value != want {
				t.Fatalf("metadata %q mismatch: got %q want %q", key, pair.Value, want)
			}
			return
		}
	}

	t.Fatalf("metadata %q not found", key)
}
