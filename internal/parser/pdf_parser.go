package parser

import (
	"io"
	"regexp"

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

	urlRegex := regexp.MustCompile(`https?://[^\s)"]+`)

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	numPages := r.NumPage()
	for pageIndex := 1; pageIndex <= numPages; pageIndex++ {
		page := r.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}
		
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		matches := urlRegex.FindAllString(text, -1)
		for _, match := range matches {
			if !seen[match] {
				links = append(links, match)
				seen[match] = true
			}
		}
	}

	return links, nil
}
