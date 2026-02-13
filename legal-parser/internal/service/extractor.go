package service

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/legisgraph/legal-parser/internal/domain"
)

// --- Regex patterns for hierarchical elements ---

var (
	// Art. 1º, Art. 2o, Art. 10.
	artRegex = regexp.MustCompile(`(?i)^\s*Art\.\s*(\d+)[ºo°.]?\s*[-–]?\s*`)

	// § 1º, § 2o, Parágrafo único.
	paragraphRegex = regexp.MustCompile(`(?i)^\s*(?:§\s*(\d+)[ºo°.]?|Par[áa]grafo\s+[úu]nico\.?)\s*[-–]?\s*`)
	paragraphUnicoRegex = regexp.MustCompile(`(?i)^\s*Par[áa]grafo\s+[úu]nico`)

	// I -, II -, III -, IV - (Roman numerals for incisos)
	incisoRegex = regexp.MustCompile(`(?i)^\s*(X{0,3}(?:IX|IV|V?I{0,3}))\s*[-–]\s*`)

	// a), b), c) (alíneas)
	alineaRegex = regexp.MustCompile(`(?i)^\s*([a-z])\)\s*`)
)

// ExtractStructure parses the HTML document into a hierarchical tree of
// Articles → Paragraphs → Incisos → Alíneas.
func ExtractStructure(doc *goquery.Document) []domain.Article {
	var articles []domain.Article
	var currentArticle *domain.Article
	var currentParagraph *domain.Paragraph
	var currentInciso *domain.Subsection

	// Track whether we're inside an article's incisos vs paragraph's incisos
	inArticleIncisos := false

	// Find all paragraph-like elements (Planalto uses <p> tags)
	doc.Find("p").Each(func(_ int, sel *goquery.Selection) {
		text := ExtractCleanText(sel)
		if text == "" {
			return
		}

		revoked := IsRevoked(sel)
		status := "vigente"
		if revoked {
			status = "revogado"
		}

		// Check for VETADO
		if strings.Contains(strings.ToUpper(text), "(VETADO)") || strings.Contains(strings.ToUpper(text), "VETADO") {
			status = "vetado"
		}

		// --- Match Article ---
		if m := artRegex.FindStringSubmatch(text); m != nil {
			// Save previous article
			flushInciso(currentInciso, currentParagraph, currentArticle, inArticleIncisos)
			flushParagraph(currentParagraph, currentArticle)
			if currentArticle != nil {
				articles = append(articles, *currentArticle)
			}

			caput := artRegex.ReplaceAllString(text, "")
			currentArticle = &domain.Article{
				Numero: m[1],
				Caput:  strings.TrimSpace(caput),
				Status: status,
			}
			currentParagraph = nil
			currentInciso = nil
			inArticleIncisos = false
			return
		}

		if currentArticle == nil {
			return // Skip content before any article
		}

		// --- Match Paragraph ---
		if m := paragraphRegex.FindStringSubmatch(text); m != nil {
			flushInciso(currentInciso, currentParagraph, currentArticle, inArticleIncisos)
			flushParagraph(currentParagraph, currentArticle)

			numero := m[1]
			if paragraphUnicoRegex.MatchString(text) {
				numero = "unico"
			}

			paraText := paragraphRegex.ReplaceAllString(text, "")
			currentParagraph = &domain.Paragraph{
				Numero: numero,
				Texto:  strings.TrimSpace(paraText),
				Status: status,
			}
			currentInciso = nil
			inArticleIncisos = false
			return
		}

		// --- Match Inciso ---
		if m := incisoRegex.FindStringSubmatch(text); m != nil {
			flushInciso(currentInciso, currentParagraph, currentArticle, inArticleIncisos)

			incisoText := incisoRegex.ReplaceAllString(text, "")
			currentInciso = &domain.Subsection{
				Numero: m[1],
				Texto:  strings.TrimSpace(incisoText),
				Status: status,
			}

			// If we have a current paragraph, incisos go under it
			inArticleIncisos = (currentParagraph == nil)
			return
		}

		// --- Match Alínea ---
		if m := alineaRegex.FindStringSubmatch(text); m != nil && currentInciso != nil {
			alineaText := alineaRegex.ReplaceAllString(text, "")
			currentInciso.Alineas = append(currentInciso.Alineas, domain.Item{
				Numero: m[1],
				Texto:  strings.TrimSpace(alineaText),
				Status: status,
			})
			return
		}

		// --- Continuation text (append to current context) ---
		if currentInciso != nil {
			currentInciso.Texto += " " + text
		} else if currentParagraph != nil {
			currentParagraph.Texto += " " + text
		} else if currentArticle != nil {
			currentArticle.Caput += " " + text
		}
	})

	// Flush remaining state
	flushInciso(currentInciso, currentParagraph, currentArticle, inArticleIncisos)
	flushParagraph(currentParagraph, currentArticle)
	if currentArticle != nil {
		articles = append(articles, *currentArticle)
	}

	return articles
}

func flushInciso(inciso *domain.Subsection, para *domain.Paragraph, art *domain.Article, inArticle bool) {
	if inciso == nil {
		return
	}
	if !inArticle && para != nil {
		para.Incisos = append(para.Incisos, *inciso)
	} else if art != nil {
		art.Incisos = append(art.Incisos, *inciso)
	}
}

func flushParagraph(para *domain.Paragraph, art *domain.Article) {
	if para == nil || art == nil {
		return
	}
	art.Paragrafos = append(art.Paragrafos, *para)
}
