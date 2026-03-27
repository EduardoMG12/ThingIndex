package indexing

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"thing-index/internal/models"

	"github.com/ledongthuc/pdf"
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

func TestDeterministicIndexerParsesLineOrientedPlainTextHeadings(t *testing.T) {
	t.Parallel()

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       101,
		MimeType: "text/plain",
	}

	content := "MASTER SERVICE AGREEMENT\n\n1 Scope\nThe platform subscription covers onboarding support.\n\n2 Payment Terms\nInvoices must be paid within 15 days of receipt."
	result, err := indexer.Index(context.Background(), doc, bytes.NewBufferString(content))
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if len(result.OutlineNodes) != 3 {
		t.Fatalf("expected 3 outline nodes, got %#v", result.OutlineNodes)
	}
	if result.OutlineNodes[1].Title != "1 Scope" || result.OutlineNodes[2].Title != "2 Payment Terms" {
		t.Fatalf("unexpected outline titles: %#v", result.OutlineNodes)
	}
	if len(result.ParagraphNodes) != 2 {
		t.Fatalf("expected 2 paragraph nodes, got %#v", result.ParagraphNodes)
	}
	if result.ParagraphNodes[0].Content != "The platform subscription covers onboarding support." {
		t.Fatalf("unexpected first paragraph: %#v", result.ParagraphNodes[0])
	}
	if result.ParagraphNodes[1].Content != "Invoices must be paid within 15 days of receipt." {
		t.Fatalf("unexpected second paragraph: %#v", result.ParagraphNodes[1])
	}
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
	if got := result.OutlineNodes[1].Metadata["paragraph_end"]; got != 1 {
		t.Fatalf("unexpected outline paragraph_end: %#v", got)
	}
}

func TestDeterministicIndexerBuildsDeterministicSectionSummaries(t *testing.T) {
	t.Parallel()

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       102,
		MimeType: "text/html",
	}

	content := `<html><body><h1>DATA RETENTION POLICY</h1><h2>Retention Schedule</h2><p>Customer invoices are retained for 5 years.</p><table><tr><th>Artifact</th><th>Retention</th></tr><tr><td>Security incidents</td><td>7 years</td></tr></table></body></html>`
	result, err := indexer.Index(context.Background(), doc, bytes.NewBufferString(content))
	if err != nil {
		t.Fatalf("Index() returned error: %v", err)
	}

	if len(result.OutlineNodes) < 2 {
		t.Fatalf("expected outline nodes, got %#v", result.OutlineNodes)
	}
	if !strings.Contains(result.OutlineNodes[1].Summary, "Customer invoices are retained for 5 years") {
		t.Fatalf("expected paragraph-derived summary, got %#v", result.OutlineNodes[1])
	}
	if !strings.Contains(result.OutlineNodes[1].Summary, "Security incidents") {
		t.Fatalf("expected table-derived signal in summary, got %#v", result.OutlineNodes[1])
	}
	assertMetadataValue(t, result.MetadataPairs, "tree_summary_coverage_ratio", "0.50")
}

func TestOutlineIdentifiersRespectStorageLimits(t *testing.T) {
	t.Parallel()

	longTitle := strings.Repeat("Secao Muito Longa Com Texto Repetido ", 12)
	nodeID := buildOutlineNodeID(7, longTitle)
	slug := buildOutlineSlug(longTitle)
	nodePath := buildOutlineNodePath([]string{
		buildOutlineNodeID(1, longTitle),
		buildOutlineNodeID(2, longTitle),
		buildOutlineNodeID(3, longTitle),
		buildOutlineNodeID(4, longTitle),
		buildOutlineNodeID(5, longTitle),
		buildOutlineNodeID(6, longTitle),
		buildOutlineNodeID(7, longTitle),
		buildOutlineNodeID(8, longTitle),
		buildOutlineNodeID(9, longTitle),
		buildOutlineNodeID(10, longTitle),
		buildOutlineNodeID(11, longTitle),
		buildOutlineNodeID(12, longTitle),
	})

	if len(nodeID) > maxOutlineNodeIDLength {
		t.Fatalf("node id exceeded limit: got %d", len(nodeID))
	}
	if len(slug) > maxOutlineSlugLength {
		t.Fatalf("slug exceeded limit: got %d", len(slug))
	}
	if len(nodePath) > maxOutlineNodePathLength {
		t.Fatalf("node path exceeded limit: got %d", len(nodePath))
	}
}

