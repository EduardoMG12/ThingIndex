package indexing

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func (i *DeterministicIndexer) indexHTML(documentID uint, data []byte) (domain.IndexingResult, error) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse html: %v", domain.ErrDocumentParsingFailed, err)
	}

	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	tables := make([]models.TableSection, 0)
	pageNumber := 1
	currentSectionKey := ""
	currentSectionTitle := ""

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			switch node.Data {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level, _ := strconv.Atoi(strings.TrimPrefix(node.Data, "h"))
				title := normalizeText(extractHTMLText(node))
				if title != "" {
					currentSectionTitle = title
					currentSectionKey = slugify(title)
					headings = append(headings, outlineDraft{
						Level:          level,
						Title:          title,
						ParagraphStart: len(paragraphs),
						SectionKey:     currentSectionKey,
					})
				}
			case "p", "li":
				text := normalizeText(extractHTMLText(node))
				if text != "" {
					paragraphs = append(paragraphs, textBlock{
						Content:    text,
						PageNumber: &pageNumber,
						SectionKey: currentSectionKey,
					})
				}
			case "table":
				section := extractHTMLTable(documentID, len(tables), node)
				if len(section.Headers) > 0 || len(section.Rows) > 0 {
					if currentSectionTitle != "" {
						section.Title = currentSectionTitle
						section.Metadata["section_key"] = currentSectionKey
					}
					tables = append(tables, section)
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)

	return buildIndexingResult(documentID, headings, paragraphs, tables), nil
}

func extractHTMLText(node *html.Node) string {
	var parts []string
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current == nil {
			return
		}
		if current.Type == html.TextNode {
			text := strings.TrimSpace(current.Data)
			if text != "" {
				parts = append(parts, text)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return strings.Join(parts, " ")
}

func extractHTMLTable(documentID uint, sequence int, node *html.Node) models.TableSection {
	rows := make([][]string, 0)
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current == nil {
			return
		}
		if current.Type == html.ElementNode && current.Data == "tr" {
			cells := make([]string, 0)
			for child := current.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
					text := normalizeText(extractHTMLText(child))
					cells = append(cells, text)
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)

	section := models.TableSection{
		DocumentID:    documentID,
		SequenceOrder: sequence,
		Headers:       []string{},
		Rows:          [][]string{},
		Metadata:      map[string]any{},
	}
	if len(rows) > 0 {
		section.Headers = rows[0]
	}
	if len(rows) > 1 {
		section.Rows = rows[1:]
	}
	return section
}
