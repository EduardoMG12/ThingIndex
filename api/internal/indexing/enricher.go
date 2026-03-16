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
	Nodes []struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	} `json:"nodes"`
}

type pageSummaryResponse struct {
	Pages []struct {
		PageNumber int    `json:"page_number"`
		Summary    string `json:"summary"`
	} `json:"pages"`
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
	if treeBuilt, err := e.buildTree(ctx, llm, doc, updated); err == nil {
		updated = treeBuilt
	}

	contextBlock := buildPromptContext(doc, updated)

	synopsis, err := e.generateDocumentSynopsis(ctx, llm, contextBlock)
	if err != nil {
		return result, err
	}
	applyDocumentSynopsis(&updated, documentID, synopsis)

	sectionSummaries, err := e.generateOutlineNodeSummaries(ctx, llm, contextBlock)
	if err == nil {
		applyOutlineNodeSummaries(&updated, sectionSummaries)
	}

	pageSummaries, err := e.generatePageSummaries(ctx, llm, contextBlock)
	if err == nil {
		applyPageSummaries(&updated, pageSummaries)
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

	payload.DocumentSummary = clampText(payload.DocumentSummary, 800)
	payload.DocumentPurpose = clampText(payload.DocumentPurpose, 400)
	payload.Keywords = sanitizeStringList(payload.Keywords, 8, 80)

	return payload, nil
}

func (e *LLMEnricher) generateOutlineNodeSummaries(ctx context.Context, llm domain.LLMService, contextBlock string) (sectionSummaryResponse, error) {
	systemPrompt, userPrompt := buildOutlineNodeSummaryPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return sectionSummaryResponse{}, err
	}

	var payload sectionSummaryResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return sectionSummaryResponse{}, fmt.Errorf("parse section enrichment json: %w", err)
	}

	payload.Nodes = sanitizeSectionSummaries(payload.Nodes, llmOutlineLimit)

	return payload, nil
}

func (e *LLMEnricher) generatePageSummaries(ctx context.Context, llm domain.LLMService, contextBlock string) (pageSummaryResponse, error) {
	systemPrompt, userPrompt := buildPageSummaryPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return pageSummaryResponse{}, err
	}

	var payload pageSummaryResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return pageSummaryResponse{}, fmt.Errorf("parse page enrichment json: %w", err)
	}

	payload.Pages = sanitizePageSummaries(payload.Pages, 8)
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

	payload.Topics = sanitizeStringList(payload.Topics, 8, 80)
	payload.Entities = sanitizeStringList(payload.Entities, 12, 80)
	payload.Dates = sanitizeStringList(payload.Dates, 10, 40)

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

func applyOutlineNodeSummaries(result *domain.IndexingResult, payload sectionSummaryResponse) {
	if result == nil {
		return
	}

	for _, section := range payload.Nodes {
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

func applyPageSummaries(result *domain.IndexingResult, payload pageSummaryResponse) {
	if result == nil {
		return
	}

	for _, page := range payload.Pages {
		if page.PageNumber <= 0 {
			continue
		}
		for idx := range result.DocumentPages {
			if result.DocumentPages[idx].PageNumber != page.PageNumber {
				continue
			}
			result.DocumentPages[idx].Summary = strings.TrimSpace(page.Summary)
			break
		}
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
		SequenceOrder: nextIndexingMetadataSequence(result.MetadataPairs),
		Key:           key,
		Value:         strings.TrimSpace(value),
	})
}

func nextIndexingMetadataSequence(pairs []models.MetadataPair) int {
	maxSequence := -1
	for _, pair := range pairs {
		if pair.SequenceOrder > maxSequence {
			maxSequence = pair.SequenceOrder
		}
	}
	return maxSequence + 1
}

func documentIDFromResult(result domain.IndexingResult) uint {
	if len(result.DocumentPages) > 0 {
		return result.DocumentPages[0].DocumentID
	}
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
	items := make([]string, 0, min(8, len(keywords)))
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
		if len(items) == 8 {
			break
		}
	}
	return strings.Join(items, ", ")
}

func sanitizeStringList(values []string, limit, maxLen int) []string {
	items := make([]string, 0, min(limit, len(values)))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		cleaned := clampText(value, maxLen)
		if cleaned == "" {
			continue
		}

		normalizedKey := strings.ToLower(cleaned)
		if _, ok := seen[normalizedKey]; ok {
			continue
		}
		seen[normalizedKey] = struct{}{}
		items = append(items, cleaned)
		if len(items) == limit {
			break
		}
	}

	return items
}

func sanitizeSectionSummaries(
	sections []struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	},
	limit int,
) []struct {
	SequenceOrder int    `json:"sequence_order"`
	Summary       string `json:"summary"`
} {
	items := make([]struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	}, 0, min(limit, len(sections)))
	seen := make(map[int]struct{}, len(sections))
	for _, section := range sections {
		if section.SequenceOrder < 0 {
			continue
		}
		if _, ok := seen[section.SequenceOrder]; ok {
			continue
		}

		summary := clampText(section.Summary, 280)
		if summary == "" {
			continue
		}

		seen[section.SequenceOrder] = struct{}{}
		items = append(items, struct {
			SequenceOrder int    `json:"sequence_order"`
			Summary       string `json:"summary"`
		}{
			SequenceOrder: section.SequenceOrder,
			Summary:       summary,
		})
		if len(items) == limit {
			break
		}
	}

	return items
}

func sanitizePageSummaries(
	pages []struct {
		PageNumber int    `json:"page_number"`
		Summary    string `json:"summary"`
	},
	limit int,
) []struct {
	PageNumber int    `json:"page_number"`
	Summary    string `json:"summary"`
} {
	items := make([]struct {
		PageNumber int    `json:"page_number"`
		Summary    string `json:"summary"`
	}, 0, min(limit, len(pages)))
	seen := make(map[int]struct{}, len(pages))
	for _, page := range pages {
		if page.PageNumber <= 0 {
			continue
		}
		if _, ok := seen[page.PageNumber]; ok {
			continue
		}

		summary := clampText(page.Summary, 320)
		if summary == "" {
			continue
		}

		seen[page.PageNumber] = struct{}{}
		items = append(items, struct {
			PageNumber int    `json:"page_number"`
			Summary    string `json:"summary"`
		}{
			PageNumber: page.PageNumber,
			Summary:    summary,
		})
		if len(items) == limit {
			break
		}
	}

	return items
}

func clampText(value string, maxLen int) string {
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
