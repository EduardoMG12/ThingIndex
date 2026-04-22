package indexing

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
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

const (
	maxOutlineNodeIDLength   = 128
	maxOutlineSlugLength     = 255
	maxOutlineNodePathLength = 1024
)

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

func (i *DeterministicIndexer) indexHTML(documentID uint, data []byte) (domain.IndexingResult, error) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse html: %v", domain.ErrDocumentParsingFailed, err)
	}

	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	tables := make([]models.TableSection, 0)
	pageNumber := 1
	currentSectionKey := ""
	currentSectionTitle := ""

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			switch node.Data {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level, _ := strconv.Atoi(strings.TrimPrefix(node.Data, "h"))
				title := normalizeText(extractHTMLText(node))
				if title != "" {
					currentSectionTitle = title
					currentSectionKey = slugify(title)
					headings = append(headings, outlineDraft{
						Level:          level,
						Title:          title,
						ParagraphStart: len(paragraphs),
						SectionKey:     currentSectionKey,
					})
				}
			case "p", "li":
				text := normalizeText(extractHTMLText(node))
				if text != "" {
					paragraphs = append(paragraphs, textBlock{
						Content:    text,
						PageNumber: &pageNumber,
						SectionKey: currentSectionKey,
					})
				}
			case "table":
				section := extractHTMLTable(documentID, len(tables), node)
				if len(section.Headers) > 0 || len(section.Rows) > 0 {
					if currentSectionTitle != "" {
						section.Title = currentSectionTitle
						section.Metadata["section_key"] = currentSectionKey
					}
					tables = append(tables, section)
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)

	return buildIndexingResult(documentID, headings, paragraphs, tables), nil
}

func (i *DeterministicIndexer) indexSpreadsheet(documentID uint, data []byte) (domain.IndexingResult, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse spreadsheet: %v", domain.ErrDocumentParsingFailed, err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	tables := make([]models.TableSection, 0)

	for sheetIndex, sheetName := range workbook.GetSheetList() {
		rows, err := workbook.GetRows(sheetName)
		if err != nil {
			return domain.IndexingResult{}, fmt.Errorf("%w: read spreadsheet rows: %v", domain.ErrDocumentParsingFailed, err)
		}

		sectionKey := slugify(sheetName)
		headings = append(headings, outlineDraft{
			Level:          1,
			Title:          sheetName,
			ParagraphStart: len(paragraphs),
			SectionKey:     sectionKey,
		})
		paragraphs = append(paragraphs, textBlock{
			Content:    fmt.Sprintf("Planilha %s com %d linhas.", sheetName, len(rows)),
			PageNumber: intPtr(sheetIndex + 1),
			SectionKey: sectionKey,
		})

		section := models.TableSection{
			DocumentID:    documentID,
			SequenceOrder: sheetIndex,
			Title:         sheetName,
			Headers:       []string{},
			Rows:          [][]string{},
			Metadata: map[string]any{
				"sheet_name":  sheetName,
				"row_count":   len(rows),
				"section_key": sectionKey,
			},
		}

		if len(rows) > 0 {
			section.Headers = normalizeCells(rows[0])
		}
		if len(rows) > 1 {
			for _, row := range rows[1:] {
				section.Rows = append(section.Rows, normalizeCells(row))
			}
		}

		tables = append(tables, section)
	}

	return buildIndexingResult(documentID, headings, paragraphs, tables), nil
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

func newCommandPDFTextExtractor() pdfTextExtractor {
	binPath, err := exec.LookPath("pdftotext")
	if err != nil {
		return nil
	}
	return &commandPDFTextExtractor{binPath: binPath}
}

func (e *commandPDFTextExtractor) Name() string {
	return "pdftotext"
}

func (e *commandPDFTextExtractor) Extract(data []byte) (pdfTextExtraction, error) {
	if e == nil {
		return pdfTextExtraction{}, fmt.Errorf("pdftotext not available")
	}

	tempDir, err := os.MkdirTemp("", "thingindex-pdf-extract-*")
	if err != nil {
		return pdfTextExtraction{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	pdfPath := filepath.Join(tempDir, "document.pdf")
	txtPath := filepath.Join(tempDir, "document.txt")

	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return pdfTextExtraction{}, fmt.Errorf("write temporary pdf: %w", err)
	}

	cmd := exec.Command(e.binPath, "-layout", "-enc", "UTF-8", pdfPath, txtPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return pdfTextExtraction{}, fmt.Errorf("pdftotext failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	rawText, err := os.ReadFile(txtPath)
	if err != nil {
		return pdfTextExtraction{}, fmt.Errorf("read extracted text: %w", err)
	}

	text := string(rawText)
	if !isPlausibleText(text) {
		return pdfTextExtraction{}, fmt.Errorf("pdftotext returned implausible text (garbage or empty)")
	}

	extraction := pdfTextExtraction{
		PageCount:  countPagesInExtractedText(text),
		PageBlocks: make(map[int][]textBlock),
	}

	pageSeparator := regexp.MustCompile(`(?m)^-+\s*$`)
	pages := pageSeparator.Split(text, -1)
	currentPage := 1

	for _, pageText := range pages {
		pageText = strings.TrimSpace(pageText)
		if pageText == "" {
			continue
		}

		lines := strings.Split(pageText, "\n")
		var blocks []textBlock
		var currentBlock strings.Builder

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				if currentBlock.Len() > 0 {
					blocks = append(blocks, textBlock{
						Content:    normalizeText(currentBlock.String()),
						PageNumber: &currentPage,
						Source:     "pdftotext",
					})
					currentBlock.Reset()
				}
				continue
			}
			currentBlock.WriteString(trimmed)
			currentBlock.WriteString("\n")
		}

		if currentBlock.Len() > 0 {
			blocks = append(blocks, textBlock{
				Content:    normalizeText(currentBlock.String()),
				PageNumber: &currentPage,
				Source:     "pdftotext",
			})
		}

		if len(blocks) > 0 {
			extraction.PageBlocks[currentPage] = blocks
		}
		currentPage++
	}

	return extraction, nil
}

func isPlausibleText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 10 {
		return false
	}

	letters := 0
	garbage := 0
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			letters++
		}
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			garbage++
		}
	}

	total := letters + garbage
	if total == 0 {
		return false
	}

	garbageRatio := float64(garbage) / float64(total)
	return garbageRatio < 0.1
}

func countPagesInExtractedText(text string) int {
	pageSeparator := regexp.MustCompile(`(?m)^-+\s*$`)
	return len(pageSeparator.Split(text, -1))
}

