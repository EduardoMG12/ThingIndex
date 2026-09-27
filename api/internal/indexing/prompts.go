package indexing

import (
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func buildDocumentIdentityBlock(doc *models.Document) string {
	var builder strings.Builder
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
	return builder.String()
}

func buildDocumentSynopsisPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Ground every field strictly in the supplied document content.
Do not invent facts, sections, parties, dates, obligations, or conclusions.
Keep outputs compact, auditable, and useful for later retrieval.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: create a compact document synopsis for later retrieval and inspection.\n")
	userPrompt.WriteString("Return JSON with fields document_summary, keywords, document_purpose.\n")
	userPrompt.WriteString("document_summary and document_purpose must be in Portuguese.\n")
	userPrompt.WriteString("document_summary must be at most 200 words.\n")
	userPrompt.WriteString("document_purpose must be at most 60 words.\n")
	userPrompt.WriteString("keywords must be a short array of concrete retrieval terms with at most 8 items.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildTreeConstructionPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You construct a hierarchical page-grounded index for a document.
Return only valid JSON.
Do not wrap the response in markdown fences.
Every node must be grounded in the supplied page content.
Prefer a compact, auditable tree over a large noisy tree.
Children must stay inside the page range of their parent.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: build a tree index for this document.\n")
	userPrompt.WriteString("Return JSON with field nodes.\n")
	userPrompt.WriteString("nodes must be an array of objects with node_id, parent_node_id, title, level, start_page, end_page, summary.\n")
	userPrompt.WriteString("Use Portuguese for summaries.\n")
	userPrompt.WriteString("Keep summaries under 40 words.\n")
	userPrompt.WriteString("If a node has no parent, use an empty string for parent_node_id.\n")
	userPrompt.WriteString("Prefer 3 to 10 nodes unless the document is clearly very small.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildSubtreeExpansionPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You expand one oversized tree node into grounded child nodes.
Return only valid JSON.
Do not wrap the response in markdown fences.
Every child must stay inside the supplied parent page range.
Prefer a small set of concrete children over speculative structure.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: expand the supplied parent node into child nodes.\n")
	userPrompt.WriteString("Return JSON with field nodes.\n")
	userPrompt.WriteString("nodes must be an array of objects with node_id, parent_node_id, title, level, start_page, end_page, summary.\n")
	userPrompt.WriteString("Each child must use the supplied parent node id as parent_node_id.\n")
	userPrompt.WriteString("Use Portuguese for summaries and keep them under 40 words.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildTreeFixPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You repair a page-grounded tree index that failed verification.
Return only valid JSON.
Do not wrap the response in markdown fences.
Fix page ranges, hierarchy issues, and low-coverage structure using only the supplied pages.
Prefer a smaller correct tree over a larger incorrect tree.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: repair the supplied tree index.\n")
	userPrompt.WriteString("Return JSON with field nodes.\n")
	userPrompt.WriteString("nodes must be an array of objects with node_id, parent_node_id, title, level, start_page, end_page, summary.\n")
	userPrompt.WriteString("Keep summaries in Portuguese and under 40 words.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildOutlineNodeSummaryPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Summarize every node listed, using only the text supplied for it.
When the text is short, say what the title and the available text state; never invent facts.
Prefer navigation-grade summaries, page-grounded and auditable.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: summarize indexed tree nodes for navigation and grounded retrieval.\n")
	userPrompt.WriteString("Return JSON with field nodes.\n")
	userPrompt.WriteString("nodes must be an array of objects with sequence_order and summary.\n")
	userPrompt.WriteString("Each summary must be factual, in Portuguese, and between 25 and 60 words: say what the section decides or describes, with the numbers, names and deadlines it contains.\n")
	userPrompt.WriteString("Do not repeat the node title inside the summary.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildPageSummaryPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Summarize only page content that is explicitly present in the supplied page text.
Keep the summaries useful for tree navigation and later evidence lookup.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: summarize indexed pages for grounded retrieval.\n")
	userPrompt.WriteString("Return JSON with field pages.\n")
	userPrompt.WriteString("pages must be an array of objects with page_number, summary and topics.\n")
	userPrompt.WriteString("Each summary must be factual, in Portuguese, and between 25 and 60 words: say what the page contains, with the numbers, names, dates and rules present in it.\n")
	userPrompt.WriteString("topics must be an array of at most 5 short noun phrases in Portuguese naming the subjects of the page.\n\n")
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
Avoid generic labels such as documento, contrato, planilha, secao, conteudo.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: extract retrieval signals from the indexed document.\n")
	userPrompt.WriteString("Return JSON with fields topics, entities, dates.\n")
	userPrompt.WriteString("Each field must be an array of short strings grounded in the content.\n")
	userPrompt.WriteString("Use at most 8 topics, 12 entities, and 10 dates.\n")
	userPrompt.WriteString("If a field has no reliable evidence, return an empty array.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}

func buildPromptContext(doc *models.Document, result domain.IndexingResult) string {
	promptProfile := promptProfileForDocument(doc)

	var builder strings.Builder
	builder.WriteString("Document type guidance:\n")
	builder.WriteString(promptProfile)
	builder.WriteString("\nPipeline versions:\n- chunk_strategy=section-window-v3\n- prompt_profile=llm-index-v5")
	builder.WriteString("\n\n")

	builder.WriteString(buildDocumentIdentityBlock(doc))

	builder.WriteString("Outline:\n")
	for idx, node := range result.OutlineNodes {
		if idx >= llmOutlineLimit {
			break
		}
		builder.WriteString("- sequence_order=")
		builder.WriteString(strconv.Itoa(node.SequenceOrder))
		builder.WriteString(", level=")
		builder.WriteString(strconv.Itoa(node.Level))
		if node.StartPage != nil {
			builder.WriteString(", start_page=")
			builder.WriteString(strconv.Itoa(*node.StartPage))
		}
		if node.EndPage != nil {
			builder.WriteString(", end_page=")
			builder.WriteString(strconv.Itoa(*node.EndPage))
		}
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

	if len(result.DocumentPages) > 0 {
		builder.WriteString("\nPages:\n")
		for idx, page := range result.DocumentPages {
			if idx >= 8 {
				break
			}
			builder.WriteString("- page_number=")
			builder.WriteString(strconv.Itoa(page.PageNumber))
			builder.WriteString(", content=")
			builder.WriteString(page.Content)
			builder.WriteString("\n")
		}
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
	lowerTitle := normalizeHeadingLexicon(doc.Title)
	switch {
	case strings.Contains(lowerTitle, "contrat") || strings.Contains(lowerTitle, "acordo"):
		return "Focus on obligations, parties, deadlines, conditions, exceptions, penalties, and clauses that structure the agreement. Prefer operational language over marketing language."
	case containsAnySubstring(lowerTitle, "peti", "contest", "recurso") ||
		containsHeadingKeywordFamily(lowerTitle, legalDocumentRootHeadingKeywords):
		return "Focus on procedural posture, parties, requests, defenses, cited facts, dates, legal grounds, and outcomes. Keep summaries neutral and auditable."
	case containsHeadingKeywordFamily(lowerTitle, attachmentHeadingKeywords):
		return "Focus on the role of the annex, referenced parties or case identifiers, tabular fields, values, dates, and any official instructions that explain how the attachment supports the main document."
	default:
		return "Focus on section hierarchy, key facts, decision points, deadlines, definitions, and operational content. Prefer traceable summaries over interpretation."
	}
}

func containsHeadingKeywordFamily(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(value, keyword) {
			return true
		}
	}
	return false
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

// buildPageQuestionsPrompt gera as perguntas que cada pagina responde, em
// linguagem de leigo.
//
// O texto deste prompt esta CONGELADO pelo pre-registro em
// docs/prds/pre-registro-perguntas-geradas.md e travado por teste. Muda-lo
// invalida a medicao: o compromisso registrado e rodar uma versao so, porque
// rodar varias e reportar a melhor produz um numero favoravel e sem valor.
//
// A diferenca em relacao ao prompt de resumo e o ponto inteiro do experimento.
// O de resumo manda dizer o que a pagina contem "with the numbers, names, dates
// and rules present in it", ou seja, pede para REUTILIZAR o vocabulario da
// pagina. Este pede o contrario: a palavra que a pessoa usaria, justamente
// porque um indice lexical so consegue usar termo que ainda nao esta no texto.
func buildPageQuestionsPrompt(contextBlock string) (string, string) {
	systemPrompt := strings.TrimSpace(`
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Write only questions that the supplied page text actually answers.
Write them in the everyday Portuguese of a person who has NOT read the document.
Do not reuse the technical vocabulary of the page when a common word exists.
`)

	var userPrompt strings.Builder
	userPrompt.WriteString("Task: write the questions each indexed page answers, for retrieval by everyday vocabulary.\n")
	userPrompt.WriteString("Return JSON with field pages.\n")
	userPrompt.WriteString("pages must be an array of objects with page_number and questions.\n")
	userPrompt.WriteString("questions must be an array of 3 to 5 short questions in Portuguese, each one a question a non-specialist would actually type.\n")
	userPrompt.WriteString("Each question must be answerable by the page text alone.\n")
	userPrompt.WriteString("Prefer the everyday word over the technical term: write \"quanto tempo\" instead of \"prazo decadencial\", \"quem paga\" instead of \"responsabilidade pelo custeio\".\n")
	userPrompt.WriteString("Do not number the questions and do not answer them.\n\n")
	userPrompt.WriteString(contextBlock)

	return systemPrompt, userPrompt.String()
}
