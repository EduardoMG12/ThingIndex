package indexing

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

type DeterministicIndexer struct{}

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
}

func NewDeterministicIndexer() *DeterministicIndexer {
	return &DeterministicIndexer{}
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
		return i.indexPDF(doc.ID, data)
	default:
		return i.indexPlainText(doc.ID, string(data)), nil
	}
}

func (i *DeterministicIndexer) indexHTML(documentID uint, data []byte) (domain.IndexingResult, error) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse html: %v", domain.ErrDocumentIndexing, err)
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
		return domain.IndexingResult{}, fmt.Errorf("%w: parse spreadsheet: %v", domain.ErrDocumentIndexing, err)
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
			return domain.IndexingResult{}, fmt.Errorf("%w: read spreadsheet rows: %v", domain.ErrDocumentIndexing, err)
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

func (i *DeterministicIndexer) indexPDF(documentID uint, data []byte) (domain.IndexingResult, error) {
	reader := bytes.NewReader(data)
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse pdf: %v", domain.ErrDocumentIndexing, err)
	}

	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	currentSectionKey := ""

	for pageIndex := 1; pageIndex <= pdfReader.NumPage(); pageIndex++ {
		page := pdfReader.Page(pageIndex)
		if page.V.IsNull() || page.V.Key("Contents").Kind() == pdf.Null {
			continue
		}

		rows, rowErr := page.GetTextByRow()
		if rowErr != nil {
			return domain.IndexingResult{}, fmt.Errorf("%w: extract pdf rows: %v", domain.ErrDocumentIndexing, rowErr)
		}

		pageNumber := pageIndex
		for _, block := range blocksFromPDFRows(rows, &pageNumber) {
			if looksLikeHeading(block.Content) {
				currentSectionKey = slugify(block.Content)
				headings = append(headings, outlineDraft{
					Level:          inferHeadingLevel(block.Content),
					Title:          block.Content,
					PageNumber:     block.PageNumber,
					ParagraphStart: len(paragraphs),
					SectionKey:     currentSectionKey,
				})
				continue
			}

			block.SectionKey = currentSectionKey
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

	return buildIndexingResult(documentID, headings, paragraphs, nil), nil
}

func (i *DeterministicIndexer) indexPlainText(documentID uint, raw string) domain.IndexingResult {
	blocks := splitBlocks(raw)
	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0, len(blocks))
	currentSectionKey := ""

	for _, block := range blocks {
		if looksLikeHeading(block) {
			currentSectionKey = slugify(block)
			headings = append(headings, outlineDraft{
				Level:          inferHeadingLevel(block),
				Title:          block,
				ParagraphStart: len(paragraphs),
				SectionKey:     currentSectionKey,
			})
			continue
		}

		paragraphs = append(paragraphs, textBlock{
			Content:    block,
			SectionKey: currentSectionKey,
		})
	}

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

	metadataPairs := []models.MetadataPair{
		{DocumentID: documentID, SequenceOrder: 0, Key: "paragraph_count", Value: strconv.Itoa(len(paragraphNodes))},
		{DocumentID: documentID, SequenceOrder: 1, Key: "table_count", Value: strconv.Itoa(len(tables))},
		{DocumentID: documentID, SequenceOrder: 2, Key: "outline_count", Value: strconv.Itoa(len(outlineNodes))},
	}

	allTextParts := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph.Content) != "" {
			allTextParts = append(allTextParts, paragraph.Content)
		}
	}
	allText := strings.Join(allTextParts, " ")
	metadataPairs = append(metadataPairs,
		models.MetadataPair{DocumentID: documentID, SequenceOrder: 3, Key: "word_count", Value: strconv.Itoa(len(strings.Fields(allText)))},
		models.MetadataPair{DocumentID: documentID, SequenceOrder: 4, Key: "character_count", Value: strconv.Itoa(len([]rune(allText)))},
	)

	return domain.IndexingResult{
		DocumentPages:  documentPages,
		ParagraphNodes: paragraphNodes,
		TableSections:  tables,
		MetadataPairs:  metadataPairs,
		OutlineNodes:   outlineNodes,
	}
}

func buildDocumentPages(documentID uint, paragraphs []textBlock) []models.DocumentPage {
	pageBuckets := make(map[int][]string)
	pageSectionKeys := make(map[int]map[string]struct{})

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
		nodeID := fmt.Sprintf("node-%03d-%s", len(nodes), slugify(title))
		startPage, endPage := headingPageRange(idx, headings, paragraphs)

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
			"section_key":     heading.SectionKey,
		}
		if heading.PageNumber != nil && *heading.PageNumber > 0 {
			metadata["page_number"] = *heading.PageNumber
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
			Slug:          slugify(title),
			Summary:       "",
			StartPage:     startPage,
			EndPage:       endPage,
			NodePath:      strings.Join(pathParts, "/"),
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
		endParagraph = max(startParagraph, headings[index+1].ParagraphStart-1)
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

func blocksFromPDFRows(rows pdf.Rows, pageNumber *int) []textBlock {
	blocks := make([]textBlock, 0, len(rows))
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
		})
	}

	for _, row := range rows {
		line := pdfRowToLine(row)
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

func pdfRowToLine(row *pdf.Row) string {
	if row == nil {
		return ""
	}

	parts := make([]string, 0, len(row.Content))
	for _, word := range row.Content {
		text := normalizeText(word.S)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return normalizeText(strings.Join(parts, " "))
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

func looksLikeHeading(text string) bool {
	if text == "" || strings.Contains(text, ". ") && len(text) > 120 {
		return false
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

func inferHeadingLevel(text string) int {
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
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	compacted := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
		if line != "" {
			compacted = append(compacted, line)
		}
	}
	return strings.TrimSpace(strings.Join(compacted, "\n"))
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
