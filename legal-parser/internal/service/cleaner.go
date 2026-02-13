package service

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

// CleanHTML takes raw HTML bytes (potentially Windows-1252 encoded),
// converts to UTF-8, removes styling tags, and returns clean text.
// It also returns the goquery document for further structural parsing.
func CleanHTML(raw []byte) (*goquery.Document, error) {
	// Auto-detect encoding and convert to UTF-8
	reader, err := charset.NewReader(bytes.NewReader(raw), "text/html")
	if err != nil {
		// Fallback: try raw bytes directly
		reader = bytes.NewReader(raw)
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	// Remove script and style elements entirely
	doc.Find("script, style, noscript").Remove()

	return doc, nil
}

// ExtractCleanText extracts human-readable text from a goquery selection,
// stripping all HTML formatting tags but preserving structure.
func ExtractCleanText(sel *goquery.Selection) string {
	if sel == nil {
		return ""
	}

	text := sel.Text()
	text = cleanWhitespace(text)
	return text
}

// IsRevoked checks if a goquery selection contains revocation markers:
// <strike>, <s>, or style="...line-through..."
func IsRevoked(sel *goquery.Selection) bool {
	// Check for <strike> or <s> tags
	if sel.Find("strike, s").Length() > 0 {
		return true
	}

	// Check inline style for line-through
	style, exists := sel.Attr("style")
	if exists && strings.Contains(strings.ToLower(style), "line-through") {
		return true
	}

	// Check children for line-through style
	found := false
	sel.Find("[style]").Each(func(_ int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		if strings.Contains(strings.ToLower(style), "line-through") {
			found = true
		}
	})

	return found
}

// ExtractTitle tries to find the law title from the document.
// It looks for common Planalto patterns.
func ExtractTitle(doc *goquery.Document) string {
	// Pattern 1: Link with legislacao.planalto.gov.br/legisla
	var title string
	doc.Find("a[href*='legislacao.planalto.gov.br']").Each(func(_ int, s *goquery.Selection) {
		if title == "" {
			title = cleanWhitespace(s.Text())
		}
	})
	if title != "" {
		return title
	}

	// Pattern 2: <title> tag
	title = cleanWhitespace(doc.Find("title").Text())
	if title != "" {
		return title
	}

	return ""
}

// ExtractEmenta tries to find the "ementa" (summary) of the law.
// On Planalto, it's often in a table cell with color="#800000".
func ExtractEmenta(doc *goquery.Document) string {
	var ementa string
	doc.Find(`font[color="#800000"], font[color="#800080"]`).Each(func(_ int, s *goquery.Selection) {
		text := cleanWhitespace(s.Text())
		if len(text) > 20 && ementa == "" {
			ementa = text
		}
	})
	return ementa
}

// cleanWhitespace normalizes whitespace: collapses multiple spaces/newlines,
// replaces &nbsp; sequences, and trims.
var spaceRegex = regexp.MustCompile(`[\s\x{00A0}]+`)

func cleanWhitespace(s string) string {
	s = spaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