func (nativePDFTextExtractor) Extract(data []byte) (pdfTextExtraction, error) {
	reader := bytes.NewReader(data)
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return pdfTextExtraction{}, err
	}

	extraction := pdfTextExtraction{
		PageCount:  pdfReader.NumPage(),
		PageBlocks: make(map[int][]textBlock, pdfReader.NumPage()),
	}

	for pageIndex := 1; pageIndex <= pdfReader.NumPage(); pageIndex++ {
		page := pdfReader.Page(pageIndex)
		if page.V.IsNull() || page.V.Key("Contents").Kind() == pdf.Null {
			continue
		}

		pageNumber := pageIndex
		blocks := blocksFromPDFPage(page, &pageNumber)
		if len(blocks) > 0 {
			extraction.PageBlocks[pageNumber] = blocks
		}
	}

	return extraction, nil
}

func newCommandPDFOCRExtractor() pdfOCRExtractor {
	rendererPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil
	}
	tesseractPath, err := exec.LookPath("tesseract")
	if err != nil {
		return nil
	}

	return &commandPDFOCRExtractor{
		rendererPath:  rendererPath,
		tesseractPath: tesseractPath,
	}
}

func (e *commandPDFOCRExtractor) Name() string {
	return "local_tesseract"
}

func (e *commandPDFOCRExtractor) Extract(
	ctx context.Context,
	data []byte,
	pageNumbers []int,
) (map[int]string, error) {
	if e == nil || len(pageNumbers) == 0 {
		return nil, nil
	}

	tempDir, err := os.MkdirTemp("", "thingindex-pdf-ocr-*")
	if err != nil {
		return nil, fmt.Errorf("create ocr temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	pdfPath := filepath.Join(tempDir, "document.pdf")
	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("write temporary pdf: %w", err)
	}

	results := make(map[int]string, len(pageNumbers))
	for _, pageNumber := range pageNumbers {
		if pageNumber <= 0 {
			continue
		}

		imageBase := filepath.Join(tempDir, fmt.Sprintf("page-%d", pageNumber))
		renderCmd := exec.CommandContext(
			ctx,
			e.rendererPath,
			"-f", strconv.Itoa(pageNumber),
			"-l", strconv.Itoa(pageNumber),
			"-singlefile",
			"-png",
			pdfPath,
			imageBase,
		)
		if output, err := renderCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("render pdf page %d for ocr: %w: %s", pageNumber, err, strings.TrimSpace(string(output)))
		}

		imagePath := imageBase + ".png"
		ocrArgs := []string{imagePath, "stdout", "--psm", "1"}
		if langs := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_LANGS")); langs != "" {
			ocrArgs = append(ocrArgs, "-l", langs)
		}

		ocrCmd := exec.CommandContext(ctx, e.tesseractPath, ocrArgs...)
		output, err := ocrCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("ocr pdf page %d: %w: %s", pageNumber, err, strings.TrimSpace(string(output)))
		}

		text := normalizeText(string(output))
		if text != "" {
			results[pageNumber] = text
		}
	}

	return results, nil
}

func (i *DeterministicIndexer) applyPDFOCRFallback(
	ctx context.Context,
	data []byte,
	pageCount int,
	pageBlocks map[int][]textBlock,
) (map[int][]textBlock, pdfOCRFallbackStats) {
	pagesNeedingOCR, reason := pagesNeedingPDFOCRFallback(pageCount, pageBlocks)
	stats := pdfOCRFallbackStats{
		Needed:      len(pagesNeedingOCR) > 0,
		Reason:      reason,
		PageNumbers: append([]int(nil), pagesNeedingOCR...),
	}
	if len(pagesNeedingOCR) == 0 {
		return clonePDFPageBlocks(pageBlocks), stats
	}

	merged := clonePDFPageBlocks(pageBlocks)
	if i.pdfOCRExtractor == nil {
		stats.Reason = strings.TrimSpace(stats.Reason + "_ocr_unavailable")
		return merged, stats
	}

	stats.Attempted = true
	stats.Engine = i.pdfOCRExtractor.Name()
	ocrTextByPage, err := i.pdfOCRExtractor.Extract(ctx, data, pagesNeedingOCR)
	if err != nil {
		stats.Reason = strings.TrimSpace(stats.Reason + "_ocr_failed")
		return merged, stats
	}

	appliedPages := make([]int, 0, len(pagesNeedingOCR))
	for _, pageNumber := range pagesNeedingOCR {
		text := strings.TrimSpace(ocrTextByPage[pageNumber])
		if text == "" {
			continue
		}

		blocks := blocksFromOCRPageText(text, pageNumber)
		if len(blocks) == 0 {
			continue
		}
		merged[pageNumber] = blocks
		appliedPages = append(appliedPages, pageNumber)
	}

	stats.Applied = len(appliedPages) > 0
	stats.PageNumbers = appliedPages
	if !stats.Applied {
		stats.Reason = strings.TrimSpace(reason + "_ocr_empty")
	}

	return merged, stats
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

func buildIndexingResult(documentID uint, headings []outlineDraft, paragraphs []textBlock, tables []models.TableSection) domain.IndexingResult {
	paragraphNodes := make([]models.ParagraphNode, 0, len(paragraphs))
	for idx, paragraph := range paragraphs {
		metadata := map[string]any{}
		if paragraph.SectionKey != "" {
			metadata["section_key"] = paragraph.SectionKey
			metadata["section_title"] = sectionTitleForKey(headings, paragraph.SectionKey)
		}
		if strings.TrimSpace(paragraph.Source) != "" {
			metadata["source"] = paragraph.Source
		}

		paragraphNodes = append(paragraphNodes, models.ParagraphNode{
			DocumentID:    documentID,
			SequenceOrder: idx,
			PageNumber:    paragraph.PageNumber,
			Content:       paragraph.Content,
			Metadata:      metadata,
		})
	}

	documentPages := buildDocumentPages(documentID, paragraphs)
	outlineNodes := buildOutlineNodes(documentID, headings, paragraphs)
	outlineNodes = populateDeterministicOutlineSummaries(outlineNodes, paragraphs, tables)

	result := domain.IndexingResult{
		DocumentPages:  documentPages,
		ParagraphNodes: paragraphNodes,
		TableSections:  tables,
		OutlineNodes:   outlineNodes,
	}

	treeStats := treeVerificationStats{
		Builder:            "deterministic",
		TotalNodes:         len(result.OutlineNodes),
		TotalPages:         len(result.DocumentPages),
		CoveredPages:       len(result.DocumentPages),
		CoverageRatio:      1,
		MaxDepth:           outlineMaxDepth(result.OutlineNodes),
		VerificationPassed: len(result.OutlineNodes) > 0 || len(result.DocumentPages) == 0,
	}
	if len(result.DocumentPages) > 0 {
		result.OutlineNodes, treeStats = verifyAndScoreTree(result, result.OutlineNodes, "deterministic")
		if shouldFallback, reason := shouldUseConservativeTreeFallback(treeStats); shouldFallback {
			result, treeStats = applyConservativeTreeFallback(result, reason)
		}
	}

	totalPages := treeStats.TotalPages
	coveredPages := treeStats.CoveredPages
	coverageRatio := treeStats.CoverageRatio
	maxDepth := treeStats.MaxDepth
	populateOutlineRetrievalText(&result)
	summaryCoverageRatio := outlineSummaryCoverageRatio(result.OutlineNodes)
	verificationPassed := treeStats.VerificationPassed
	documentSummary := buildDeterministicDocumentSummary(result.OutlineNodes, paragraphs)
	fallbackApplied := treeStats.Builder == conservativeTreeFallbackBuilder

	metadataPairs := make([]models.MetadataPair, 0, 16)
	appendMetadataPair := func(key, value string) {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return
		}
		metadataPairs = append(metadataPairs, models.MetadataPair{
			DocumentID:    documentID,
			SequenceOrder: len(metadataPairs),
			Key:           key,
			Value:         value,
		})
	}

	appendMetadataPair("paragraph_count", strconv.Itoa(len(paragraphNodes)))
	appendMetadataPair("table_count", strconv.Itoa(len(tables)))
	appendMetadataPair("outline_count", strconv.Itoa(len(result.OutlineNodes)))
	appendMetadataPair("tree_builder", treeStats.Builder)
	appendMetadataPair("tree_total_nodes", strconv.Itoa(len(result.OutlineNodes)))
	appendMetadataPair("tree_total_pages", strconv.Itoa(totalPages))
	appendMetadataPair("tree_covered_pages", strconv.Itoa(coveredPages))
	appendMetadataPair("tree_coverage_ratio", strconv.FormatFloat(coverageRatio, 'f', 2, 64))
	appendMetadataPair("tree_max_depth", strconv.Itoa(maxDepth))
	appendMetadataPair("tree_low_confidence_nodes", strconv.Itoa(treeStats.LowConfidenceNodes))
	appendMetadataPair("tree_oversized_nodes", strconv.Itoa(treeStats.OversizedNodes))
	appendMetadataPair("tree_summary_coverage_ratio", strconv.FormatFloat(summaryCoverageRatio, 'f', 2, 64))
	appendMetadataPair("tree_verification_passed", strconv.FormatBool(verificationPassed))
	appendMetadataPair("tree_fallback_applied", strconv.FormatBool(fallbackApplied))
	if strings.TrimSpace(treeStats.FallbackReason) != "" {
		appendMetadataPair("tree_fallback_reason", treeStats.FallbackReason)
	}
	appendMetadataPair("document_summary", documentSummary)

	allTextParts := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph.Content) != "" {
			allTextParts = append(allTextParts, paragraph.Content)
		}
	}
	allText := strings.Join(allTextParts, " ")
	appendMetadataPair("word_count", strconv.Itoa(len(strings.Fields(allText))))
	appendMetadataPair("character_count", strconv.Itoa(len([]rune(allText))))

	result.MetadataPairs = metadataPairs
	return result
}

