package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// --- Canonical ID Generation ---

// GenerateCanonicalID builds an idempotent identifier for a legal document.
// Format: tipo-esfera-numero-ano (e.g., "lei-federal-9503-1997").
func GenerateCanonicalID(tipo, esfera, numero, ano string) string {
	t := normalizeToken(tipo)
	e := normalizeToken(esfera)
	n := normalizeNumero(numero)
	a := strings.TrimSpace(ano)

	if t == "" || n == "" || a == "" {
		return ""
	}
	if e == "" {
		e = "federal" // Default for Planalto laws
	}
	return fmt.Sprintf("%s-%s-%s-%s", t, e, n, a)
}

// --- Title Parsing ---

// TitleComponents holds the parts extracted from a law title.
type TitleComponents struct {
	Tipo   string
	Numero string
	Ano    string
}

// Patterns for titles like:
//   "LEI Nº 9.503, DE 23 DE SETEMBRO DE 1997"
//   "DECRETO-LEI Nº 2.848, DE 7 DE DEZEMBRO DE 1940"
//   "LEI COMPLEMENTAR Nº 101, DE 4 DE MAIO DE 2000"
//   "MEDIDA PROVISÓRIA Nº 2.200-2, DE 24 DE AGOSTO DE 2001"
var titleRegex = regexp.MustCompile(
	`(?i)(lei\s+complementar|decreto[\s-]*lei|medida\s+provis[oó]ria|emenda\s+constitucional|lei|decreto)` +
		`\s+(?:n[ºo°]?\s*)?` +
		`([\d][\d.,\-]*)` +
		`.*?(\d{4})`,
)

// ExtractIdentityFromTitle parses a legislative title string into its components.
func ExtractIdentityFromTitle(title string) (*TitleComponents, error) {
	// Normalize unicode (handle accents, special chars)
	normalized := norm.NFC.String(title)

	matches := titleRegex.FindStringSubmatch(normalized)
	if matches == nil {
		return nil, fmt.Errorf("could not extract identity from title: %q", title)
	}

	return &TitleComponents{
		Tipo:   normalizeTipo(matches[1]),
		Numero: normalizeNumero(matches[2]),
		Ano:    matches[3],
	}, nil
}

// --- Normalization Helpers ---

// normalizeToken lowercases, strips accents, and removes special characters.
func normalizeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = removeAccents(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove anything that's not alphanumeric or hyphen
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeNumero strips dots, commas, and leading zeros from a law number.
// "9.503" -> "9503", "001" -> "1"
func normalizeNumero(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	// Keep hyphens (e.g., "2200-2" for Medida Provisória)
	s = strings.TrimLeft(s, "0")
	if s == "" {
		s = "0"
	}
	return s
}

// normalizeTipo maps various type names to a canonical short form.
func normalizeTipo(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	lower = removeAccents(lower)

	switch {
	case strings.Contains(lower, "lei complementar"):
		return "lc"
	case strings.Contains(lower, "decreto-lei"), strings.Contains(lower, "decreto lei"):
		return "decreto-lei"
	case strings.Contains(lower, "medida provisoria"):
		return "mpv"
	case strings.Contains(lower, "emenda constitucional"):
		return "ec"
	case strings.Contains(lower, "decreto"):
		return "decreto"
	case strings.Contains(lower, "lei"):
		return "lei"
	default:
		return normalizeToken(lower)
	}
}

// removeAccents strips diacritical marks from a string.
func removeAccents(s string) string {
	// Decompose to NFD (separate base char + combining marks)
	t := norm.NFD.String(s)
	var b strings.Builder
	for _, r := range t {
		// Skip combining marks (accents)
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
