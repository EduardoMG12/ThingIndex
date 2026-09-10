package indexing

import (
	"context"
	"fmt"
	"io"
	"regexp"
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
	// Lines sao as linhas visuais do bloco com a caixa de cada uma, para o
	// grifo apontar a linha citada em vez do paragrafo inteiro.
	Lines []textLine
}

type textLine struct {
	Text string
	Box  textBox
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
	// PageTables sao as tabelas detectadas pelas caixas de palavra, por pagina.
	PageTables map[int][]models.TableSection
}

type pdfTextExtractor interface {
	Extract(data []byte) (pdfTextExtraction, error)
}

// pdfOCRExtractor devolve blocos, nao texto puro, porque um provider capaz de
// layout (como o Unlimited-OCR) sabe a caixa de cada regiao. Providers que so
// entregam texto devolvem blocos com Box nil, e o grifo posicional
// simplesmente nao existe para aquelas paginas.
type pdfOCRExtractor interface {
	Name() string
	Extract(ctx context.Context, data []byte, pageNumbers []int) (map[int][]textBlock, error)
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
	// AnchoredBlocks conta quantos blocos vindos do OCR trouxeram caixa. Com
	// tesseract e sempre zero; com um provider de layout mede a cobertura de
	// ancoragem posicional em documento digitalizado.
	AnchoredBlocks int
}

func NewDeterministicIndexer() *DeterministicIndexer {
	pdfExtractor := newCommandPDFTextExtractor()
	if pdfExtractor == nil {
		pdfExtractor = nativePDFTextExtractor{}
	}

	return &DeterministicIndexer{
		pdfTextExtractor: pdfExtractor,
		// newPDFOCRExtractor, e nao newCommandPDFOCRExtractor: a primeira tenta
		// o provider remoto (THINGINDEX_OCR_ENDPOINT) e cai para o tesseract
		// local quando ele nao esta configurado.
		//
		// Chamar o construtor local direto tornava o provider remoto codigo
		// morto: configurar o endpoint nao mudava nada, o indice sempre gravava
		// ocr_engine='local_tesseract' e o servico `ocr` do compose ficava
		// inalcancavel -- sem erro e sem log, so a funcionalidade ausente.
		pdfOCRExtractor: newPDFOCRExtractor(),
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
	headings, paragraphs := buildPDFStructure(pageBlocks, extraction.PageTables, extraction.PageCount)
	result := buildIndexingResult(documentID, headings, paragraphs, flattenPageTables(documentID, extraction.PageTables, extraction.PageCount))
	appendOCRFallbackMetadata(&result, documentID, ocrStats)
	return result, nil
}

func buildPDFStructureFromPageBlocks(pageBlocks map[int][]textBlock, pageCount int) ([]outlineDraft, []textBlock) {
	return buildPDFStructure(pageBlocks, nil, pageCount)
}

// buildPDFStructure monta titulos e paragrafos; bloco dentro de uma tabela da
// pagina nunca vira titulo (continua como paragrafo, para o texto da pagina).
func buildPDFStructure(pageBlocks map[int][]textBlock, pageTables map[int][]models.TableSection, pageCount int) ([]outlineDraft, []textBlock) {
	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	currentSectionKey := ""

	pageOccurrences := countPDFBlockPageOccurrences(pageBlocks, pageCount)
	emittedRepeatedPDFHeadings := make(map[string]struct{})
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		// Pagina de sumario: as linhas "1.1.1 Denominacao do Curso 9" parecem
		// titulos e duplicavam cada secao do PPC (124 de 861). Nela, nada vira
		// titulo; o texto fica como paragrafo.
		tocPage := isPDFTableOfContentsPage(pageBlocks[pageNumber])
		tableBoxes := pdfTableBoxes(pageTables[pageNumber])
		for _, block := range pageBlocks[pageNumber] {
			content := strings.TrimSpace(block.Content)
			if content == "" || isLowSignalPDFText(content) {
				continue
			}
			if !tocPage && !blockInsideTable(block, tableBoxes) && looksLikePDFHeading(content) {
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

// tocEntryPattern casa "1.2.3 Titulo 45" e "APENDICE A - REGULAMENTO 176":
// texto que termina com um numero de pagina solto.
var tocEntryPattern = regexp.MustCompile(`\S\s+\d{1,3}$`)

// PDFTableOfContentsMinEntries e quantas linhas terminadas em numero de
// pagina uma pagina precisa ter para ser tratada como sumario.
const PDFTableOfContentsMinEntries = 5

// isPDFTableOfContentsPage diz se a pagina e um sumario: cinco ou mais
// blocos que parecem titulo e terminam em numero de pagina.
func isPDFTableOfContentsPage(blocks []textBlock) bool {
	entries := 0
	for _, block := range blocks {
		content := strings.TrimSpace(block.Content)
		if content == "" || !tocEntryPattern.MatchString(content) {
			continue
		}
		if looksLikeHeading(content) {
			entries++
			if entries >= PDFTableOfContentsMinEntries {
				return true
			}
		}
	}
	return false
}
