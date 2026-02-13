package parser

import (
	"strings"
	"testing"
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
