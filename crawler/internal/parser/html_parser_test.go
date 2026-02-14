package parser

import (
	"strings"
	"testing"

	"github.com/user/legisgraph-crawler/internal/domain"
)

type extractLinksTestCase struct {
	name          string
	html          string
	expectedLinks map[string]string
	expectError   bool
}

var extractLinksTests = []extractLinksTestCase{
	{
		name: "Simple links",
		html: `<html><body>
			<a href="https://example.com/lei1">Lei 1</a>
			<a href="https://example.com/lei2">Lei 2</a>
		</body></html>`,
		expectedLinks: map[string]string{
			"https://example.com/lei1": "Lei 1",
			"https://example.com/lei2": "Lei 2",
		},
		expectError: false,
	},
	{
		name: "Link with title attribute",
		html: `<html><body>
			<a href="/doc/123" title="Official Document">Click here</a>
		</body></html>`,
		expectedLinks: map[string]string{
			"/doc/123": "Click here Official Document",
		},
		expectError: false,
	},
	{
		name: "Link with image alt text",
		html: `<html><body>
			<a href="/pdf/456"><img src="icon.png" alt="Download PDF"/></a>
		</body></html>`,
		expectedLinks: map[string]string{
			"/pdf/456": "Download PDF",
		},
		expectError: false,
	},
	{
		name:          "HTML without links returns empty map",
		html:          `<html><body><p>No links here</p></body></html>`,
		expectedLinks: map[string]string{},
		expectError:   false,
	},
	{
		name:          "Empty HTML returns empty map",
		html:          "",
		expectedLinks: map[string]string{},
		expectError:   false,
	},
}

func TestNewHTMLParser(t *testing.T) {
	parser := NewHTMLParser()
	if parser == nil {
		t.Fatal("NewHTMLParser() returned nil")
	}
}

func TestExtractLinks(t *testing.T) {
	parser := NewHTMLParser()

	for _, tt := range extractLinksTests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			links, err := parser.ExtractLinks(reader)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(links) != len(tt.expectedLinks) {
				t.Errorf("Link count mismatch. Expected: %d, Got: %d", len(tt.expectedLinks), len(links))
			}

			for href, expectedText := range tt.expectedLinks {
				gotText, exists := links[href]
				if !exists {
					t.Errorf("Expected link not found: %s", href)
					continue
				}
				if gotText != expectedText {
					t.Errorf("Wrong text for link %s. Expected: %q, Got: %q", href, expectedText, gotText)
				}
			}
		})
	}
}

// --- Tests for ExtractLinksWithContext ---

type contextTestCase struct {
	name     string
	html     string
	expected []domain.EnrichedLink
}

