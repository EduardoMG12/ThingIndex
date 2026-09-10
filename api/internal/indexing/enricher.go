package indexing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	} else {
		if len(updated.DocumentPages) > 0 {
			verifiedNodes, stats := verifyAndScoreTree(updated, updated.OutlineNodes, "deterministic")
			updated.OutlineNodes = verifiedNodes
			if shouldFallback, reason := shouldUseConservativeTreeFallback(stats); shouldFallback {
				updated, stats = applyConservativeTreeFallback(updated, reason)
			}
			appendTreeVerificationMetadata(&updated, documentID, stats)
		} else {
			appendMetadata(&updated, documentID, "tree_builder", "deterministic")
			appendMetadata(&updated, documentID, "tree_verification_passed", "false")
		}
	}

	contextBlock := buildPromptContext(doc, updated)

	// Fallback nunca e silencioso: sem o log, cinco chamadas devolvendo um
	// token cada passaram meses como "enrichment_status=applied".
	synopsis, err := e.generateDocumentSynopsis(ctx, llm, contextBlock)
	if err != nil {
		log.Printf("enricher: sinopse do documento %d caiu no fallback: %v", documentID, err)
		synopsis = buildFallbackDocumentSynopsis(doc, updated)
		appendMetadata(&updated, documentID, "synopsis_source", "fallback")
	} else {
		appendMetadata(&updated, documentID, "synopsis_source", "llm")
	}
	applyDocumentSynopsis(&updated, documentID, synopsis)
	sectionSummaries, err := e.generateOutlineNodeSummaries(ctx, llm, contextBlock)
	if err == nil {
		applyOutlineNodeSummaries(&updated, sectionSummaries)
	} else {
		log.Printf("enricher: resumos de secao do documento %d descartados: %v", documentID, err)
	}
	pageSummaries, err := e.generatePageSummaries(ctx, llm, contextBlock)
	if err == nil {
		applyPageSummaries(&updated, pageSummaries)
	} else {
		log.Printf("enricher: resumos de pagina do documento %d descartados: %v", documentID, err)
	}
	fillMissingPageSummaries(&updated)
	populateOutlineRetrievalText(&updated)

	retrievalSignals, err := e.generateRetrievalSignals(ctx, llm, contextBlock)
	if err == nil {
		applyRetrievalSignals(&updated, documentID, retrievalSignals)
	} else {
		log.Printf("enricher: sinais de recuperacao do documento %d descartados: %v", documentID, err)
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

	if err := json.Unmarshal([]byte(cleaned), target); err == nil {
		return nil
	}

	extracted := extractJSONEnvelope(cleaned)
	if extracted == "" {
		return json.Unmarshal([]byte(cleaned), target)
	}

	return json.Unmarshal([]byte(extracted), target)
}

func extractJSONEnvelope(raw string) string {
	start := -1
	for idx, r := range raw {
		if r == '{' || r == '[' {
			start = idx
			break
		}
	}
	if start < 0 {
		return ""
	}

	var (
		depth       int
		inString    bool
		escapeNext  bool
		openingRune rune
		closingRune rune
	)

	for idx, r := range raw[start:] {
		absoluteIdx := start + idx
		if openingRune == 0 {
			openingRune = r
			if openingRune == '{' {
				closingRune = '}'
			} else {
				closingRune = ']'
			}
		}

		if inString {
			if escapeNext {
				escapeNext = false
				continue
			}
			if r == '\\' {
				escapeNext = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}

		switch r {
		case '"':
			inString = true
		case openingRune:
			depth++
		case closingRune:
			depth--
			if depth == 0 {
				return strings.TrimSpace(raw[start : absoluteIdx+1])
			}
		}
	}

	return ""
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

func buildFallbackDocumentSynopsis(doc *models.Document, result domain.IndexingResult) documentSynopsisResponse {
	summary := firstNonEmptyNonBlank(
		firstNonEmptyPageContent(result.DocumentPages),
		firstNonEmptyParagraphContent(result.ParagraphNodes),
		titleFromDocument(doc),
	)
	purpose := firstNonEmptyNonBlank(
		firstNonEmptyOutlineTitle(result.OutlineNodes),
		summary,
		titleFromDocument(doc),
	)

	return documentSynopsisResponse{
		DocumentSummary: clampText(summary, 800),
		DocumentPurpose: clampText(purpose, 400),
		Keywords:        fallbackSynopsisKeywords(doc, result),
	}
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

func fillMissingPageSummaries(result *domain.IndexingResult) {
	if result == nil {
		return
	}

	for idx := range result.DocumentPages {
		if strings.TrimSpace(result.DocumentPages[idx].Summary) != "" {
			continue
		}
		result.DocumentPages[idx].Summary = fallbackPageSummary(result.DocumentPages[idx].Content)
	}
}

func fallbackPageSummary(content string) string {
	return clampText(content, 320)
}

func fallbackSynopsisKeywords(doc *models.Document, result domain.IndexingResult) []string {
	candidates := []string{}

	if title := strings.TrimSpace(titleFromDocument(doc)); title != "" {
		candidates = append(candidates, title)
	}
	for _, node := range result.OutlineNodes {
		if title := strings.TrimSpace(node.Title); title != "" {
			candidates = append(candidates, title)
		}
	}

	return sanitizeStringList(candidates, 8, 80)
}

func firstNonEmptyPageContent(pages []models.DocumentPage) string {
	for _, page := range pages {
		if value := strings.TrimSpace(page.Content); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyParagraphContent(paragraphs []models.ParagraphNode) string {
	for _, paragraph := range paragraphs {
		if value := strings.TrimSpace(paragraph.Content); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyOutlineTitle(nodes []models.DocumentOutlineNode) string {
	for _, node := range nodes {
		if value := strings.TrimSpace(node.Title); value != "" {
			return value
		}
	}
	return ""
}

func titleFromDocument(doc *models.Document) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(doc.Title)
}

func firstNonEmptyNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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

	truncated := string(runes[:maxLen])
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		return truncated[:lastSpace]
	}
	return strings.TrimSpace(truncated)
}
