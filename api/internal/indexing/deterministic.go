package indexing

import (
	"context"
	"fmt"
	"io"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type DeterministicIndexer struct {
	pdfTextExtractor pdfTextExtractor
	pdfOCRExtractor  pdfOCRExtractor
}

type outlineDraft struct {
	Level          int
	Title          string
	PageNumber     *int
	ParagraphStart int
	SectionKey     string
}

type textBlock struct {
	Content    string
	PageNumber *int
	SectionKey string
	Source     string
	Box        *textBox
	// LineCount guarda quantas linhas o extrator agrupou neste bloco; blocos de
	// uma linha podem ser remontados com vizinhos da mesma linha visual.
	LineCount int
}

// textBox ancora um bloco na coordenada original da pagina, em pontos PDF com
// origem no canto superior esquerdo. PageWidth/PageHeight acompanham a caixa
// para que o consumidor normalize sem precisar reabrir o documento.
type textBox struct {
	XMin       float64 `json:"x_min"`
	YMin       float64 `json:"y_min"`
	XMax       float64 `json:"x_max"`
	YMax       float64 `json:"y_max"`
	PageWidth  float64 `json:"page_width"`
	PageHeight float64 `json:"page_height"`
}

type pdfTextExtraction struct {
	PageCount  int
	PageBlocks map[int][]textBlock
}

type pdfTextExtractor interface {
	Extract(data []byte) (pdfTextExtraction, error)
}

type pdfOCRExtractor interface {
	Name() string
	Extract(ctx context.Context, data []byte, pageNumbers []int) (map[int]string, error)
}

type nativePDFTextExtractor struct{}

type commandPDFTextExtractor struct {
	binPath string
}

type commandPDFOCRExtractor struct {
	rendererPath  string
	tesseractPath string
}

type pdfOCRFallbackStats struct {
	Needed      bool
	Attempted   bool
	Applied     bool
	Engine      string
	Reason      string
	PageNumbers []int
}

func NewDeterministicIndexer() *DeterministicIndexer {
	pdfExtractor := newCommandPDFTextExtractor()
	if pdfExtractor == nil {
		pdfExtractor = nativePDFTextExtractor{}
	}

	return &DeterministicIndexer{
		pdfTextExtractor: pdfExtractor,
		pdfOCRExtractor:  newCommandPDFOCRExtractor(),
	}
}

func newDeterministicIndexerWithExtractors(
	textExtractor pdfTextExtractor,
	ocrExtractor pdfOCRExtractor,
) *DeterministicIndexer {
	if textExtractor == nil {
		textExtractor = nativePDFTextExtractor{}
	}

	return &DeterministicIndexer{
		pdfTextExtractor: textExtractor,
		pdfOCRExtractor:  ocrExtractor,
	}
}

func (i *DeterministicIndexer) Index(ctx context.Context, doc *models.Document, file io.Reader) (domain.IndexingResult, error) {
	select {
	case <-ctx.Done():
		return domain.IndexingResult{}, ctx.Err()
	default:
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: read document: %v", domain.ErrDocumentIndexing, err)
	}

	switch doc.MimeType {
	case "text/html":
		return i.indexHTML(doc.ID, data)
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return i.indexSpreadsheet(doc.ID, data)
	case "application/pdf":
		return i.indexPDF(ctx, doc.ID, data)
	default:
		return i.indexPlainText(doc.ID, string(data)), nil
	}
}

func (i *DeterministicIndexer) indexPDF(ctx context.Context, documentID uint, data []byte) (domain.IndexingResult, error) {
	textExtractor := i.pdfTextExtractor
	if textExtractor == nil {
		textExtractor = nativePDFTextExtractor{}
	}

	extraction, err := textExtractor.Extract(data)
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse pdf: %v", domain.ErrDocumentParsingFailed, err)
	}

	pageBlocks, ocrStats := i.applyPDFOCRFallback(ctx, data, extraction.PageCount, extraction.PageBlocks)
	headings, paragraphs := buildPDFStructureFromPageBlocks(pageBlocks, extraction.PageCount)
	result := buildIndexingResult(documentID, headings, paragraphs, nil)
	appendOCRFallbackMetadata(&result, documentID, ocrStats)
	return result, nil
}

