package processing

import (
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const (
	documentChunkTypeParagraphWindow = "paragraph_window"
	documentChunkTypeTable           = "table_section"
	chunkStrategyVersion             = "section-window-v3"
	chunkTargetChars                 = 1200
	chunkOverlapParagraphs           = 1
	maxRelatedTablesPerChunk         = 2
	relatedTableRowLimit             = 2
)

func buildDocumentChunks(documentID uint, result domain.IndexingResult) []models.DocumentChunk {
	chunks := make([]models.DocumentChunk, 0)
	appendParagraphWindowChunks(documentID, result, &chunks)
	appendTableChunks(documentID, result, &chunks)

	return chunks
}

func appendParagraphWindowChunks(documentID uint, result domain.IndexingResult, chunks *[]models.DocumentChunk) {
	if len(result.ParagraphNodes) == 0 {
		return
	}

	start := 0
	for start < len(result.ParagraphNodes) {
		end := nextChunkBoundary(result.ParagraphNodes, start)

		window := result.ParagraphNodes[start:end]
		title, outlineNode := chunkTitle(window, result.OutlineNodes)
		relatedTables := relatedTablesForWindow(window, title, outlineNode, result.TableSections)
		content := buildParagraphChunkContent(window, relatedTables)
		if strings.TrimSpace(content) != "" {
			chunkStart := window[0].SequenceOrder
			chunkEnd := window[len(window)-1].SequenceOrder
			pageStart, pageEnd := paragraphPageRange(window)

			metadata := map[string]any{
				"paragraph_count":        len(window),
				"chunk_strategy_version": chunkStrategyVersion,
				"outline_hint":           title,
				"related_table_count":    len(relatedTables),
			}
			if pageStart != nil {
				metadata["page_start"] = *pageStart
			}
			if pageEnd != nil {
				metadata["page_end"] = *pageEnd
			}
			if sectionKey := paragraphMetadataValue(window[0], "section_key"); sectionKey != "" {
				metadata["section_key"] = sectionKey
			}
			if sectionTitle := paragraphMetadataValue(window[0], "section_title"); sectionTitle != "" {
				metadata["section_title"] = sectionTitle
			}
			if outlineNode != nil {
				metadata["outline_node_id"] = outlineNode.NodeID
				metadata["outline_level"] = outlineNode.Level
				if strings.TrimSpace(outlineNode.NodePath) != "" {
					metadata["outline_node_path"] = outlineNode.NodePath
				}
				if metadata["section_key"] == nil {
					if sectionKey := outlineMetadataValue(outlineNode.Metadata, "section_key"); sectionKey != "" {
						metadata["section_key"] = sectionKey
					}
				}
			}
			if len(relatedTables) > 0 {
				metadata["related_table_titles"] = relatedTableTitles(relatedTables)
			}

			*chunks = append(*chunks, models.DocumentChunk{
				DocumentID:           documentID,
				SequenceOrder:        len(*chunks),
				ChunkType:            documentChunkTypeParagraphWindow,
				Title:                title,
				Content:              content,
				SourceParagraphStart: &chunkStart,
				SourceParagraphEnd:   &chunkEnd,
				TokenEstimate:        estimateTokenCount(content),
				Metadata:             metadata,
			})
		}

		if end >= len(result.ParagraphNodes) {
			break
		}

		nextStart := end
		if shouldOverlapChunkBoundary(result.ParagraphNodes, end-1, end) {
			nextStart = end - chunkOverlapParagraphs
			if nextStart <= start {
				nextStart = end
			}
		}
		start = nextStart
	}
}

func appendTableChunks(documentID uint, result domain.IndexingResult, chunks *[]models.DocumentChunk) {
	for _, table := range result.TableSections {
		content := renderTableChunkContent(table)
		if strings.TrimSpace(content) == "" {
			continue
		}

		metadata := map[string]any{
			"header_count":           len(table.Headers),
			"row_count":              len(table.Rows),
			"chunk_strategy_version": chunkStrategyVersion,
		}
		if sectionKey := tableMetadataValue(table.Metadata, "section_key"); sectionKey != "" {
			metadata["section_key"] = sectionKey
		}

		*chunks = append(*chunks, models.DocumentChunk{
			DocumentID:    documentID,
			SequenceOrder: len(*chunks),
			ChunkType:     documentChunkTypeTable,
			Title:         strings.TrimSpace(table.Title),
			Content:       content,
			TokenEstimate: estimateTokenCount(content),
			Metadata:      metadata,
		})
	}
}

func joinParagraphWindow(nodes []models.ParagraphNode) string {
	parts := make([]string, 0, len(nodes))
	for _, node := range nodes {
		content := strings.TrimSpace(node.Content)
		if content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func buildParagraphChunkContent(window []models.ParagraphNode, relatedTables []models.TableSection) string {
	content := joinParagraphWindow(window)
	if len(relatedTables) == 0 {
		return content
	}

	tableParts := make([]string, 0, len(relatedTables))
	for _, table := range relatedTables {
		rendered := renderRelatedTableContext(table)
		if strings.TrimSpace(rendered) != "" {
			tableParts = append(tableParts, rendered)
		}
	}
	if len(tableParts) == 0 {
		return content
	}

	if strings.TrimSpace(content) == "" {
		return "Related tables:\n" + strings.Join(tableParts, "\n")
	}

	return strings.TrimSpace(content) + "\n\nRelated tables:\n" + strings.Join(tableParts, "\n")
}

func estimateTokenCount(content string) int {
	return len(strings.Fields(content))
}

func renderTableChunkContent(table models.TableSection) string {
	var parts []string
	if title := strings.TrimSpace(table.Title); title != "" {
		parts = append(parts, "Table: "+title)
	}
	if len(table.Headers) > 0 {
		parts = append(parts, "Headers: "+strings.Join(table.Headers, " | "))
	}
	limit := min(3, len(table.Rows))
	for idx := 0; idx < limit; idx++ {
		parts = append(parts, "Row: "+strings.Join(table.Rows[idx], " | "))
	}
	return strings.Join(parts, "\n")
}

func renderRelatedTableContext(table models.TableSection) string {
	parts := make([]string, 0, relatedTableRowLimit+2)
	if title := strings.TrimSpace(table.Title); title != "" {
		parts = append(parts, "Table: "+title)
	}
	if len(table.Headers) > 0 {
		parts = append(parts, "Headers: "+strings.Join(table.Headers, " | "))
	}

	for idx := 0; idx < len(table.Rows) && idx < relatedTableRowLimit; idx++ {
		row := make([]string, 0, len(table.Rows[idx]))
		for _, cell := range table.Rows[idx] {
			cell = strings.TrimSpace(cell)
			if cell != "" {
				row = append(row, cell)
			}
		}
		if len(row) > 0 {
			parts = append(parts, "Row: "+strings.Join(row, " | "))
		}
	}

	return strings.Join(parts, "\n")
}

func nextChunkBoundary(nodes []models.ParagraphNode, start int) int {
	end := start
	charCount := 0
	sectionKey := paragraphMetadataValue(nodes[start], "section_key")
	pageNumber := nodes[start].PageNumber

	for end < len(nodes) {
		node := nodes[end]
		paragraph := strings.TrimSpace(node.Content)
		if paragraph == "" {
			end++
			continue
		}

		if end > start {
			if sectionKey != "" && paragraphMetadataValue(node, "section_key") != sectionKey {
				break
			}
			if pageNumber != nil && node.PageNumber != nil && *node.PageNumber != *pageNumber {
				break
			}
		}

		nextCount := charCount + len([]rune(paragraph))
		if end > start {
			nextCount++
		}
		if end > start && nextCount > chunkTargetChars {
			break
		}

		charCount = nextCount
		end++
	}

	if end == start {
		return start + 1
	}

	return end
}

func chunkTitle(window []models.ParagraphNode, outline []models.DocumentOutlineNode) (string, *models.DocumentOutlineNode) {
	if len(window) == 0 {
		return "", nil
	}

	if sectionTitle := paragraphMetadataValue(window[0], "section_title"); sectionTitle != "" {
		if node := bestOutlineNodeForParagraphRange(outline, window[0].SequenceOrder, window[len(window)-1].SequenceOrder); node != nil {
			return sectionTitle, node
		}
		return sectionTitle, nil
	}

	start := window[0].SequenceOrder
	end := window[len(window)-1].SequenceOrder
	if node := bestOutlineNodeForParagraphRange(outline, start, end); node != nil {
		return strings.TrimSpace(node.Title), node
	}
	return outlineTitleForParagraphRange(outline, start, end), nil
}

func paragraphMetadataValue(node models.ParagraphNode, key string) string {
	if node.Metadata == nil {
		return ""
	}

	raw, ok := node.Metadata[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func outlineMetadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	raw, ok := metadata[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func tableMetadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	raw, ok := metadata[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func paragraphPageRange(window []models.ParagraphNode) (*int, *int) {
	var pageStart *int
	var pageEnd *int

	for _, node := range window {
		if node.PageNumber == nil {
			continue
		}
		if pageStart == nil {
			start := *node.PageNumber
			pageStart = &start
		}
		end := *node.PageNumber
		pageEnd = &end
	}

	return pageStart, pageEnd
}

func shouldOverlapChunkBoundary(nodes []models.ParagraphNode, leftIdx, rightIdx int) bool {
	if leftIdx < 0 || rightIdx < 0 || leftIdx >= len(nodes) || rightIdx >= len(nodes) {
		return false
	}

	leftSection := paragraphMetadataValue(nodes[leftIdx], "section_key")
	rightSection := paragraphMetadataValue(nodes[rightIdx], "section_key")
	if leftSection != "" && rightSection != "" && leftSection != rightSection {
		return false
	}

	if nodes[leftIdx].PageNumber != nil && nodes[rightIdx].PageNumber != nil && *nodes[leftIdx].PageNumber != *nodes[rightIdx].PageNumber {
		return false
	}

	return true
}

func outlineTitleForParagraphRange(outline []models.DocumentOutlineNode, start, end int) string {
	if len(outline) == 0 {
		return ""
	}

	if node := bestOutlineNodeForParagraphRange(outline, start, end); node != nil {
		return strings.TrimSpace(node.Title)
	}

	for _, node := range outline {
		if strings.TrimSpace(node.Title) == "" {
			continue
		}
		if node.SequenceOrder >= 0 {
			return strings.TrimSpace(node.Title)
		}
	}

	return ""
}

func bestOutlineNodeForParagraphRange(outline []models.DocumentOutlineNode, start, end int) *models.DocumentOutlineNode {
	bestIdx := -1
	bestCovers := false
	bestLevel := -1
	bestSpan := 0
	bestOverlap := 0

	for idx := range outline {
		paragraphStart, paragraphEnd, ok := outlineParagraphRange(outline[idx])
		if !ok {
			continue
		}

		overlap := paragraphRangeOverlap(start, end, paragraphStart, paragraphEnd)
		if overlap <= 0 {
			continue
		}

		covers := paragraphStart <= start && paragraphEnd >= end
		span := paragraphEnd - paragraphStart + 1
		if bestIdx < 0 {
			bestIdx = idx
			bestCovers = covers
			bestLevel = outline[idx].Level
			bestSpan = span
			bestOverlap = overlap
			continue
		}

		switch {
		case covers && !bestCovers:
			bestIdx = idx
			bestCovers = true
			bestLevel = outline[idx].Level
			bestSpan = span
			bestOverlap = overlap
		case covers == bestCovers && covers && outline[idx].Level > bestLevel:
			bestIdx = idx
			bestLevel = outline[idx].Level
			bestSpan = span
			bestOverlap = overlap
		case covers == bestCovers && covers && outline[idx].Level == bestLevel && span < bestSpan:
			bestIdx = idx
			bestSpan = span
			bestOverlap = overlap
		case !bestCovers && overlap > bestOverlap:
			bestIdx = idx
			bestLevel = outline[idx].Level
			bestSpan = span
			bestOverlap = overlap
		case !bestCovers && overlap == bestOverlap && outline[idx].Level > bestLevel:
			bestIdx = idx
			bestLevel = outline[idx].Level
			bestSpan = span
		}
	}

	if bestIdx < 0 {
		return nil
	}

	return &outline[bestIdx]
}

func outlineParagraphRange(node models.DocumentOutlineNode) (int, int, bool) {
	if node.Metadata == nil {
		return 0, 0, false
	}

	start, startOK := metadataIntValue(node.Metadata, "paragraph_start")
	end, endOK := metadataIntValue(node.Metadata, "paragraph_end")
	if !startOK || !endOK || start < 0 || end < start {
		return 0, 0, false
	}

	return start, end, true
}

func metadataIntValue(metadata map[string]any, key string) (int, bool) {
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

func paragraphRangeOverlap(leftStart, leftEnd, rightStart, rightEnd int) int {
	start := max(leftStart, rightStart)
	end := min(leftEnd, rightEnd)
	if end < start {
		return 0
	}
	return end - start + 1
}

func relatedTablesForWindow(
	window []models.ParagraphNode,
	title string,
	outlineNode *models.DocumentOutlineNode,
	tables []models.TableSection,
) []models.TableSection {
	if len(window) == 0 || len(tables) == 0 {
		return nil
	}

	sectionKey := paragraphMetadataValue(window[0], "section_key")
	if sectionKey == "" && outlineNode != nil {
		sectionKey = outlineMetadataValue(outlineNode.Metadata, "section_key")
	}
	sectionTitle := paragraphMetadataValue(window[0], "section_title")
	if sectionTitle == "" {
		sectionTitle = strings.TrimSpace(title)
	}

	matches := make([]models.TableSection, 0, maxRelatedTablesPerChunk)
	for _, table := range tables {
		tableSectionKey := tableMetadataValue(table.Metadata, "section_key")
		switch {
		case sectionKey != "" && tableSectionKey == sectionKey:
			matches = append(matches, table)
		case sectionTitle != "" && strings.EqualFold(strings.TrimSpace(table.Title), sectionTitle):
			matches = append(matches, table)
		case strings.TrimSpace(title) != "" && strings.EqualFold(strings.TrimSpace(table.Title), strings.TrimSpace(title)):
			matches = append(matches, table)
		}

		if len(matches) == maxRelatedTablesPerChunk {
			break
		}
	}

	return matches
}

func relatedTableTitles(tables []models.TableSection) []string {
	titles := make([]string, 0, len(tables))
	for _, table := range tables {
		title := strings.TrimSpace(table.Title)
		if title != "" {
			titles = append(titles, title)
		}
	}
	return titles
}
