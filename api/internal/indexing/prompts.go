package indexing

import (
	"strings"

	"thing-index/internal/models"
)

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
