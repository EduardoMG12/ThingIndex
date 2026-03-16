package processing

import (
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const (
	documentChunkTypeParagraphWindow = "paragraph_window"
	documentChunkTypeTable           = "table_section"
	chunkStrategyVersion             = "tree-aware-v1"
	chunkTargetChars                 = 1200
	chunkOverlapParagraphs           = 1
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
		content := joinParagraphWindow(window)
		if strings.TrimSpace(content) != "" {
			chunkStart := window[0].SequenceOrder
			chunkEnd := window[len(window)-1].SequenceOrder
			title := chunkTitle(window, result.OutlineNodes, len(result.ParagraphNodes))
			pageStart, pageEnd := paragraphPageRange(window)

			metadata := map[string]any{
				"paragraph_count":        len(window),
				"chunk_strategy_version": chunkStrategyVersion,
				"outline_hint":           title,
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

func chunkTitle(window []models.ParagraphNode, outline []models.DocumentOutlineNode, paragraphCount int) string {
	if len(window) == 0 {
		return ""
	}

	if sectionTitle := paragraphMetadataValue(window[0], "section_title"); sectionTitle != "" {
		return sectionTitle
	}

	start := window[0].SequenceOrder
	end := window[len(window)-1].SequenceOrder
	return outlineTitleForParagraphRange(outline, paragraphCount, start, end)
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

func outlineTitleForParagraphRange(outline []models.DocumentOutlineNode, paragraphCount, start, end int) string {
	if len(outline) == 0 || paragraphCount <= 0 {
		return ""
	}

	midpoint := (start + end) / 2
	bucketSize := max(1, paragraphCount/len(outline))
	index := midpoint / bucketSize
	if index >= len(outline) {
		index = len(outline) - 1
	}

	return strings.TrimSpace(outline[index].Title)
}
