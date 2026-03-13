package indexing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const llmOutlineLimit = 12

type LLMEnricher struct{}

type documentSynopsisResponse struct {
	DocumentSummary string   `json:"document_summary"`
	Keywords        []string `json:"keywords"`
	DocumentPurpose string   `json:"document_purpose"`
}

type sectionSummaryResponse struct {
	Sections []struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	} `json:"sections"`
}

type retrievalSignalsResponse struct {
	Topics   []string `json:"topics"`
	Entities []string `json:"entities"`
	Dates    []string `json:"dates"`
}

func NewLLMEnricher() *LLMEnricher {
	return &LLMEnricher{}
}

func (e *LLMEnricher) Enrich(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult) (domain.IndexingResult, error) {
	if llm == nil {
		return result, nil
	}

	updated := result
	documentID := documentIDFromResult(result)
	contextBlock := buildPromptContext(doc, result)

	synopsis, err := e.generateDocumentSynopsis(ctx, llm, contextBlock)
	if err != nil {
		return result, err
	}
	applyDocumentSynopsis(&updated, documentID, synopsis)

	sectionSummaries, err := e.generateSectionSummaries(ctx, llm, contextBlock)
	if err == nil {
		applySectionSummaries(&updated, sectionSummaries)
	}

	retrievalSignals, err := e.generateRetrievalSignals(ctx, llm, contextBlock)
	if err == nil {
		applyRetrievalSignals(&updated, documentID, retrievalSignals)
	}

	return updated, nil
}

func (e *LLMEnricher) generateDocumentSynopsis(ctx context.Context, llm domain.LLMService, contextBlock string) (documentSynopsisResponse, error) {
	systemPrompt, userPrompt := buildDocumentSynopsisPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return documentSynopsisResponse{}, err
	}

	var payload documentSynopsisResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return documentSynopsisResponse{}, fmt.Errorf("parse synopsis enrichment json: %w", err)
	}

	return payload, nil
}

func (e *LLMEnricher) generateSectionSummaries(ctx context.Context, llm domain.LLMService, contextBlock string) (sectionSummaryResponse, error) {
	systemPrompt, userPrompt := buildSectionSummaryPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return sectionSummaryResponse{}, err
	}

	var payload sectionSummaryResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return sectionSummaryResponse{}, fmt.Errorf("parse section enrichment json: %w", err)
	}

	return payload, nil
}

func (e *LLMEnricher) generateRetrievalSignals(ctx context.Context, llm domain.LLMService, contextBlock string) (retrievalSignalsResponse, error) {
	systemPrompt, userPrompt := buildRetrievalSignalsPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return retrievalSignalsResponse{}, err
	}

	var payload retrievalSignalsResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return retrievalSignalsResponse{}, fmt.Errorf("parse retrieval signal json: %w", err)
	}

	return payload, nil
}

func parseLLMJSON(raw string, target any) error {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	return json.Unmarshal([]byte(cleaned), target)
}

func applyDocumentSynopsis(result *domain.IndexingResult, documentID uint, payload documentSynopsisResponse) {
	if result == nil {
		return
	}

	appendMetadataIfPresent(result, documentID, "document_summary", payload.DocumentSummary)

	if keywords := normalizeKeywords(payload.Keywords); keywords != "" {
		appendMetadata(result, documentID, "keywords", keywords)
	}

	appendMetadataIfPresent(result, documentID, "document_purpose", payload.DocumentPurpose)
}

func applySectionSummaries(result *domain.IndexingResult, payload sectionSummaryResponse) {
	if result == nil {
		return
	}

	for _, section := range payload.Sections {
		if section.SequenceOrder < 0 || section.SequenceOrder >= len(result.OutlineNodes) {
			continue
		}
		summary := strings.TrimSpace(section.Summary)
		if summary == "" {
			continue
		}
		result.OutlineNodes[section.SequenceOrder].Summary = summary
	}
}

func applyRetrievalSignals(result *domain.IndexingResult, documentID uint, payload retrievalSignalsResponse) {
	if result == nil {
		return
	}

	appendMetadataIfPresent(result, documentID, "topics", normalizeKeywords(payload.Topics))
	appendMetadataIfPresent(result, documentID, "entities", normalizeKeywords(payload.Entities))
	appendMetadataIfPresent(result, documentID, "dates", normalizeKeywords(payload.Dates))
}

func appendMetadataIfPresent(result *domain.IndexingResult, documentID uint, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	appendMetadata(result, documentID, key, value)
}

func appendMetadata(result *domain.IndexingResult, documentID uint, key, value string) {
	result.MetadataPairs = append(result.MetadataPairs, models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: nextMetadataSequence(result.MetadataPairs),
		Key:           key,
		Value:         strings.TrimSpace(value),
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

func documentIDFromResult(result domain.IndexingResult) uint {
	if len(result.ParagraphNodes) > 0 {
		return result.ParagraphNodes[0].DocumentID
	}
	if len(result.TableSections) > 0 {
		return result.TableSections[0].DocumentID
	}
	if len(result.MetadataPairs) > 0 {
		return result.MetadataPairs[0].DocumentID
	}
	if len(result.OutlineNodes) > 0 {
		return result.OutlineNodes[0].DocumentID
	}
	if len(result.Chunks) > 0 {
		return result.Chunks[0].DocumentID
	}
	return 0
}

func normalizeKeywords(keywords []string) string {
	items := make([]string, 0, len(keywords))
	seen := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		normalizedKey := strings.ToLower(keyword)
		if _, ok := seen[normalizedKey]; ok {
			continue
		}
		seen[normalizedKey] = struct{}{}
		items = append(items, keyword)
	}
	return strings.Join(items, ", ")
}