func buildPDFStructureFromPageBlocks(pageBlocks map[int][]textBlock, pageCount int) ([]outlineDraft, []textBlock) {
	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	currentSectionKey := ""

	pageOccurrences := countPDFBlockPageOccurrences(pageBlocks, pageCount)
	emittedRepeatedPDFHeadings := make(map[string]struct{})
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		for _, block := range pageBlocks[pageNumber] {
			content := strings.TrimSpace(block.Content)
			if content == "" || isLowSignalPDFText(content) {
				continue
			}

			if looksLikePDFHeading(content) {
				if shouldSuppressRepeatedPDFHeading(content, pageOccurrences) {
					if _, seen := emittedRepeatedPDFHeadings[content]; seen {
						continue
					}
					emittedRepeatedPDFHeadings[content] = struct{}{}
				}

				currentSectionKey = slugify(content)
				headings = append(headings, outlineDraft{
					Level:          inferHeadingLevel(content),
					Title:          content,
					PageNumber:     block.PageNumber,
					ParagraphStart: len(paragraphs),
					SectionKey:     currentSectionKey,
				})
				continue
			}

			block.SectionKey = currentSectionKey
			block.Content = content
			paragraphs = append(paragraphs, block)
		}
	}

	if len(headings) == 0 && len(paragraphs) > 0 {
		headings = append(headings, outlineDraft{
			Level:          1,
			Title:          firstSentence(paragraphs[0].Content),
			PageNumber:     paragraphs[0].PageNumber,
			ParagraphStart: 0,
			SectionKey:     slugify(firstSentence(paragraphs[0].Content)),
		})
		if headings[0].SectionKey != "" && paragraphs[0].SectionKey == "" {
			paragraphs[0].SectionKey = headings[0].SectionKey
		}
	}

	return headings, paragraphs
}

func (i *DeterministicIndexer) indexPlainText(documentID uint, raw string) domain.IndexingResult {
	headings, paragraphs := parsePlainTextStructure(raw)

	if len(headings) == 0 && len(paragraphs) > 0 {
		title := firstSentence(paragraphs[0].Content)
		headings = append(headings, outlineDraft{
			Level:          1,
			Title:          title,
			PageNumber:     intPtr(1),
			ParagraphStart: 0,
			SectionKey:     slugify(title),
		})
		paragraphs[0].SectionKey = headings[0].SectionKey
	}

	return buildIndexingResult(documentID, headings, paragraphs, nil)
}

func parsePlainTextStructure(raw string) ([]outlineDraft, []textBlock) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0, len(lines))
	currentSectionKey := ""
	paragraphLines := make([]string, 0, 4)
	pageNumber := 1

	flushParagraph := func() {
		if len(paragraphLines) == 0 {
			return
		}
		content := normalizeText(strings.Join(paragraphLines, " "))
		paragraphLines = paragraphLines[:0]
		if content == "" {
			return
		}
		paragraphs = append(paragraphs, textBlock{
			Content:    content,
			PageNumber: intPtr(pageNumber),
			SectionKey: currentSectionKey,
		})
	}

	for _, rawLine := range lines {
		line := normalizeText(rawLine)
		if line == "" {
			flushParagraph()
			continue
		}

		if looksLikeHeading(line) {
			flushParagraph()
			currentSectionKey = slugify(line)
			headings = append(headings, outlineDraft{
				Level:          inferHeadingLevel(line),
				Title:          line,
				PageNumber:     intPtr(pageNumber),
				ParagraphStart: len(paragraphs),
				SectionKey:     currentSectionKey,
			})
			continue
		}

		paragraphLines = append(paragraphLines, line)
	}

	flushParagraph()
	return headings, paragraphs
}

func splitBlocks(text string) []string {
	if text == "" {
		return nil
	}

	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	rawBlocks := strings.Split(normalized, "\n\n")
	blocks := make([]string, 0, len(rawBlocks))
	for _, block := range rawBlocks {
		cleaned := normalizeText(block)
		if cleaned != "" {
			blocks = append(blocks, cleaned)
		}
	}
	return blocks
}
