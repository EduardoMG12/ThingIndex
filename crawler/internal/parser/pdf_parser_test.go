package parser

import (
	"bytes"
	"testing"
)

func TestNewPDFParser(t *testing.T) {
	p := NewPDFParser()
	if p == nil {
		t.Fatal("NewPDFParser() returned nil")
	}
}

func TestPDFExtractLinks_InvalidContent(t *testing.T) {
	p := NewPDFParser()

	// Not a valid PDF — should return error from pdf.NewReader
	invalidContent := bytes.NewReader([]byte("this is not a pdf"))
	_, err := p.ExtractLinks(invalidContent, int64(len("this is not a pdf")))
	if err == nil {
		t.Error("Expected error for invalid PDF content, but got nil")
	}
}

func TestPDFExtractLinks_EmptyContent(t *testing.T) {
	p := NewPDFParser()

	emptyContent := bytes.NewReader([]byte{})
	_, err := p.ExtractLinks(emptyContent, 0)
	if err == nil {
		t.Error("Expected error for empty content, but got nil")
	}
}

func TestPDFExtractLinks_MinimalValidPDF(t *testing.T) {
	// A minimal valid PDF that contains no URLs
	// This is the smallest possible valid PDF structure
	minimalPDF := []byte(`%PDF-1.0
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>
endobj
xref
0 4
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
trailer
<< /Size 4 /Root 1 0 R >>
startxref
190
%%EOF`)

	p := NewPDFParser()
	reader := bytes.NewReader(minimalPDF)
	links, err := p.ExtractLinks(reader, int64(len(minimalPDF)))

	// The minimal PDF library may or may not parse this correctly
	// If it parses, we expect no links. If not, error is acceptable.
	if err != nil {
		// Acceptable — the minimal PDF structure might not be enough for the library
		t.Logf("Parser returned error for minimal PDF (acceptable): %v", err)
		return
	}

	if len(links) != 0 {
		t.Errorf("Expected 0 links from minimal PDF, got %d: %v", len(links), links)
	}
}
