package crawler

import (
	"testing"
)

func TestResolveRelURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		href     string
		expected string
	}{
		{
			name:     "Absolute URL stays the same",
			baseURL:  "https://example.com/page",
			href:     "https://other.com/doc.pdf",
			expected: "https://other.com/doc.pdf",
		},
		{
			name:     "Relative URL resolved against base",
			baseURL:  "https://example.com/dir/page.html",
			href:     "../file.pdf",
			expected: "https://example.com/file.pdf",
		},
		{
			name:     "Root-relative URL resolved against base",
			baseURL:  "https://example.com/dir/page.html",
			href:     "/documents/law.pdf",
			expected: "https://example.com/documents/law.pdf",
		},
		{
			name:     "Invalid href returns empty",
			baseURL:  "https://example.com",
			href:     "://bad",
			expected: "",
		},
		{
			name:     "Invalid base returns empty",
			baseURL:  "://bad",
			href:     "/test",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveRelURL(tt.baseURL, tt.href)
			if result != tt.expected {
				t.Errorf("resolveRelURL(%q, %q) = %q, want %q", tt.baseURL, tt.href, result, tt.expected)
			}
		})
	}
}

func TestShouldVisit(t *testing.T) {
	tests := []struct {
		name           string
		ignoreKeywords []string
		url            string
		linkText       string
		expected       bool
	}{
		{
			name:           "PDF link should be visited",
			ignoreKeywords: nil,
			url:            "https://example.com/doc.pdf",
			linkText:       "Download",
			expected:       true,
		},
		{
			name:           "PDF with ext param should be visited",
			ignoreKeywords: nil,
			url:            "https://example.com/doc?ext=.pdf",
			linkText:       "",
			expected:       true,
		},
		{
			name:           "HTML link should be visited",
			ignoreKeywords: nil,
			url:            "https://example.com/page.html",
			linkText:       "Some page",
			expected:       true,
		},
		{
			name:           "Link without extension should be visited",
			ignoreKeywords: nil,
			url:            "https://example.com/page",
			linkText:       "Some page",
			expected:       true,
		},
		{
			name:           "Image file should NOT be visited",
			ignoreKeywords: nil,
			url:            "https://example.com/photo.jpg",
			linkText:       "Photo",
			expected:       false,
		},
		{
			name:           "Link matching ignore keyword should NOT be visited",
			ignoreKeywords: []string{"login", "signup"},
			url:            "https://example.com/page.html",
			linkText:       "Login here",
			expected:       false,
		},
		{
			name:           "Link NOT matching ignore keyword should be visited",
			ignoreKeywords: []string{"login", "signup"},
			url:            "https://example.com/page.html",
			linkText:       "Important document",
			expected:       true,
		},
		{
			name:           "Planalto HTML link should always be visited",
			ignoreKeywords: []string{"login"},
			url:            "https://www.planalto.gov.br/ccivil_03/leis/page.htm",
			linkText:       "login",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &CrawlerService{
				ignoreKeywords: tt.ignoreKeywords,
			}
			result := svc.shouldVisit(tt.url, tt.linkText)
			if result != tt.expected {
				t.Errorf("shouldVisit(%q, %q) = %v, want %v", tt.url, tt.linkText, result, tt.expected)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		ext          string
		dirName      string
		outputSubdir string
		wantContains string
	}{
		{
			name:         "PDF filename contains extension",
			url:          "https://example.com/documents/law-123.pdf",
			ext:          ".pdf",
			dirName:      "",
			outputSubdir: "",
			wantContains: ".pdf",
		},
		{
			name:         "HTML filename contains extension",
			url:          "https://example.com/page",
			ext:          ".htm",
			dirName:      "",
			outputSubdir: "",
			wantContains: ".htm",
		},
		{
			name:         "Output subdir is prepended",
			url:          "https://example.com/doc.pdf",
			ext:          ".pdf",
			dirName:      "",
			outputSubdir: "laws",
			wantContains: "laws/",
		},
		{
			name:         "Same URL always generates same filename",
			url:          "https://example.com/stable.pdf",
			ext:          ".pdf",
			dirName:      "",
			outputSubdir: "",
			wantContains: ".pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &CrawlerService{
				outputSubdir: tt.outputSubdir,
			}
			result := svc.generateFilename(tt.url, tt.ext, tt.dirName)
			if result == "" {
				t.Error("generateFilename returned empty string")
			}
			if !contains(result, tt.wantContains) {
				t.Errorf("generateFilename(%q, %q) = %q, expected to contain %q", tt.url, tt.ext, result, tt.wantContains)
			}
		})
	}

	// Determinism test: same input = same output
	t.Run("Deterministic output", func(t *testing.T) {
		svc := &CrawlerService{}
		a := svc.generateFilename("https://example.com/test.pdf", ".pdf", "")
		b := svc.generateFilename("https://example.com/test.pdf", ".pdf", "")
		if a != b {
			t.Errorf("generateFilename is not deterministic: %q != %q", a, b)
		}
	})

	// Different URLs = different filenames
	t.Run("Different URLs produce different filenames", func(t *testing.T) {
		svc := &CrawlerService{}
		a := svc.generateFilename("https://example.com/doc1.pdf", ".pdf", "")
		b := svc.generateFilename("https://example.com/doc2.pdf", ".pdf", "")
		if a == b {
			t.Errorf("Different URLs produced same filename: %q", a)
		}
	})
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