func buildDocumentPages(documentID uint, paragraphs []textBlock) []models.DocumentPage {
	pageBuckets := make(map[int][]string)
	pageSectionKeys := make(map[int]map[string]struct{})
	pageSources := make(map[int]map[string]struct{})

	for _, paragraph := range paragraphs {
		pageNumber := 1
		if paragraph.PageNumber != nil && *paragraph.PageNumber > 0 {
			pageNumber = *paragraph.PageNumber
		}

		content := strings.TrimSpace(paragraph.Content)
		if content != "" {
			pageBuckets[pageNumber] = append(pageBuckets[pageNumber], content)
		}
		if paragraph.SectionKey != "" {
			if pageSectionKeys[pageNumber] == nil {
				pageSectionKeys[pageNumber] = map[string]struct{}{}
			}
			pageSectionKeys[pageNumber][paragraph.SectionKey] = struct{}{}
		}
		if strings.TrimSpace(paragraph.Source) != "" {
			if pageSources[pageNumber] == nil {
				pageSources[pageNumber] = map[string]struct{}{}
			}
			pageSources[pageNumber][paragraph.Source] = struct{}{}
		}
	}

	if len(pageBuckets) == 0 {
		return nil
	}

	pageNumbers := make([]int, 0, len(pageBuckets))
	for pageNumber := range pageBuckets {
		pageNumbers = append(pageNumbers, pageNumber)
	}
	sortInts(pageNumbers)

	pages := make([]models.DocumentPage, 0, len(pageNumbers))
	for _, pageNumber := range pageNumbers {
		content := strings.Join(pageBuckets[pageNumber], "\n\n")
		metadata := map[string]any{
			"section_keys": sortedMapKeys(pageSectionKeys[pageNumber]),
		}
		sources := sortedMapKeys(pageSources[pageNumber])
		switch len(sources) {
		case 1:
			metadata["source"] = sources[0]
		case 2, 3, 4:
			metadata["source"] = "mixed"
		}
		pages = append(pages, models.DocumentPage{
			DocumentID:    documentID,
			PageNumber:    pageNumber,
			Content:       content,
			TokenEstimate: estimateTextTokens(content),
			Metadata:      metadata,
		})
	}

	return pages
}

func buildOutlineNodes(documentID uint, headings []outlineDraft, paragraphs []textBlock) []models.DocumentOutlineNode {
	nodes := make([]models.DocumentOutlineNode, 0, len(headings))
	if len(headings) == 0 {
		return nodes
	}

	type levelEntry struct {
		level     int
		nodeID    string
		sequence  int
		startPage *int
		endPage   *int
		title     string
	}

	stack := make([]levelEntry, 0, len(headings))
	for idx, heading := range headings {
		title := strings.TrimSpace(heading.Title)
		if title == "" {
			continue
		}

		level := max(1, heading.Level)
		nodeID := buildOutlineNodeID(len(nodes), title)
		slug := buildOutlineSlug(title)
		startPage, endPage := headingPageRange(idx, headings, paragraphs)
		paragraphEnd := len(paragraphs) - 1
		if idx+1 < len(headings) {
			paragraphEnd = headings[idx+1].ParagraphStart - 1
		}

		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}

		var parentNodeID string
		pathParts := make([]string, 0, len(stack)+1)
		for _, item := range stack {
			pathParts = append(pathParts, item.nodeID)
		}
		if len(stack) > 0 {
			parentNodeID = stack[len(stack)-1].nodeID
		}
		pathParts = append(pathParts, nodeID)

		metadata := map[string]any{
			"paragraph_start": heading.ParagraphStart,
			"paragraph_end":   paragraphEnd,
			"section_key":     heading.SectionKey,
		}
		if heading.PageNumber != nil && *heading.PageNumber > 0 {
			metadata["page_number"] = *heading.PageNumber
		}
		if startPage != nil {
			metadata["start_page"] = *startPage
		}
		if endPage != nil {
			metadata["end_page"] = *endPage
		}
		if parentNodeID != "" {
			metadata["parent_node_id"] = parentNodeID
		}

		nodes = append(nodes, models.DocumentOutlineNode{
			DocumentID:    documentID,
			SequenceOrder: len(nodes),
			NodeID:        nodeID,
			Level:         level,
			Title:         title,
			Slug:          slug,
			Summary:       "",
			StartPage:     startPage,
			EndPage:       endPage,
			NodePath:      buildOutlineNodePath(pathParts),
			Metadata:      metadata,
		})

		stack = append(stack, levelEntry{
			level:     level,
			nodeID:    nodeID,
			sequence:  len(nodes) - 1,
			startPage: startPage,
			endPage:   endPage,
			title:     title,
		})
	}

	return nodes
}

