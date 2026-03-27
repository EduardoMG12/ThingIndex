package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestGetDocumentContentMarksMissingSourceAsOrphaned(t *testing.T) {
	t.Parallel()

	doc := &models.Document{ID: 12, UserID: 5, Title: "Contrato.pdf", FileURL: "docs/contrato.pdf"}
	repo := &documentRepoStub{document: doc}
	useCase := &documentCatalogUseCase{
		documentRepo: repo,
		storage: &storageStub{
			openErrByPath: map[string]error{
				"docs/contrato.pdf": errors.New("object not found"),
			},
			existsByPath: map[string]bool{
				"docs/contrato.pdf": false,
			},
		},
		storageResolver: &resolverStub{err: domain.ErrStorageProviderNotConfigured},
	}

	_, err := useCase.GetDocumentContent(context.Background(), 5, 12)
	if !errors.Is(err, domain.ErrDocumentSourceMissing) {
		t.Fatalf("expected missing source error, got %v", err)
	}
	if len(repo.markOrphanedDocumentIDs) != 1 || repo.markOrphanedDocumentIDs[0] != 12 {
		t.Fatalf("expected document to be marked orphaned, got %#v", repo.markOrphanedDocumentIDs)
	}
}

func TestGetDocumentContentFallsBackToDefaultStorageWhenCustomStorageFails(t *testing.T) {
	t.Parallel()

	doc := &models.Document{ID: 15, UserID: 7, Title: "Relatorio.pdf", FileURL: "docs/relatorio.pdf"}
	repo := &documentRepoStub{document: doc}
	defaultStorage := &storageStub{
		openBodyByPath: map[string]string{
			"docs/relatorio.pdf": "fallback-pdf-content",
		},
	}
	customStorage := &storageStub{
		openErrByPath: map[string]error{
			"docs/relatorio.pdf": errors.New("custom storage unavailable"),
		},
		existsByPath: map[string]bool{
			"docs/relatorio.pdf": false,
		},
	}
	useCase := &documentCatalogUseCase{
		documentRepo:    repo,
		storage:         defaultStorage,
		storageResolver: &resolverStub{storage: customStorage},
	}

	content, err := useCase.GetDocumentContent(context.Background(), 7, 15)
	if err != nil {
		t.Fatalf("expected fallback content to succeed, got %v", err)
	}
	defer content.Reader.Close()

	body, readErr := io.ReadAll(content.Reader)
	if readErr != nil {
		t.Fatalf("expected to read fallback content, got %v", readErr)
	}
	if string(body) != "fallback-pdf-content" {
		t.Fatalf("unexpected fallback content: %q", string(body))
	}
	if len(repo.markOrphanedDocumentIDs) != 0 {
		t.Fatalf("expected no orphan marking when fallback succeeds, got %#v", repo.markOrphanedDocumentIDs)
	}
}

func TestUploadDocumentUsesResolvedCustomStorageWhenAvailable(t *testing.T) {
	t.Parallel()

	collectionID := uint(14)
	defaultStorage := &storageStub{uploadURL: "docs/default.pdf"}
	customStorage := &storageStub{uploadURL: "docs/custom.pdf"}
	repo := &documentRepoStub{}
	useCase := &documentCatalogUseCase{
		categoryRepo:       &categoryRepoStub{},
		collectionRepo:     &collectionRepoStub{collectionByName: &models.Collection{ID: collectionID, UserID: 8, Name: "Lote", NormalizedName: "lote"}},
		documentRepo:       repo,
		storage:            defaultStorage,
		storageResolver:    &resolverStub{storage: customStorage},
		maxUploadSizeBytes: 10 << 20,
	}

	doc, err := useCase.UploadDocument(context.Background(), 8, "Relatorio.pdf", io.NopCloser(strings.NewReader("%PDF-1.7 sample")), 15, "application/pdf", "", "", "Lote")
	if err != nil {
		t.Fatalf("expected upload with custom storage to succeed, got %v", err)
	}
	if doc.FileURL != "docs/custom.pdf" {
		t.Fatalf("expected file url from custom storage, got %#v", doc)
	}
	if customStorage.uploadCalls != 1 {
		t.Fatalf("expected custom storage upload to be used once, got %d", customStorage.uploadCalls)
	}
	if defaultStorage.uploadCalls != 0 {
		t.Fatalf("expected default storage upload not to run, got %d", defaultStorage.uploadCalls)
	}
}
