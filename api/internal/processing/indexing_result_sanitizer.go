package processing

import (
	"strings"

	"thing-index/internal/domain"
)

const (
	maxStructuredTitleLength = 255
	maxOutlinePathLength     = 1024
)

func sanitizeIndexingResult(result *domain.IndexingResult) {
	if result == nil {
		return
	}

	for idx := range result.TableSections {
		result.TableSections[idx].Title = clampStructuredText(result.TableSections[idx].Title, maxStructuredTitleLength)
	}

	for idx := range result.Chunks {
		result.Chunks[idx].Title = clampStructuredText(result.Chunks[idx].Title, maxStructuredTitleLength)
	}

	for idx := range result.OutlineNodes {
		result.OutlineNodes[idx].Title = clampStructuredText(result.OutlineNodes[idx].Title, maxStructuredTitleLength)
		result.OutlineNodes[idx].Slug = clampStructuredText(result.OutlineNodes[idx].Slug, maxStructuredTitleLength)
		result.OutlineNodes[idx].NodePath = clampStructuredText(result.OutlineNodes[idx].NodePath, maxOutlinePathLength)
	}
}

func clampStructuredText(value string, maxLen int) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if cleaned == "" || maxLen <= 0 {
		return ""
	}

	runes := []rune(cleaned)
	if len(runes) <= maxLen {
		return cleaned
	}

	return strings.TrimSpace(string(runes[:maxLen]))
}