func TestLineFromPDFTextsReconstructsBrokenWords(t *testing.T) {
	t.Parallel()

	line := lineFromPDFTexts([]pdf.Text{
		{S: "D", X: 10, W: 4, FontSize: 12},
		{S: "e", X: 14.2, W: 4, FontSize: 12},
		{S: "l", X: 18.1, W: 2, FontSize: 12},
		{S: "e", X: 20.3, W: 4, FontSize: 12},
		{S: "g", X: 24.4, W: 5, FontSize: 12},
		{S: "a", X: 29.1, W: 4, FontSize: 12},
		{S: "c", X: 33.0, W: 4, FontSize: 12},
		{S: "i", X: 37.1, W: 2, FontSize: 12},
		{S: "a", X: 39.4, W: 4, FontSize: 12},
		{S: "G", X: 48, W: 6, FontSize: 12},
		{S: "e", X: 54.1, W: 4, FontSize: 12},
		{S: "r", X: 58.1, W: 3, FontSize: 12},
		{S: "a", X: 61.2, W: 4, FontSize: 12},
		{S: "l", X: 65.3, W: 2, FontSize: 12},
	})

	if line != "Delegacia Geral" {
		t.Fatalf("unexpected reconstructed line: %q", line)
	}
}

func TestLineFromPDFTextsPreservesRealSpaces(t *testing.T) {
	t.Parallel()

	line := lineFromPDFTexts([]pdf.Text{
		{S: "4", X: 10, W: 5, FontSize: 12},
		{S: "-", X: 18, W: 3, FontSize: 12},
		{S: "Un", X: 26, W: 10, FontSize: 12},
		{S: "id", X: 36.5, W: 8, FontSize: 12},
		{S: "ad", X: 44.8, W: 8, FontSize: 12},
		{S: "e", X: 53.1, W: 4, FontSize: 12},
		{S: "s", X: 57.2, W: 4, FontSize: 12},
		{S: "P", X: 69, W: 6, FontSize: 12},
		{S: "o", X: 75.2, W: 5, FontSize: 12},
		{S: "lic", X: 80.1, W: 11, FontSize: 12},
		{S: "iais", X: 91.4, W: 13, FontSize: 12},
	})

	if line != "4 - Unidades Policiais" {
		t.Fatalf("unexpected reconstructed line: %q", line)
	}
}

func TestLooksLikePDFHeadingRejectsLowSignalAndFieldLikeBlocks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		text string
		want bool
	}{
		{text: "DECISÃO", want: true},
		{text: "CERTIDÃO", want: true},
		{text: "OFÍCIO REQUISITÓRIO", want: true},
		{text: "PROCESSAMENTO", want: true},
		{text: "P", want: false},
		{text: "4", want: false},
		{text: "395478", want: false},
		{text: "o T", want: false},
		{text: "25 33", want: false},
		{text: "Advogado(a):", want: false},
		{text: "Ó rgão:", want: false},
		{text: "sp1jefaz@tjsp.jus.br", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.text, func(t *testing.T) {
			t.Parallel()

			if got := looksLikePDFHeading(tc.text); got != tc.want {
				t.Fatalf("looksLikePDFHeading(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestCountPDFBlockPageOccurrencesCountsPagesOnlyOnce(t *testing.T) {
	t.Parallel()

	pageBlocks := [][]textBlock{
		{
			{Content: "TRIBUNAL DE JUSTIÇA DO ESTADO DE SÃO PAULO"},
			{Content: "TRIBUNAL DE JUSTIÇA DO ESTADO DE SÃO PAULO"},
			{Content: "DECISÃO"},
		},
		{
			{Content: "TRIBUNAL DE JUSTIÇA DO ESTADO DE SÃO PAULO"},
			{Content: "DECISÃO"},
		},
		{
			{Content: "TRIBUNAL DE JUSTIÇA DO ESTADO DE SÃO PAULO"},
			{Content: "CERTIDÃO"},
		},
	}

	occurrences := countPDFBlockPageOccurrences(pageBlocks)

	if got := occurrences["TRIBUNAL DE JUSTIÇA DO ESTADO DE SÃO PAULO"]; got != 3 {
		t.Fatalf("unexpected repeated heading count: got %d want 3", got)
	}
	if got := occurrences["DECISÃO"]; got != 2 {
		t.Fatalf("unexpected section count: got %d want 2", got)
	}
	if !shouldSuppressRepeatedPDFHeading("TRIBUNAL DE JUSTIÇA DO ESTADO DE SÃO PAULO", occurrences) {
		t.Fatalf("expected tribunal heading to be suppressed when repeated across pages")
	}
	if shouldSuppressRepeatedPDFHeading("DECISÃO", occurrences) {
		t.Fatalf("did not expect decision heading to be suppressed with only two page occurrences")
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
