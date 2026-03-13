package indexing

import (
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func buildDocumentSynopsisPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Ground every field strictly in the supplied document content.
Do not invent facts, sections, parties, dates, obligations, or conclusions.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: create a compact document synopsis for later retrieval and inspection.\n")
	userPrompt.WriteString("Return JSON with fields document_summary, keywords, document_purpose.\n")
	userPrompt.WriteString("document_summary and document_purpose must be in Portuguese.\n")
	userPrompt.WriteString("keywords must be a short array of concrete retrieval terms.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildSectionSummaryPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Summarize only sections that are actually evidenced by the supplied outline and text.
If evidence is weak, leave the section out instead of guessing.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: summarize indexed sections for navigation and grounded retrieval.\n")
	userPrompt.WriteString("Return JSON with field sections.\n")
	userPrompt.WriteString("sections must be an array of objects with sequence_order and summary.\n")
	userPrompt.WriteString("Each summary must be concise, factual, and in Portuguese.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildRetrievalSignalsPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Extract only signals that improve retrieval later.
Prefer concrete nouns, entities, dates, and operational topics over generic abstractions.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: extract retrieval signals from the indexed document.\n")
	userPrompt.WriteString("Return JSON with fields topics, entities, dates.\n")
	userPrompt.WriteString("Each field must be an array of short strings grounded in the content.\n")
	userPrompt.WriteString("If a field has no reliable evidence, return an empty array.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildPromptContext(doc *models.Document, result domain.IndexingResult) string {
	promptProfile := promptProfileForDocument(doc)

	var builder strings.Builder
	builder.WriteString("Document type guidance:\n")
	builder.WriteString(promptProfile)
	builder.WriteString("\n\n")

	if doc != nil {
		builder.WriteString("Document metadata:\n")
		builder.WriteString("- title=")
		builder.WriteString(doc.Title)
		builder.WriteString("\n- mime_type=")
		builder.WriteString(doc.MimeType)
		builder.WriteString("\n")
		if doc.CollectionRef != nil {
			builder.WriteString("- collection=")
			builder.WriteString(doc.CollectionRef.Name)
			builder.WriteString("\n")
			if doc.CollectionRef.Category != nil {
				builder.WriteString("- category=")
				builder.WriteString(doc.CollectionRef.Category.Name)
				builder.WriteString("\n")
			}
		}
		builder.WriteString("\n")
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
		builder.WriteString("- sequence_order=")
		builder.WriteString(strconv.Itoa(node.SequenceOrder))
		builder.WriteString(", content=")
		builder.WriteString(node.Content)
		builder.WriteString("\n")
	}

	if len(result.Chunks) > 0 {
		builder.WriteString("\nChunk samples:\n")
		for idx, chunk := range result.Chunks {
			if idx >= 6 {
				break
			}
			builder.WriteString("- sequence_order=")
			builder.WriteString(strconv.Itoa(chunk.SequenceOrder))
			builder.WriteString(", chunk_type=")
			builder.WriteString(chunk.ChunkType)
			if chunk.SourceParagraphStart != nil && chunk.SourceParagraphEnd != nil {
				builder.WriteString(", source_paragraph_range=")
				builder.WriteString(strconv.Itoa(*chunk.SourceParagraphStart))
				builder.WriteString("-")
				builder.WriteString(strconv.Itoa(*chunk.SourceParagraphEnd))
			}
			builder.WriteString(", content=")
			builder.WriteString(chunk.Content)
			builder.WriteString("\n")
		}
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
			if len(table.Rows) > 0 {
				builder.WriteString(", first_row=")
				builder.WriteString(strings.Join(table.Rows[0], ", "))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func promptProfileForDocument(doc *models.Document) string {
	if doc == nil {
		return genericPromptProfile()
	}

	switch {
	case doc.MimeType == "application/pdf":
		return pdfPromptProfile(doc)
	case doc.MimeType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return spreadsheetPromptProfile()
	case doc.MimeType == "text/html":
		return htmlPromptProfile()
	default:
		return genericPromptProfile()
	}
}

func pdfPromptProfile(doc *models.Document) string {
	lowerTitle := strings.ToLower(strings.TrimSpace(doc.Title))
	switch {
	case strings.Contains(lowerTitle, "contrat") || strings.Contains(lowerTitle, "acordo"):
		return "Focus on obligations, parties, deadlines, conditions, exceptions, penalties, and clauses that structure the agreement. Prefer operational language over marketing language."
	case strings.Contains(lowerTitle, "peti") || strings.Contains(lowerTitle, "contest") || strings.Contains(lowerTitle, "senten") || strings.Contains(lowerTitle, "acorda"):
		return "Focus on procedural posture, parties, requests, defenses, cited facts, dates, legal grounds, and outcomes. Keep summaries neutral and auditable."
	default:
		return "Focus on section hierarchy, key facts, decision points, deadlines, definitions, and operational content. Prefer traceable summaries over interpretation."
	}
}

func spreadsheetPromptProfile() string {
	return "Focus on what each sheet represents, the business meaning of columns, repeated dimensions, metrics, totals, and notable patterns visible from headers and sampled rows."
}

func htmlPromptProfile() string {
	return "Focus on page hierarchy, headings, navigational sections, main content blocks, tables, and the purpose of each section. Ignore ornamental boilerplate unless it changes meaning."
}

func genericPromptProfile() string {
	return "Focus on section hierarchy, major topics, key entities, dates, metrics, and explicit relationships that help retrieve the document later."
}
