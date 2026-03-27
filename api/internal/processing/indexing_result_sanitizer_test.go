package processing

import (
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestSanitizeIndexingResultClampsStructuredTitles(t *testing.T) {
	longTitle := strings.Repeat("A", 400)
	longPath := strings.Repeat("node/", 300)

	result := domain.IndexingResult{
		TableSections: []models.TableSection{
			{Title: longTitle},
		},
		Chunks: []models.DocumentChunk{
			{Title: longTitle},
		},
		OutlineNodes: []models.DocumentOutlineNode{
			{Title: longTitle, Slug: longTitle, NodePath: longPath},
		},
	}

	sanitizeIndexingResult(&result)

	if got := len([]rune(result.TableSections[0].Title)); got > maxStructuredTitleLength {
		t.Fatalf("table title not clamped: %d", got)
	}
	if got := len([]rune(result.Chunks[0].Title)); got > maxStructuredTitleLength {
		t.Fatalf("chunk title not clamped: %d", got)
	}
	if got := len([]rune(result.OutlineNodes[0].Title)); got > maxStructuredTitleLength {
		t.Fatalf("outline title not clamped: %d", got)
	}
	if got := len([]rune(result.OutlineNodes[0].Slug)); got > maxStructuredTitleLength {
		t.Fatalf("outline slug not clamped: %d", got)
	}
	if got := len([]rune(result.OutlineNodes[0].NodePath)); got > maxOutlinePathLength {
		t.Fatalf("outline node path not clamped: %d", got)
	}
}