func headingPageRange(index int, headings []outlineDraft, paragraphs []textBlock) (*int, *int) {
	if index < 0 || index >= len(headings) {
		return nil, nil
	}

	startParagraph := max(0, headings[index].ParagraphStart)
	endParagraph := len(paragraphs) - 1
	if index+1 < len(headings) {
		endParagraph = headings[index+1].ParagraphStart - 1
	}

	var startPage *int
	var endPage *int
	if headings[index].PageNumber != nil {
		start := *headings[index].PageNumber
		startPage = &start
	}

	for paragraphIndex := startParagraph; paragraphIndex <= endParagraph && paragraphIndex < len(paragraphs); paragraphIndex++ {
		pageNumber := 1
		if paragraphs[paragraphIndex].PageNumber != nil {
			pageNumber = *paragraphs[paragraphIndex].PageNumber
		}

		if startPage == nil {
			start := pageNumber
			startPage = &start
		}
		end := pageNumber
		endPage = &end
	}

	if endPage == nil && startPage != nil {
		end := *startPage
		endPage = &end
	}

	return startPage, endPage
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

func populateDeterministicOutlineSummaries(
	nodes []models.DocumentOutlineNode,
	paragraphs []textBlock,
	tables []models.TableSection,
) []models.DocumentOutlineNode {
	if len(nodes) == 0 {
		return nodes
	}

	updated := make([]models.DocumentOutlineNode, 0, len(nodes))
	for _, node := range nodes {
		summary := buildDeterministicSectionSummary(node, paragraphs, tables)
		if node.Metadata == nil {
			node.Metadata = map[string]any{}
		}
		if summary != "" {
			node.Metadata["summary_source"] = "deterministic"
		}
		node.Summary = summary
		updated = append(updated, node)
	}

	return updated
}

func buildDeterministicSectionSummary(
	node models.DocumentOutlineNode,
	paragraphs []textBlock,
	tables []models.TableSection,
) string {
	fragments := make([]string, 0, 3)

	start, end, ok := outlineParagraphRange(node)
	if ok {
		for idx := start; idx <= end && idx < len(paragraphs); idx++ {
			sentence := firstSentence(paragraphs[idx].Content)
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			fragments = appendUniqueSummaryFragment(fragments, sentence)
			if len(fragments) == 2 {
				break
			}
		}
	}

	for _, table := range matchingOutlineTables(node, tables) {
		tableSummary := summarizeTableSection(table)
		if tableSummary == "" {
			continue
		}
		fragments = appendUniqueSummaryFragment(fragments, tableSummary)
		if len(fragments) == 3 {
			break
		}
	}

	return clampSummary(strings.Join(fragments, ". "), 320)
}

func buildDeterministicDocumentSummary(nodes []models.DocumentOutlineNode, paragraphs []textBlock) string {
	fragments := make([]string, 0, 3)
	for _, node := range nodes {
		summary := strings.TrimSpace(node.Summary)
		if summary == "" {
			continue
		}
		fragments = appendUniqueSummaryFragment(fragments, summary)
		if len(fragments) == 2 {
			break
		}
	}

	if len(fragments) == 0 && len(paragraphs) > 0 {
		fragments = appendUniqueSummaryFragment(fragments, firstSentence(paragraphs[0].Content))
	}

	return clampSummary(strings.Join(fragments, ". "), 320)
}

func populateOutlineRetrievalText(result *domain.IndexingResult) {
	if result == nil || len(result.OutlineNodes) == 0 {
		return
	}

	childParagraphRanges := buildOutlineChildParagraphRanges(result.OutlineNodes)
	for idx := range result.OutlineNodes {
		if result.OutlineNodes[idx].Metadata == nil {
			result.OutlineNodes[idx].Metadata = map[string]any{}
		}

		retrievalText := buildOutlineRetrievalText(
			result.OutlineNodes[idx],
			childParagraphRanges[result.OutlineNodes[idx].NodeID],
			result.ParagraphNodes,
			result.DocumentPages,
			result.TableSections,
		)
		if retrievalText == "" {
			delete(result.OutlineNodes[idx].Metadata, "retrieval_text")
			delete(result.OutlineNodes[idx].Metadata, "retrieval_text_source")
			continue
		}

		result.OutlineNodes[idx].Metadata["retrieval_text"] = retrievalText
		result.OutlineNodes[idx].Metadata["retrieval_text_source"] = "composite"
	}
}

func buildOutlineRetrievalText(
	node models.DocumentOutlineNode,
	childRanges []outlineParagraphSpan,
	paragraphs []models.ParagraphNode,
	pages []models.DocumentPage,
	tables []models.TableSection,
) string {
	fragments := make([]string, 0, 6)

	paragraphSpanSource, _ := metadataString(node.Metadata, "paragraph_span_source")
	if start, end, ok := outlineParagraphRange(node); ok && paragraphSpanSource != "page_range" {
		for idx := start; idx <= end && idx < len(paragraphs); idx++ {
			if paragraphIndexCoveredByAnyRange(idx, childRanges) {
				continue
			}
			content := clampText(strings.TrimSpace(paragraphs[idx].Content), 360)
			if content == "" {
				continue
			}
			fragments = appendUniqueSummaryFragment(fragments, content)
			if len(strings.Join(fragments, " ")) >= 900 {
				break
			}
		}
	} else if shouldUsePageFallbackRetrievalText(node) {
		for _, page := range matchingOutlinePages(node, pages) {
			pageSummary := clampText(strings.TrimSpace(page.Summary), 220)
			if pageSummary != "" {
				fragments = appendUniqueSummaryFragment(fragments, pageSummary)
			}

			pageContent := clampText(strings.TrimSpace(page.Content), 360)
			if pageContent != "" {
				fragments = appendUniqueSummaryFragment(fragments, pageContent)
			}
			if len(strings.Join(fragments, " ")) >= 900 {
				break
			}
		}
	}

	for _, table := range matchingOutlineTables(node, tables) {
		retrievalText := outlineTableRetrievalText(table)
		if retrievalText == "" {
			continue
		}
		fragments = appendUniqueSummaryFragment(fragments, retrievalText)
		if len(strings.Join(fragments, " ")) >= 1100 {
			break
		}
	}

	if len(fragments) == 0 && len(childRanges) == 0 {
		summary := strings.TrimSpace(node.Summary)
		if summary != "" {
			fragments = appendUniqueSummaryFragment(fragments, summary)
		}
	}

	return clampText(strings.Join(fragments, ". "), 1200)
}

type outlineParagraphSpan struct {
	start int
	end   int
}

func buildOutlineChildParagraphRanges(nodes []models.DocumentOutlineNode) map[string][]outlineParagraphSpan {
	rangesByParent := make(map[string][]outlineParagraphSpan)
	for _, node := range nodes {
		parentNodeID, _ := metadataString(node.Metadata, "parent_node_id")
		if parentNodeID == "" {
			continue
		}
		start, end, ok := outlineParagraphRange(node)
		if !ok {
			continue
		}
		rangesByParent[parentNodeID] = append(rangesByParent[parentNodeID], outlineParagraphSpan{
			start: start,
			end:   end,
		})
	}
	return rangesByParent
}

func paragraphIndexCoveredByAnyRange(index int, ranges []outlineParagraphSpan) bool {
	for _, span := range ranges {
		if index >= span.start && index <= span.end {
			return true
		}
	}
	return false
}

func shouldUsePageFallbackRetrievalText(node models.DocumentOutlineNode) bool {
	if node.Metadata == nil {
		return false
	}
	if fallbackStructure, ok := node.Metadata["fallback_structure"].(bool); ok && fallbackStructure {
		return true
	}
	segmentType, _ := metadataString(node.Metadata, "segment_type")
	return segmentType == "page"
}

func outlineParagraphRange(node models.DocumentOutlineNode) (int, int, bool) {
	start, startOK := metadataInt(node.Metadata, "paragraph_start")
	end, endOK := metadataInt(node.Metadata, "paragraph_end")
	if !startOK || !endOK || start < 0 || end < start {
		return 0, 0, false
	}
	return start, end, true
}

func matchingOutlineTables(node models.DocumentOutlineNode, tables []models.TableSection) []models.TableSection {
	if len(tables) == 0 {
		return nil
	}

	sectionKey, _ := metadataString(node.Metadata, "section_key")
	matches := make([]models.TableSection, 0, 2)
	for _, table := range tables {
		tableSectionKey, _ := metadataString(table.Metadata, "section_key")
		switch {
		case sectionKey != "" && tableSectionKey == sectionKey:
			matches = append(matches, table)
		case strings.EqualFold(strings.TrimSpace(table.Title), strings.TrimSpace(node.Title)):
			matches = append(matches, table)
		}
	}

	return matches
}

func matchingOutlinePages(node models.DocumentOutlineNode, pages []models.DocumentPage) []models.DocumentPage {
	if len(pages) == 0 || node.StartPage == nil || node.EndPage == nil {
		return nil
	}

	matches := make([]models.DocumentPage, 0, max(1, *node.EndPage-*node.StartPage+1))
	for _, page := range pages {
		if page.PageNumber < *node.StartPage || page.PageNumber > *node.EndPage {
			continue
		}
		matches = append(matches, page)
	}
	return matches
}

func summarizeTableSection(table models.TableSection) string {
	parts := make([]string, 0, 3)
	if title := strings.TrimSpace(table.Title); title != "" {
		parts = append(parts, "Table "+title)
	}
	if len(table.Headers) > 0 {
		parts = append(parts, "headers "+strings.Join(table.Headers, ", "))
	}
	if len(table.Rows) > 0 {
		firstRow := make([]string, 0, len(table.Rows[0]))
		for _, cell := range table.Rows[0] {
			cell = strings.TrimSpace(cell)
			if cell != "" {
				firstRow = append(firstRow, cell)
			}
		}
		if len(firstRow) > 0 {
			parts = append(parts, strings.Join(firstRow, " | "))
		}
	}

	return clampSummary(strings.Join(parts, " with "), 180)
}

func outlineTableRetrievalText(table models.TableSection) string {
	parts := make([]string, 0, 4)
	if title := strings.TrimSpace(table.Title); title != "" {
		parts = append(parts, title)
	}
	if len(table.Headers) > 0 {
		parts = append(parts, strings.Join(table.Headers, " "))
	}
	for rowIdx := 0; rowIdx < len(table.Rows) && rowIdx < 2; rowIdx++ {
		row := make([]string, 0, len(table.Rows[rowIdx]))
		for _, cell := range table.Rows[rowIdx] {
			cell = strings.TrimSpace(cell)
			if cell != "" {
				row = append(row, cell)
			}
		}
		if len(row) > 0 {
			parts = append(parts, strings.Join(row, " "))
		}
	}

	return clampText(strings.Join(parts, ". "), 260)
}

func appendUniqueSummaryFragment(fragments []string, fragment string) []string {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return fragments
	}

	normalized := strings.ToLower(fragment)
	for _, existing := range fragments {
		if strings.ToLower(existing) == normalized {
			return fragments
		}
	}

	return append(fragments, fragment)
}

