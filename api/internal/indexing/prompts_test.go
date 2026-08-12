package indexing

import (
	"strings"
	"testing"

	"thing-index/internal/models"
)

func TestPDFPromptProfileDetectsLegalAndAttachmentFamilies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		title       string
		contains    string
		notContains string
	}{
		{
			name:        "legal root title",
			title:       "Sentença",
			contains:    "procedural posture",
			notContains: "role of the annex",
		},
		{
			name:        "certificate title",
			title:       "Certidão de Publicação",
			contains:    "procedural posture",
			notContains: "role of the annex",
		},
		{
			name:        "attachment title",
			title:       "Anexo I - Planilha de Cálculo",
			contains:    "role of the annex",
			notContains: "operational language over marketing language",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			profile := pdfPromptProfile(&models.Document{Title: tc.title, MimeType: "application/pdf"})
			if !strings.Contains(profile, tc.contains) {
				t.Fatalf("expected profile for %q to contain %q, got %q", tc.title, tc.contains, profile)
			}
			if tc.notContains != "" && strings.Contains(profile, tc.notContains) {
				t.Fatalf("expected profile for %q to avoid %q, got %q", tc.title, tc.notContains, profile)
			}
		})
	}
}
