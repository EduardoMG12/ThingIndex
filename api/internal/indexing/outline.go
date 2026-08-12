package indexing

import (
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

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
		if paragraph.Box != nil {
			metadata["box"] = paragraph.Box
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

	chunks := newChunkBuilder(documentID).buildFromOutline(outlineNodes, paragraphNodes, tables, documentPages)

	result := domain.IndexingResult{
		DocumentPages:  documentPages,
		ParagraphNodes: paragraphNodes,
		TableSections:  tables,
		OutlineNodes:   outlineNodes,
		Chunks:         chunks,
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