func clampSummary(value string, maxLen int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || maxLen <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	truncated := string(runes[:maxLen])
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		return truncated[:lastSpace]
	}
	return strings.TrimSpace(truncated)
}

func outlineCoverageStats(pages []models.DocumentPage, nodes []models.DocumentOutlineNode) (int, int, float64) {
	totalPages := len(pages)
	if totalPages == 0 {
		return 0, 0, 0
	}

	coveredPages := make(map[int]struct{}, totalPages)
	for _, node := range nodes {
		if node.StartPage == nil || node.EndPage == nil {
			continue
		}

		start := *node.StartPage
		end := *node.EndPage
		if end < start {
			start, end = end, start
		}
		for pageNumber := start; pageNumber <= end; pageNumber++ {
			coveredPages[pageNumber] = struct{}{}
		}
	}

	coveredCount := len(coveredPages)
	return totalPages, coveredCount, float64(coveredCount) / float64(totalPages)
}

func outlineMaxDepth(nodes []models.DocumentOutlineNode) int {
	depth := 0
	for _, node := range nodes {
		if node.Level > depth {
			depth = node.Level
		}
	}
	return depth
}

func outlineSummaryCoverageRatio(nodes []models.DocumentOutlineNode) float64 {
	if len(nodes) == 0 {
		return 0
	}

	summarized := 0
	for _, node := range nodes {
		if strings.TrimSpace(node.Summary) != "" {
			summarized++
		}
	}

	return float64(summarized) / float64(len(nodes))
}

