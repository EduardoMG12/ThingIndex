package processing

import (
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const (
	documentChunkTypeParagraphWindow = "paragraph_window"
	chunkTargetChars                 = 1200
	chunkOverlapParagraphs           = 1
)

func buildDocumentChunks(documentID uint, result domain.IndexingResult) []models.DocumentChunk {
	if len(result.ParagraphNodes) == 0 {
		return []models.DocumentChunk{}
	}

	chunks := make([]models.DocumentChunk, 0)
	start := 0
	for start < len(result.ParagraphNodes) {
		end := start
		charCount := 0

		for end < len(result.ParagraphNodes) {
			paragraph := strings.TrimSpace(result.ParagraphNodes[end].Content)
			if paragraph == "" {
				end++
				continue
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
			end++
		}

		window := result.ParagraphNodes[start:end]
		content := joinParagraphWindow(window)
		if strings.TrimSpace(content) != "" {
			chunkStart := window[0].SequenceOrder
			chunkEnd := window[len(window)-1].SequenceOrder

			chunks = append(chunks, models.DocumentChunk{
				DocumentID:           documentID,
				SequenceOrder:        len(chunks),
				ChunkType:            documentChunkTypeParagraphWindow,
				Content:              content,
				SourceParagraphStart: &chunkStart,
				SourceParagraphEnd:   &chunkEnd,
				TokenEstimate:        estimateTokenCount(content),
				Metadata: map[string]any{
					"paragraph_count": len(window),
				},
			})
		}

		if end >= len(result.ParagraphNodes) {
			break
		}

		nextStart := end - chunkOverlapParagraphs
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	return chunks
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
