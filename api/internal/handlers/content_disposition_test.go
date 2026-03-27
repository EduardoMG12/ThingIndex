package handlers

import (
	"strings"
	"testing"
)

func TestSanitizeContentDispositionFilenameRemovesControlCharsAndPathSegments(t *testing.T) {
	t.Parallel()

	got := sanitizeContentDispositionFilename("..\\relatorios/\r\ncontrato final.pdf")
	want := "contrato final.pdf"

	if got != want {
		t.Fatalf("expected sanitized filename %q, got %q", want, got)
	}
}

func TestSanitizeContentDispositionFilenameFallsBackWhenEmpty(t *testing.T) {
	t.Parallel()

	if got := sanitizeContentDispositionFilename(" \r\n "); got != fallbackDownloadFilename {
		t.Fatalf("expected fallback filename %q, got %q", fallbackDownloadFilename, got)
	}
}

func TestBuildSafeContentDispositionKeepsInlineAndEscapesFilename(t *testing.T) {
	t.Parallel()

	got := buildSafeContentDisposition("inline", "Contrato final.pdf")
	if !strings.HasPrefix(got, "inline;") {
		t.Fatalf("expected inline disposition, got %q", got)
	}
	if !strings.Contains(got, `filename="Contrato final.pdf"`) {
		t.Fatalf("expected sanitized filename in disposition, got %q", got)
	}
}

func TestBuildSafeContentDispositionNormalizesUnexpectedDisposition(t *testing.T) {
	t.Parallel()

	got := buildSafeContentDisposition("form-data", "../../segredo.txt")
	if !strings.HasPrefix(got, "attachment;") {
		t.Fatalf("expected attachment disposition, got %q", got)
	}
	if strings.Contains(got, "..") {
		t.Fatalf("expected path traversal markers to be removed, got %q", got)
	}
}
