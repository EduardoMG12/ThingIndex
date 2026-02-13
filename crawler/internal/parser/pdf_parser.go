package parser

import (
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/user/legisgraph-crawler/internal/domain"
)

type PDFParser struct{}

func NewPDFParser() domain.Parser {
	return &PDFParser{}
}

func (p *PDFParser) ExtractLinks(content io.ReaderAt, size int64) ([]string, error) {
	r, err := pdf.NewReader(content, size)
	if err != nil {
		return nil, err
	}

	var links []string
	seen := make(map[string]bool)

	urlRegex := regexp.MustCompile(`https?://[^\s)\]">]+`)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in PDF parser: %v", r)
		}
	}()

	numPages := r.NumPage()
	fullText := ""
	for pageIndex := 1; pageIndex <= numPages; pageIndex++ {
		page := r.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}
		
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		fullText += text
	}

	// Fallback/Safety: If text is empty, we can try to extract from raw content
	// but ExtractLinks already has the content as ReaderAt.
	// For now, let's just use the fullText gathered.
	
	matches := urlRegex.FindAllString(fullText, -1)
	for _, match := range matches {
		// Clean up trailing punctuation often caught by regex
		match = strings.TrimRight(match, ".,;:")
		if !seen[match] {
			links = append(links, match)
			seen[match] = true
		}
	}

	return links, nil
}