var contextTests = []contextTestCase{
	{
		name: "Link inside article caput gets article context",
		html: `<html><body>
			<p>Art. 5º - Lorem ipsum <a href="/lei/123">Lei 123</a> dolor sit amet.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/lei/123", Text: "Lei 123", Context: domain.LinkContext{Artigo: "5"}},
		},
	},
	{
		name: "Link inside paragraph gets article + paragraph context",
		html: `<html><body>
			<p>Art. 10 - Caput text here.</p>
			<p>§ 2º - Some text with <a href="/ref/abc">reference abc</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/ref/abc", Text: "reference abc", Context: domain.LinkContext{Artigo: "10", Paragrafo: "2"}},
		},
	},
	{
		name: "Link inside parágrafo único",
		html: `<html><body>
			<p>Art. 3º - Caput.</p>
			<p>Parágrafo único. See <a href="/doc/x">document X</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/doc/x", Text: "document X", Context: domain.LinkContext{Artigo: "3", Paragrafo: "unico"}},
		},
	},
	{
		name: "Link inside inciso gets article + inciso context",
		html: `<html><body>
			<p>Art. 7º - Caput text.</p>
			<p>I - First item.</p>
			<p>II - Second item with <a href="/lei/456">Lei 456</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/lei/456", Text: "Lei 456", Context: domain.LinkContext{Artigo: "7", Inciso: "II"}},
		},
	},
	{
		name: "Link inside paragraph + inciso context",
		html: `<html><body>
			<p>Art. 2º - Caput.</p>
			<p>§ 1º - Paragraph text.</p>
			<p>III - Inciso with <a href="/ref/y">ref Y</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/ref/y", Text: "ref Y", Context: domain.LinkContext{Artigo: "2", Paragrafo: "1", Inciso: "III"}},
		},
	},
	{
		name: "Link inside alínea gets full context",
		html: `<html><body>
			<p>Art. 1º - Caput.</p>
			<p>§ 3º - Paragraph.</p>
			<p>IV - Inciso text.</p>
			<p>a) Alinea with <a href="/doc/z">doc Z</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/doc/z", Text: "doc Z", Context: domain.LinkContext{Artigo: "1", Paragrafo: "3", Inciso: "IV", Alinea: "a"}},
		},
	},
	{
		name: "New article resets paragraph and inciso context",
		html: `<html><body>
			<p>Art. 1º - Caput.</p>
			<p>§ 1º - Paragraph of art 1.</p>
			<p>Art. 2º - New article with <a href="/link1">link 1</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/link1", Text: "link 1", Context: domain.LinkContext{Artigo: "2"}},
		},
	},
	{
		name: "Links before any article have empty context",
		html: `<html><body>
			<p>Some preamble with <a href="/pre/link">pre link</a>.</p>
			<p>Art. 1º - First article.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/pre/link", Text: "pre link", Context: domain.LinkContext{}},
		},
	},
	{
		name: "Multiple links in same paragraph share context",
		html: `<html><body>
			<p>Art. 5º - See <a href="/a">link A</a> and <a href="/b">link B</a>.</p>
		</body></html>`,
		expected: []domain.EnrichedLink{
			{URL: "/a", Text: "link A", Context: domain.LinkContext{Artigo: "5"}},
			{URL: "/b", Text: "link B", Context: domain.LinkContext{Artigo: "5"}},
		},
	},
	{
		name:     "HTML without links returns empty slice",
		html:     `<html><body><p>Art. 1º - No links here.</p></body></html>`,
		expected: nil,
	},
}

func TestExtractLinksWithContext(t *testing.T) {
	parser := NewHTMLParser()

	for _, tt := range contextTests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			links, err := parser.ExtractLinksWithContext(reader)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(links) != len(tt.expected) {
				t.Fatalf("Link count mismatch. Expected: %d, Got: %d.\nGot: %+v", len(tt.expected), len(links), links)
			}

			for i, expected := range tt.expected {
				got := links[i]
				if got.URL != expected.URL {
					t.Errorf("[%d] URL mismatch. Expected: %q, Got: %q", i, expected.URL, got.URL)
				}
				if got.Text != expected.Text {
					t.Errorf("[%d] Text mismatch. Expected: %q, Got: %q", i, expected.Text, got.Text)
				}
				if got.Context.Artigo != expected.Context.Artigo {
					t.Errorf("[%d] Artigo mismatch. Expected: %q, Got: %q", i, expected.Context.Artigo, got.Context.Artigo)
				}
				if got.Context.Paragrafo != expected.Context.Paragrafo {
					t.Errorf("[%d] Paragrafo mismatch. Expected: %q, Got: %q", i, expected.Context.Paragrafo, got.Context.Paragrafo)
				}
				if got.Context.Inciso != expected.Context.Inciso {
					t.Errorf("[%d] Inciso mismatch. Expected: %q, Got: %q", i, expected.Context.Inciso, got.Context.Inciso)
				}
				if got.Context.Alinea != expected.Context.Alinea {
					t.Errorf("[%d] Alinea mismatch. Expected: %q, Got: %q", i, expected.Context.Alinea, got.Context.Alinea)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || indexSubstring(s, substr) >= 0)
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
