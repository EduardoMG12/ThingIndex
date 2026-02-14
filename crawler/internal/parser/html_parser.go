package parser

import (
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/legisgraph-crawler/internal/domain"
)

// --- Regex patterns for hierarchical element detection ---
var (
	// Art. 1º, Art. 2o, Art. 10.
	ctxArtRegex = regexp.MustCompile(`(?i)^\s*Art\.\s*(\d+)[ºo°.]?\s*[-–]?\s*`)

	// § 1º, § 2o, Parágrafo único.
	ctxParagraphRegex      = regexp.MustCompile(`(?i)^\s*(?:§\s*(\d+)[ºo°.]?|Par[áa]grafo\s+[úu]nico\.?)\s*[-–]?\s*`)
	ctxParagraphUnicoRegex = regexp.MustCompile(`(?i)^\s*Par[áa]grafo\s+[úu]nico`)

	// I -, II -, III -, IV - (Roman numerals for incisos)
	ctxIncisoRegex = regexp.MustCompile(`(?i)^\s*(X{0,3}(?:IX|IV|V?I{0,3}))\s*[-–]\s*`)

	// a), b), c) (alíneas)
	ctxAlineaRegex = regexp.MustCompile(`(?i)^\s*([a-z])\)\s*`)
)

type HTMLParser struct{}

func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// ExtractLinks extracts all hrefs from a tags, and optionally allows filtering by text content if we wanted to inside here,
// but for now we just extract all and let the crawler filter.
// We return a map of URL -> Link Text to help with filtering.
func (h *HTMLParser) ExtractLinks(content io.Reader) (map[string]string, error) {
	doc, err := goquery.NewDocumentFromReader(content)
	if err != nil {
		return nil, err
	}

	links := make(map[string]string)
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			text := strings.TrimSpace(s.Text())

			// Capture title attribute
			if title, ok := s.Attr("title"); ok && title != "" {
				text = text + " " + strings.TrimSpace(title)
			}

			// Capture alt text of nested images
			s.Find("img").Each(func(j int, img *goquery.Selection) {
				if alt, ok := img.Attr("alt"); ok && alt != "" {
					text = text + " " + strings.TrimSpace(alt)
				}
			})

			links[href] = strings.TrimSpace(text)
		}
	})

	return links, nil
}

// ExtractLinksWithContext walks through all <p> tags sequentially, tracking
// the current hierarchical position (Art → § → Inciso → Alínea) using the
// same regex patterns as the legal-parser. For every <a> tag found within
// a <p>, it captures the link URL and the current structural context.
func (h *HTMLParser) ExtractLinksWithContext(content io.Reader) ([]domain.EnrichedLink, error) {
	doc, err := goquery.NewDocumentFromReader(content)
	if err != nil {
		return nil, err
	}

	var enrichedLinks []domain.EnrichedLink

	// Current hierarchical position tracker
	var currentArtigo string
	var currentParagrafo string
	var currentInciso string
	var currentAlinea string

	doc.Find("p").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text == "" {
			return
		}

		// Collapse whitespace for regex matching
		normalized := collapseWhitespace(text)

		// --- Detect hierarchical level changes ---

		// Match Article: resets everything below
		if m := ctxArtRegex.FindStringSubmatch(normalized); m != nil {
			currentArtigo = m[1]
			currentParagrafo = ""
			currentInciso = ""
			currentAlinea = ""
		} else if m := ctxParagraphRegex.FindStringSubmatch(normalized); m != nil {
			// Match Paragraph: resets inciso and alínea
			if ctxParagraphUnicoRegex.MatchString(normalized) {
				currentParagrafo = "unico"
			} else {
				currentParagrafo = m[1]
			}
			currentInciso = ""
			currentAlinea = ""
		} else if m := ctxIncisoRegex.FindStringSubmatch(normalized); m != nil {
			// Match Inciso: resets alínea
			currentInciso = m[1]
			currentAlinea = ""
		} else if m := ctxAlineaRegex.FindStringSubmatch(normalized); m != nil {
			// Match Alínea
			currentAlinea = m[1]
		}

		// --- Extract all <a> tags in this <p> ---
		sel.Find("a").Each(func(_ int, a *goquery.Selection) {
			href, exists := a.Attr("href")
			if !exists || href == "" {
				return
			}

			linkText := strings.TrimSpace(a.Text())

			// Capture title attribute
			if title, ok := a.Attr("title"); ok && title != "" {
				linkText = linkText + " " + strings.TrimSpace(title)
			}

			// Capture alt text of nested images
			a.Find("img").Each(func(_ int, img *goquery.Selection) {
				if alt, ok := img.Attr("alt"); ok && alt != "" {
					linkText = linkText + " " + strings.TrimSpace(alt)
				}
			})

			enrichedLinks = append(enrichedLinks, domain.EnrichedLink{
				URL:  href,
				Text: strings.TrimSpace(linkText),
				Context: domain.LinkContext{
					Artigo:    currentArtigo,
					Paragrafo: currentParagrafo,
					Inciso:    currentInciso,
					Alinea:    currentAlinea,
				},
			})
		})
	})

	return enrichedLinks, nil
}

// collapseWhitespace replaces all whitespace sequences (including nbsp) with a single space.
func collapseWhitespace(s string) string {
	// Replace non-breaking spaces and other unicode spaces
	s = strings.Map(func(r rune) rune {
		if r == '\u00A0' || r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, s)

	// Collapse multiple spaces
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}