func metadataInt(metadata map[string]any, key string) (int, bool) {
	if metadata == nil {
		return 0, false
	}

	value, ok := metadata[key]
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func metadataString(metadata map[string]any, key string) (string, bool) {
	if metadata == nil {
		return "", false
	}

	value, ok := metadata[key]
	if !ok {
		return "", false
	}

	text, ok := value.(string)
	if !ok {
		return "", false
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	return text, true
}

func extractHTMLText(node *html.Node) string {
	var parts []string
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current == nil {
			return
		}
		if current.Type == html.TextNode {
			text := strings.TrimSpace(current.Data)
			if text != "" {
				parts = append(parts, text)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return strings.Join(parts, " ")
}

func extractHTMLTable(documentID uint, sequence int, node *html.Node) models.TableSection {
	rows := make([][]string, 0)
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current == nil {
			return
		}
		if current.Type == html.ElementNode && current.Data == "tr" {
			cells := make([]string, 0)
			for child := current.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
					text := normalizeText(extractHTMLText(child))
					cells = append(cells, text)
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)

	section := models.TableSection{
		DocumentID:    documentID,
		SequenceOrder: sequence,
		Headers:       []string{},
		Rows:          [][]string{},
		Metadata:      map[string]any{},
	}
	if len(rows) > 0 {
		section.Headers = rows[0]
	}
	if len(rows) > 1 {
		section.Rows = rows[1:]
	}
	return section
}

func blocksFromPDFPage(page pdf.Page, pageNumber *int) []textBlock {
	content := page.Content()
	if len(content.Text) == 0 {
		return nil
	}

	rowBuckets := make(map[int][]pdf.Text)
	rowPositions := make([]int, 0)
	for _, text := range content.Text {
		if strings.TrimSpace(text.S) == "" {
			continue
		}

		key := int(math.Round(text.Y))
		if _, exists := rowBuckets[key]; !exists {
			rowPositions = append(rowPositions, key)
		}
		rowBuckets[key] = append(rowBuckets[key], text)
	}

	if len(rowPositions) == 0 {
		return nil
	}

	sort.Slice(rowPositions, func(i, j int) bool {
		return rowPositions[i] > rowPositions[j]
	})

	lines := make([]string, 0, len(rowPositions))
	for _, position := range rowPositions {
		texts := rowBuckets[position]
		sort.Slice(texts, func(i, j int) bool {
			if texts[i].X == texts[j].X {
				return texts[i].Y > texts[j].Y
			}
			return texts[i].X < texts[j].X
		})

		line := lineFromPDFTexts(texts)
		if line != "" {
			lines = append(lines, line)
		}
	}

	blocks := make([]textBlock, 0, len(lines))
	lineBuffer := make([]string, 0, 4)

	flush := func() {
		if len(lineBuffer) == 0 {
			return
		}
		content := normalizeText(strings.Join(lineBuffer, "\n"))
		lineBuffer = lineBuffer[:0]
		if content == "" {
			return
		}
		var pageRef *int
		if pageNumber != nil {
			pageCopy := *pageNumber
			pageRef = &pageCopy
		}
		blocks = append(blocks, textBlock{
			Content:    content,
			PageNumber: pageRef,
			Source:     "pdf_text",
		})
	}

	for _, line := range lines {
		if line == "" {
			flush()
			continue
		}

		lineBuffer = append(lineBuffer, line)
		if looksLikeHeading(line) || strings.HasSuffix(line, ".") || strings.HasSuffix(line, ":") {
			flush()
		}
	}

	flush()
	return blocks
}

func blocksFromOCRPageText(text string, pageNumber int) []textBlock {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	blocks := make([]textBlock, 0, len(lines))
	lineBuffer := make([]string, 0, 4)

	flush := func() {
		if len(lineBuffer) == 0 {
			return
		}
		content := normalizeText(strings.Join(lineBuffer, "\n"))
		lineBuffer = lineBuffer[:0]
		if content == "" {
			return
		}
		blocks = append(blocks, textBlock{
			Content:    content,
			PageNumber: intPtr(pageNumber),
			Source:     "ocr",
		})
	}

	for _, rawLine := range lines {
		line := normalizeText(rawLine)
		if line == "" {
			flush()
			continue
		}

		if looksLikePDFHeading(line) {
			flush()
			blocks = append(blocks, textBlock{
				Content:    line,
				PageNumber: intPtr(pageNumber),
				Source:     "ocr",
			})
			continue
		}

		lineBuffer = append(lineBuffer, line)
		if strings.HasSuffix(line, ".") || strings.HasSuffix(line, ":") {
			flush()
		}
	}

	flush()
	return blocks
}

func lineFromPDFTexts(texts []pdf.Text) string {
	if len(texts) == 0 {
		return ""
	}

	var builder strings.Builder
	lastEndX := 0.0
	hasContent := false
	lastText := ""

	for _, fragment := range texts {
		text := normalizeText(fragment.S)
		if text != "" {
			if hasContent {
				gap := fragment.X - lastEndX
				if shouldInsertPDFSpace(gap, lastText, text, fragment) {
					builder.WriteRune(' ')
				}
			}

			builder.WriteString(text)
			lastEndX = fragment.X + estimatedPDFFragmentWidth(fragment, text)
			hasContent = true
			lastText = text
		}
	}

	return normalizeText(builder.String())
}

func shouldInsertPDFSpace(gap float64, previousText, currentText string, fragment pdf.Text) bool {
	if gap <= 0 {
		return false
	}

	if gap > pdfSpaceThreshold(fragment) {
		return true
	}

	if gap > 1.5 && (isStandalonePunctuation(previousText) || isStandalonePunctuation(currentText)) {
		return true
	}

	return false
}

func pdfSpaceThreshold(fragment pdf.Text) float64 {
	fontSize := fragment.FontSize
	if fontSize <= 0 {
		fontSize = 8
	}

	return math.Max(fontSize*0.3, 1.5)
}

func estimatedPDFFragmentWidth(fragment pdf.Text, text string) float64 {
	width := fragment.W
	if width > 0 {
		return width
	}

	runeCount := len([]rune(text))
	if runeCount == 0 {
		return 0
	}

	fontSize := fragment.FontSize
	if fontSize <= 0 {
		fontSize = 8
	}

	return math.Max(fontSize*0.45*float64(runeCount), 1)
}

func isStandalonePunctuation(text string) bool {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) != 1 {
		return false
	}

	r := runes[0]
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func sectionTitleForKey(headings []outlineDraft, sectionKey string) string {
	if sectionKey == "" {
		return ""
	}

	for _, heading := range headings {
		if heading.SectionKey == sectionKey {
			return heading.Title
		}
	}

	return ""
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

var numberedHeadingPattern = regexp.MustCompile(`^\d+(\.\d+)*\s+`)
var pdfPageMarkerPattern = regexp.MustCompile(`^(p[aá]gina|page)\s+\d+(\s+(de|of)\s+\d+)?$`)
var pdfProcessNumberPattern = regexp.MustCompile(`\b\d{4,7}-\d{2}\.\d{4}\.\d\.\d{2}\.\d{4}\b`)

type documentHeadingArchetype string

const (
	documentHeadingArchetypeUnknown      documentHeadingArchetype = ""
	documentHeadingArchetypeLegalRoot    documentHeadingArchetype = "legal_root"
	documentHeadingArchetypeLegalSection documentHeadingArchetype = "legal_section"
	documentHeadingArchetypeAttachment   documentHeadingArchetype = "attachment"
)

var legalDocumentRootHeadingKeywords = []string{
	"sentenca",
	"acordao",
	"decisao",
	"despacho",
	"oficio",
	"certidao",
	"mandado",
	"alvara",
}

var legalSectionHeadingKeywords = []string{
	"relatorio",
	"fundamentacao",
	"dispositivo",
	"ementa",
	"voto",
}

var attachmentHeadingKeywords = []string{
	"anexo",
	"apendice",
	"planilha",
	"memoria de calculo",
	"quadro demonstrativo",
	"demonstrativo de calculo",
}

func looksLikeHeading(text string) bool {
	if text == "" || strings.Contains(text, ". ") && len(text) > 120 {
		return false
	}
	if classifyDocumentHeadingArchetype(text) != documentHeadingArchetypeUnknown {
		return true
	}
	if numberedHeadingPattern.MatchString(text) {
		return true
	}
	if strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?") {
		return false
	}
	if len(strings.Fields(text)) <= 12 && text == strings.ToUpper(text) {
		return true
	}
	return len(strings.Fields(text)) <= 10 && isMostlyTitleCase(text)
}

func looksLikePDFHeading(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !looksLikeHeading(trimmed) {
		return false
	}
	if isLowSignalPDFText(trimmed) || looksLikePDFFieldLine(trimmed) || looksLikePDFBoilerplate(trimmed) {
		return false
	}
	return true
}

func looksLikePDFFieldLine(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if numberedHeadingPattern.MatchString(trimmed) {
		return false
	}
	if pdfPageMarkerPattern.MatchString(strings.ToLower(trimmed)) {
		return true
	}
	if pdfProcessNumberPattern.MatchString(trimmed) {
		return true
	}
	if strings.HasSuffix(trimmed, ":") {
		return true
	}
	if strings.Contains(trimmed, "@") {
		return true
	}
	if strings.Contains(strings.ToLower(trimmed), "processo nº") ||
		strings.Contains(strings.ToLower(trimmed), "processo n°") ||
		strings.Contains(strings.ToLower(trimmed), "processo no ") {
		return true
	}
	return strings.Count(trimmed, ":") >= 1
}

func looksLikePDFBoilerplate(text string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(text)), " "))
	if normalized == "" {
		return false
	}

	if pdfPageMarkerPattern.MatchString(normalized) || pdfProcessNumberPattern.MatchString(text) {
		return true
	}

	if classifyDocumentHeadingArchetype(text) != documentHeadingArchetypeUnknown {
		return false
	}

	if len(strings.Fields(text)) >= 8 && text == strings.ToUpper(text) && !numberedHeadingPattern.MatchString(text) {
		return true
	}

	return containsAnySubstring(normalized,
		"tribunal de justiça",
		"tribunal de justica",
		"poder judiciário",
		"poder judiciario",
		"diário da justiça",
		"diario da justica",
		"publicação oficial",
		"publicacao oficial",
		"documento assinado digitalmente",
		"assinado digitalmente",
		"código de autenticidade",
		"codigo de autenticidade",
		"código verificador",
		"codigo verificador",
		"para conferir o original",
		"processo digital nº",
		"processo digital n°",
		"http://",
		"https://",
		"www.",
	)
}

func isLowSignalPDFText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	if numberedHeadingPattern.MatchString(trimmed) {
		remainder := strings.TrimSpace(numberedHeadingPattern.ReplaceAllString(trimmed, ""))
		return !containsLetter(remainder)
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return true
	}

	letters := 0
	digits := 0
	meaningfulWords := 0
	shortWords := 0

	for _, word := range words {
		wordLetters := 0
		wordDigits := 0
		wordRunes := 0
		for _, r := range word {
			if unicode.IsLetter(r) {
				letters++
				wordLetters++
				wordRunes++
				continue
			}
			if unicode.IsDigit(r) {
				digits++
				wordDigits++
				wordRunes++
			}
		}

		if wordRunes == 0 {
			continue
		}
		if wordRunes <= 2 {
			shortWords++
		}
		if wordLetters >= 3 || (wordLetters >= 1 && wordDigits >= 2) {
			meaningfulWords++
		}
	}

	switch {
	case letters == 0:
		return digits <= 6
	case letters < 3 && meaningfulWords == 0:
		return true
	case meaningfulWords == 0 && shortWords >= min(2, len(words)):
		return true
	case len(words) == 1 && letters <= 2:
		return true
	case len(words) <= 4 && shortWords == len(words) && meaningfulWords == 0:
		return true
	default:
		return false
	}
}

