package processing

import (
	"strconv"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const (
	indexPipelineVersion = "page-tree-v3"
	promptProfileVersion = "llm-index-v5"
)

type PipelineSignature struct {
	PipelineVersion      string
	PromptProfileVersion string
	ChunkStrategyVersion string
}

func CurrentPipelineSignature() PipelineSignature {
	return PipelineSignature{
		PipelineVersion:      indexPipelineVersion,
		PromptProfileVersion: promptProfileVersion,
		ChunkStrategyVersion: chunkStrategyVersion,
	}
}

func appendPipelineMetadata(result *domain.IndexingResult, documentID uint, enrichmentStatus string) {
	if result == nil {
		return
	}

	appendMetadataPair(result, documentID, "index_pipeline_version", indexPipelineVersion)
	appendMetadataPair(result, documentID, "chunk_strategy_version", chunkStrategyVersion)
	appendMetadataPair(result, documentID, "prompt_profile_version", promptProfileVersion)
	appendMetadataPair(result, documentID, "enrichment_status", strings.TrimSpace(enrichmentStatus))
}

func appendLLMUsageMetadata(result *domain.IndexingResult, documentID uint, usage llmUsageSnapshot) {
	if result == nil || documentID == 0 {
		return
	}

	appendMetadataPair(result, documentID, "llm_request_count", strconv.Itoa(usage.RequestCount))
	appendMetadataPair(result, documentID, "llm_failed_request_count", strconv.Itoa(usage.FailedRequestCount))
	appendMetadataPair(result, documentID, "llm_input_tokens_estimate", strconv.Itoa(usage.EstimatedInputTokens))
	appendMetadataPair(result, documentID, "llm_output_tokens_estimate", strconv.Itoa(usage.EstimatedOutputTokens))
	appendMetadataPair(result, documentID, "llm_total_tokens_estimate", strconv.Itoa(usage.EstimatedTotalTokens))
	appendMetadataPair(result, documentID, "llm_pricing_available", strconv.FormatBool(usage.PricingAvailable))

	if strings.TrimSpace(usage.ProviderType) != "" {
		appendMetadataPair(result, documentID, "llm_provider_type", usage.ProviderType)
	}
	if strings.TrimSpace(usage.Model) != "" {
		appendMetadataPair(result, documentID, "llm_model", usage.Model)
	}
	if usage.PricingAvailable {
		appendMetadataPair(result, documentID, "llm_input_cost_usd", formatUSD(usage.InputCostUSD))
		appendMetadataPair(result, documentID, "llm_output_cost_usd", formatUSD(usage.OutputCostUSD))
		appendMetadataPair(result, documentID, "llm_total_cost_usd", formatUSD(usage.TotalCostUSD))
	}
}

func appendProcessingTimingMetadata(result *domain.IndexingResult, documentID uint, startedAt, completedAt time.Time) {
	if result == nil || documentID == 0 {
		return
	}

	appendMetadataPair(result, documentID, "index_started_at", startedAt.UTC().Format(time.RFC3339))
	appendMetadataPair(result, documentID, "index_completed_at", completedAt.UTC().Format(time.RFC3339))
	appendMetadataPair(result, documentID, "index_duration_ms", strconv.FormatInt(completedAt.Sub(startedAt).Milliseconds(), 10))
}

func formatUSD(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
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
