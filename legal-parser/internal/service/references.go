package service

import (
	"regexp"
	"strings"
)

// --- Reference Extraction ---

// referencePatterns defines regex patterns to find citations to other laws in text.
// Each pattern captures: (tipo) ... (numero) ... (ano)
var referencePatterns = []*regexp.Regexp{
	// "Lei nº 8.112, de 1990" / "Lei 8112/90"
	regexp.MustCompile(`(?i)(lei\s+complementar|decreto[\s-]*lei|medida\s+provis[oó]ria|emenda\s+constitucional|lei|decreto)\s+(?:n[ºo°]?\s*)?(\d[\d.,\-]*)\s*(?:[,/]\s*de\s+)?(?:\d{1,2}\s+de\s+\w+\s+de\s+)?(\d{4})`),
	// "Lei nº 8.112/1990"
	regexp.MustCompile(`(?i)(lei\s+complementar|decreto[\s-]*lei|medida\s+provis[oó]ria|emenda\s+constitucional|lei|decreto)\s+(?:n[ºo°]?\s*)?([\d.,\-]+)/(\d{2,4})`),
}

// ExtractReferences scans text for citations to other Brazilian laws
// and returns their canonical IDs.
func ExtractReferences(text string) []string {
	seen := map[string]bool{}
	var refs []string

	for _, pat := range referencePatterns {
		matches := pat.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			if len(m) < 4 {
				continue
			}

			tipo := normalizeTipo(m[1])
			numero := normalizeNumero(m[2])
			ano := normalizeAno(m[3])

			id := GenerateCanonicalID(tipo, "federal", numero, ano)
			if id == "" {
				continue
			}

			if !seen[id] {
				seen[id] = true
				refs = append(refs, id)
			}
		}
	}

	return refs
}

// normalizeAno handles 2-digit years: "90" -> "1990", "03" -> "2003"
func normalizeAno(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 2 {
		y := s
		if y >= "30" {
			return "19" + y
		}
		return "20" + y
	}
	return s
}
