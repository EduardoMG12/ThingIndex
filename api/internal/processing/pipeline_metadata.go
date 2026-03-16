package processing

import (
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const (
	indexPipelineVersion = "page-tree-v1"
	promptProfileVersion = "llm-index-v3"
)

func appendPipelineMetadata(result *domain.IndexingResult, documentID uint, enrichmentStatus string) {
	if result == nil {
		return
	}

	appendMetadataPair(result, documentID, "index_pipeline_version", indexPipelineVersion)
	appendMetadataPair(result, documentID, "chunk_strategy_version", chunkStrategyVersion)
	appendMetadataPair(result, documentID, "prompt_profile_version", promptProfileVersion)
	appendMetadataPair(result, documentID, "enrichment_status", strings.TrimSpace(enrichmentStatus))
}

func appendMetadataPair(result *domain.IndexingResult, documentID uint, key, value string) {
	value = strings.TrimSpace(value)
	if result == nil || documentID == 0 || key == "" || value == "" {
		return
	}

	result.MetadataPairs = append(result.MetadataPairs, models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: nextMetadataSequence(result.MetadataPairs),
		Key:           key,
		Value:         value,
	})
}

func nextMetadataSequence(pairs []models.MetadataPair) int {
	maxSequence := -1
	for _, pair := range pairs {
		if pair.SequenceOrder > maxSequence {
			maxSequence = pair.SequenceOrder
		}
	}
	return maxSequence + 1
}
