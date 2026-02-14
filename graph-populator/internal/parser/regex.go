package parser

import (
	"regexp"
	"strings"
)

// Citation represents a reference to another law found in text
type Citation struct {
	TargetType string // e.g., "Lei", "Decreto"
	TargetNum  string // e.g., "13.105"
}

var (
	// Regex to find "Lei nº 12.345" or "Lei 12.345"
	// Captures: 1=Type (Lei), 2=Number (12.345)
	reLei = regexp.MustCompile(`(?i)(Lei|Decreto|Decreto-Lei)\s+(?:n[º°]\s*)?([\d\.]+)`)
)

// ExtractCitations finds references to other laws in the text
func ExtractCitations(text string) []Citation {
	var citations []Citation
	matches := reLei.FindAllStringSubmatch(text, -1)

	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		tipo := normalizeType(match[1])
		numero := normalizeNumber(match[2])
		key := tipo + ":" + numero

		if seen[key] {
			continue
		}
		seen[key] = true

		citations = append(citations, Citation{
			TargetType: tipo,
			TargetNum:  numero,
		})
	}

	return citations
}

func normalizeType(t string) string {
	t = strings.ToLower(t)
	if strings.Contains(t, "decreto-lei") {
		return "Decreto-Lei"
	}
	if strings.Contains(t, "decreto") {
		return "Decreto"
	}
	return "Lei"
}

func normalizeNumber(n string) string {
	// Remove dots to standardize "13.105" to "13105" if needed, 
	// but strictly we should match what is in the DB.
	// The DB seems to use "10406" (no dots) based on file names "lei-federal-10406-2002".
	// Let's verify format in DB verifier output...
	// Verifier output showed "10406", "13105", "5452". So they have NO dots.
	return strings.ReplaceAll(n, ".", "")
}
