package parser

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
			links[href] = text
		}
	})

	return links, nil
}