func containsLetter(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func clonePDFPageBlocks(pageBlocks map[int][]textBlock) map[int][]textBlock {
	if len(pageBlocks) == 0 {
		return map[int][]textBlock{}
	}

	cloned := make(map[int][]textBlock, len(pageBlocks))
	for pageNumber, blocks := range pageBlocks {
		next := make([]textBlock, len(blocks))
		copy(next, blocks)
		cloned[pageNumber] = next
	}
	return cloned
}

func pagesNeedingPDFOCRFallback(pageCount int, pageBlocks map[int][]textBlock) ([]int, string) {
	if pageCount <= 0 {
		return nil, ""
	}

	pages := make([]int, 0, pageCount)
	nativePages := 0
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		blocks := pageBlocks[pageNumber]
		if !pdfPageNeedsOCRFallback(blocks) {
			nativePages++
			continue
		}
		pages = append(pages, pageNumber)
	}

	if len(pages) == 0 {
		return nil, ""
	}
	if nativePages == 0 {
		return pages, "no_native_pdf_text"
	}
	return pages, "degraded_native_pdf_text"
}

func pdfPageNeedsOCRFallback(blocks []textBlock) bool {
	text := strings.TrimSpace(textFromBlocks(blocks))
	if text == "" {
		return true
	}

	letters := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			letters++
		}
	}

	return estimateTextTokens(text) < 12 && letters < 48
}

func textFromBlocks(blocks []textBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		content := strings.TrimSpace(block.Content)
		if content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n")
}

