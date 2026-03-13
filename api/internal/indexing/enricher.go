package indexing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const llmOutlineLimit = 12

type LLMEnricher struct{}

type enrichmentResponse struct {
	DocumentSummary string   `json:"document_summary"`
	Keywords        []string `json:"keywords"`
	Sections        []struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	} `json:"sections"`
}

func NewLLMEnricher() *LLMEnricher {
	return &LLMEnricher{}
}

func (e *LLMEnricher) Enrich(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult) (domain.IndexingResult, error) {
	if llm == nil {
		return result, nil
	}

	systemPrompt, userPrompt := buildEnrichmentPrompt(doc, result)
	responseText, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return result, err
	}

	var payload enrichmentResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(responseText)), &payload); err != nil {
		return result, fmt.Errorf("parse llm enrichment json: %w", err)
	}

	updated := result
	if summary := strings.TrimSpace(payload.DocumentSummary); summary != "" {
		updated.MetadataPairs = append(updated.MetadataPairs, models.MetadataPair{
			DocumentID:    documentIDFromResult(result),
			SequenceOrder: nextMetadataSequence(updated.MetadataPairs),
			Key:           "document_summary",
			Value:         summary,
		})
	}

	if len(payload.Keywords) > 0 {
		keywords := normalizeKeywords(payload.Keywords)
		if keywords != "" {
			updated.MetadataPairs = append(updated.MetadataPairs, models.MetadataPair{
				DocumentID:    documentIDFromResult(result),
				SequenceOrder: nextMetadataSequence(updated.MetadataPairs),
				Key:           "keywords",
				Value:         keywords,
			})
		}
	}

	for _, section := range payload.Sections {
		if section.SequenceOrder < 0 || section.SequenceOrder >= len(updated.OutlineNodes) {
			continue
		}
		updated.OutlineNodes[section.SequenceOrder].Summary = strings.TrimSpace(section.Summary)
	}

	return updated, nil
}

func buildEnrichmentPrompt(doc *models.Document, result domain.IndexingResult) (string, string) {
	promptProfile := promptProfileForDocument(doc)

	systemPrompt := strings.TrimSpace(`
You enrich a structured document index.
Return only valid JSON.
Do not wrap the response in markdown fences.
Keep summaries factual, concise, and grounded only in the supplied content.
Do not invent sections, facts, numbers, or legal conclusions not present in the input.
`)

	var builder strings.Builder
	builder.WriteString("Produce JSON with fields document_summary, keywords, sections.\n")
	builder.WriteString("sections must be an array of objects with sequence_order and summary.\n")
	builder.WriteString("Summaries must be in Portuguese.\n\n")
	builder.WriteString("Document type guidance:\n")
	builder.WriteString(promptProfile)
	builder.WriteString("\n\n")
	if doc != nil {
		builder.WriteString("Document metadata:\n")
		builder.WriteString("- title=")
		builder.WriteString(doc.Title)
		builder.WriteString("\n- mime_type=")
		builder.WriteString(doc.MimeType)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Outline:\n")
	for idx, node := range result.OutlineNodes {
		if idx >= llmOutlineLimit {
			break
		}
		builder.WriteString("- sequence_order=")
		builder.WriteString(strconv.Itoa(node.SequenceOrder))
		builder.WriteString(", level=")
		builder.WriteString(strconv.Itoa(node.Level))
		builder.WriteString(", title=")
		builder.WriteString(node.Title)
		builder.WriteString("\n")
	}

	builder.WriteString("\nParagraph samples:\n")
	for idx, node := range result.ParagraphNodes {
		if idx >= llmOutlineLimit {
			break
		}
		builder.WriteString("- ")
		builder.WriteString(node.Content)
		builder.WriteString("\n")
	}

	if len(result.TableSections) > 0 {
		builder.WriteString("\nTables:\n")
		for idx, table := range result.TableSections {
			if idx >= 4 {
				break
			}
			builder.WriteString("- title=")
			builder.WriteString(table.Title)
			builder.WriteString(", headers=")
			builder.WriteString(strings.Join(table.Headers, ", "))
			builder.WriteString("\n")
		}
	}

	return systemPrompt, builder.String()
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
	return 0
}

func normalizeKeywords(keywords []string) string {
	items := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			items = append(items, keyword)
		}
	}
	return strings.Join(items, ", ")
}
