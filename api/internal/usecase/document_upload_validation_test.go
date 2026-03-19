package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

func TestValidateUploadStreamAcceptsPDFBySignature(t *testing.T) {
	t.Parallel()

	reader, contentType, err := validateUploadStream(strings.NewReader("%PDF-1.7 sample"), "application/pdf", "contrato.pdf")
	if err != nil {
		t.Fatalf("expected pdf upload to be accepted, got %v", err)
	}
	if contentType != "application/pdf" {
		t.Fatalf("expected application/pdf, got %q", contentType)
	}

	body, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("expected to read validated stream, got %v", readErr)
	}
	if string(body) != "%PDF-1.7 sample" {
		t.Fatalf("expected validated stream to preserve content, got %q", string(body))
	}
}

func TestValidateUploadStreamAcceptsHTMLWhenBrowserSniffsTextPlain(t *testing.T) {
	t.Parallel()

	reader, contentType, err := validateUploadStream(strings.NewReader("titulo simples"), "text/html", "pagina.html")
	if err != nil {
		t.Fatalf("expected html upload to be accepted, got %v", err)
	}
	if contentType != "text/html" {
		t.Fatalf("expected text/html, got %q", contentType)
	}

	body, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("expected to read validated stream, got %v", readErr)
	}
	if string(body) != "titulo simples" {
		t.Fatalf("expected validated stream to preserve content, got %q", string(body))
	}
}

func TestValidateUploadStreamAcceptsXLSXByZipSignature(t *testing.T) {
	t.Parallel()

	reader, contentType, err := validateUploadStream(bytes.NewReader([]byte("PK\x03\x04xlsx-sample")), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "planilha.xlsx")
	if err != nil {
		t.Fatalf("expected xlsx upload to be accepted, got %v", err)
	}
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("expected xlsx content type, got %q", contentType)
	}

	body, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("expected to read validated stream, got %v", readErr)
	}
	if string(body) != "PK\x03\x04xlsx-sample" {
		t.Fatalf("expected validated stream to preserve xlsx content, got %q", string(body))
	}
}

func TestValidateUploadStreamRejectsUnsupportedMime(t *testing.T) {
	t.Parallel()

	_, _, err := validateUploadStream(bytes.NewReader([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}), "image/png", "imagem.png")
	if !errors.Is(err, domain.ErrDocumentInvalidMime) {
		t.Fatalf("expected invalid mime error, got %v", err)
	}
}

func TestValidateUploadStreamRejectsEmptyFile(t *testing.T) {
	t.Parallel()

	_, _, err := validateUploadStream(strings.NewReader(""), "application/pdf", "vazio.pdf")
	if !errors.Is(err, domain.ErrFileRequired) {
		t.Fatalf("expected file required error, got %v", err)
	}
}

func TestUploadDocumentRejectsOversizedFileBeforeStorage(t *testing.T) {
	t.Parallel()

	useCase := &documentCatalogUseCase{
		maxUploadSizeBytes: 4,
	}

	_, err := useCase.UploadDocument(context.Background(), 1, "Contrato.pdf", strings.NewReader("12345"), 5, "application/pdf", "", "", "Colecao")
	if !errors.Is(err, domain.ErrFileTooLarge) {
		t.Fatalf("expected file too large error, got %v", err)
	}
}