func appendOCRFallbackMetadata(result *domain.IndexingResult, documentID uint, stats pdfOCRFallbackStats) {
	if result == nil || documentID == 0 {
		return
	}

	appendIndexingMetadataPair(result, documentID, "ocr_fallback_needed", strconv.FormatBool(stats.Needed))
	appendIndexingMetadataPair(result, documentID, "ocr_fallback_attempted", strconv.FormatBool(stats.Attempted))
	appendIndexingMetadataPair(result, documentID, "ocr_fallback_applied", strconv.FormatBool(stats.Applied))
	if strings.TrimSpace(stats.Engine) != "" {
		appendIndexingMetadataPair(result, documentID, "ocr_engine", stats.Engine)
	}
	if strings.TrimSpace(stats.Reason) != "" {
		appendIndexingMetadataPair(result, documentID, "ocr_fallback_reason", stats.Reason)
	}
	if len(stats.PageNumbers) > 0 {
		pageNumbers := make([]string, 0, len(stats.PageNumbers))
		for _, pageNumber := range stats.PageNumbers {
			pageNumbers = append(pageNumbers, strconv.Itoa(pageNumber))
		}
		appendIndexingMetadataPair(result, documentID, "ocr_page_count", strconv.Itoa(len(stats.PageNumbers)))
		appendIndexingMetadataPair(result, documentID, "ocr_page_numbers", strings.Join(pageNumbers, ","))
	}
}

func appendIndexingMetadataPair(result *domain.IndexingResult, documentID uint, key, value string) {
	if result == nil || documentID == 0 {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return
	}

	for idx := range result.MetadataPairs {
		if result.MetadataPairs[idx].Key == key {
			result.MetadataPairs[idx].Value = value
			return
		}
	}

	result.MetadataPairs = append(result.MetadataPairs, models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: len(result.MetadataPairs),
		Key:           key,
		Value:         value,
	})
}

func countPDFBlockPageOccurrences(pageBlocks map[int][]textBlock, pageCount int) map[string]int {
	counts := make(map[string]int)
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		blocks := pageBlocks[pageNumber]
		seenOnPage := make(map[string]struct{})
		for _, block := range blocks {
			content := strings.TrimSpace(block.Content)
			if content == "" {
				continue
			}
			seenOnPage[content] = struct{}{}
		}
		for content := range seenOnPage {
			counts[content]++
		}
	}
	return counts
}

func shouldSuppressRepeatedPDFHeading(text string, occurrences map[string]int) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if looksLikePDFBoilerplate(trimmed) && occurrences[trimmed] >= 2 {
		return true
	}
	return occurrences[trimmed] >= 3 && len(strings.Fields(trimmed)) <= 16
}

func inferHeadingLevel(text string) int {
	switch classifyDocumentHeadingArchetype(text) {
	case documentHeadingArchetypeLegalRoot:
		return 1
	case documentHeadingArchetypeLegalSection, documentHeadingArchetypeAttachment:
		return 2
	}

	if matches := numberedHeadingPattern.FindString(text); matches != "" {
		trimmed := strings.TrimSpace(matches)
		return strings.Count(trimmed, ".") + 1
	}

	switch {
	case text == strings.ToUpper(text):
		return 1
	case len(strings.Fields(text)) <= 4:
		return 2
	default:
		return 3
	}
}

func classifyDocumentHeadingArchetype(text string) documentHeadingArchetype {
	normalized := normalizeHeadingLexicon(text)
	if normalized == "" {
		return documentHeadingArchetypeUnknown
	}

	wordCount := len(strings.Fields(normalized))
	if wordCount == 0 || wordCount > 14 {
		return documentHeadingArchetypeUnknown
	}

	candidates := []string{normalized}
	if stripped := strings.TrimSpace(numberedHeadingPattern.ReplaceAllString(normalized, "")); stripped != "" && stripped != normalized {
		candidates = append(candidates, stripped)
	}

	for _, candidate := range candidates {
		switch {
		case matchesHeadingKeywordFamily(candidate, legalDocumentRootHeadingKeywords):
			return documentHeadingArchetypeLegalRoot
		case matchesHeadingKeywordFamily(candidate, legalSectionHeadingKeywords):
			return documentHeadingArchetypeLegalSection
		case matchesHeadingKeywordFamily(candidate, attachmentHeadingKeywords):
			return documentHeadingArchetypeAttachment
		}
	}

	return documentHeadingArchetypeUnknown
}

func matchesHeadingKeywordFamily(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if value == keyword || strings.HasPrefix(value, keyword+" ") {
			return true
		}
	}
	return false
}

func normalizeHeadingLexicon(text string) string {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	lastSpace := false
	for _, r := range trimmed {
		folded := foldLatinRune(r)
		switch {
		case unicode.IsLetter(folded) || unicode.IsDigit(folded):
			builder.WriteRune(folded)
			lastSpace = false
		default:
			if !lastSpace && builder.Len() > 0 {
				builder.WriteRune(' ')
				lastSpace = true
			}
		}
	}

	return strings.TrimSpace(builder.String())
}

func foldLatinRune(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	default:
		return r
	}
}

func isMostlyTitleCase(text string) bool {
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}

	titleCaseCount := 0
	for _, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		if unicode.IsUpper(runes[0]) {
			titleCaseCount++
		}
	}

	return titleCaseCount*2 >= len(words)
}

func normalizeCells(cells []string) []string {
	result := make([]string, 0, len(cells))
	for _, cell := range cells {
		result = append(result, normalizeText(cell))
	}
	return result
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")

	paragraphs := strings.Split(text, "\n\n")
	normalizedParagraphs := make([]string, 0, len(paragraphs))
	for _, para := range paragraphs {
		lines := strings.Split(para, "\n")
		compacted := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
			if line != "" {
				compacted = append(compacted, line)
			}
		}
		normalized := strings.TrimSpace(strings.Join(compacted, " "))
		if normalized != "" {
			normalizedParagraphs = append(normalizedParagraphs, normalized)
		}
	}
	return strings.Join(normalizedParagraphs, "\n\n")
}

func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if idx := strings.Index(text, "."); idx > 0 {
		return strings.TrimSpace(text[:idx])
	}
	words := strings.Fields(text)
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, " ")
}

func slugify(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	var builder strings.Builder
	lastDash := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "section"
	}
	return slug
}

func buildOutlineNodeID(sequence int, title string) string {
	return clampIdentifierWithHash(
		fmt.Sprintf("node-%03d-%s", sequence, slugify(title)),
		maxOutlineNodeIDLength,
	)
}

func buildOutlineSlug(title string) string {
	return clampIdentifierWithHash(slugify(title), maxOutlineSlugLength)
}

func buildOutlineNodePath(parts []string) string {
	return clampIdentifierWithHash(strings.Join(parts, "/"), maxOutlineNodePathLength)
}

func clampIdentifierWithHash(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 9 {
		return value[:limit]
	}

	hash := shortStableHash(value)
	keep := limit - len(hash) - 1
	if keep < 1 {
		keep = 1
	}

	return value[:keep] + "-" + hash
}

func shortStableHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intPtr(value int) *int {
	return &value
}

func estimateTextTokens(content string) int {
	return len(strings.Fields(content))
}

func sortedMapKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return []string{}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortInts(values []int) {
	sort.Ints(values)
}

func containsAnySubstring(value string, substrings ...string) bool {
	for _, substring := range substrings {
		if substring != "" && strings.Contains(value, substring) {
			return true
		}
	}
	return false
}
