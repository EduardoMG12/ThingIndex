package usecase

import (
	"context"
	"errors"
	"io"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type documentRepoStub struct {
	document                *models.Document
	documentsByCollection   []models.Document
	deleteDocumentCalled    bool
	deleteDocumentAudit     *models.DocumentAuditEvent
	deleteCollectionCalled  bool
	deleteCollectionAudits  []models.DocumentAuditEvent
	recordedAuditEvents     []models.DocumentAuditEvent
	deleteDocumentErr       error
	deleteCollectionErr     error
	markOrphanedDocumentIDs []uint
}

func (r *documentRepoStub) CreateDocument(context.Context, *models.Document) error {
	return nil
}

func (r *documentRepoStub) CreateDocumentWithOutbox(context.Context, *models.Document, *models.OutboxEvent) error {
	return nil
}

func (r *documentRepoStub) CreateDocumentAuditEvents(_ context.Context, events []models.DocumentAuditEvent) error {
	r.recordedAuditEvents = append(r.recordedAuditEvents, events...)
	return nil
}

func (r *documentRepoStub) GetDocumentByID(context.Context, uint, uint) (*models.Document, error) {
	if r.document == nil {
		return nil, domain.ErrDocumentNotFound
	}
	return r.document, nil
}

func (r *documentRepoStub) GetDocumentByIDForProcessing(context.Context, uint) (*models.Document, error) {
	return nil, nil
}

func (r *documentRepoStub) GetDocumentsByUserID(context.Context, uint) ([]models.Document, error) {
	return nil, nil
}

func (r *documentRepoStub) GetDocumentsByCategoryID(context.Context, uint, uint) ([]models.Document, error) {
	return nil, nil
}

func (r *documentRepoStub) GetDocumentsByCollectionID(context.Context, uint, uint) ([]models.Document, error) {
	return r.documentsByCollection, nil
}

func (r *documentRepoStub) UpdateDocumentStatus(context.Context, uint, string) error {
	return nil
}

func (r *documentRepoStub) RetryDocumentIndexing(context.Context, uint, uint) (*models.Document, error) {
	return nil, nil
}

func (r *documentRepoStub) MarkDocumentOrphaned(_ context.Context, _ uint, documentID uint, _ string) (*models.Document, error) {
	r.markOrphanedDocumentIDs = append(r.markOrphanedDocumentIDs, documentID)
	return r.document, nil
}

func (r *documentRepoStub) DeleteDocument(_ context.Context, _ uint, _ uint, auditEvent *models.DocumentAuditEvent) error {
	r.deleteDocumentCalled = true
	if auditEvent != nil {
		copied := *auditEvent
		r.deleteDocumentAudit = &copied
	}
	return r.deleteDocumentErr
}

func (r *documentRepoStub) DeleteDocumentsByCollection(_ context.Context, _ uint, _ uint, auditEvents []models.DocumentAuditEvent) (int64, error) {
	r.deleteCollectionCalled = true
	r.deleteCollectionAudits = append(r.deleteCollectionAudits, auditEvents...)
	return int64(len(r.documentsByCollection)), r.deleteCollectionErr
}

type collectionRepoStub struct {
	collection *models.Collection
}

func (r *collectionRepoStub) CreateCollection(context.Context, *models.Collection) error {
	return nil
}

func (r *collectionRepoStub) UpdateCollection(context.Context, *models.Collection) error {
	return nil
}

func (r *collectionRepoStub) GetCollectionByID(context.Context, uint, uint) (*models.Collection, error) {
	if r.collection == nil {
		return nil, domain.ErrCollectionNotFound
	}
	return r.collection, nil
}

func (r *collectionRepoStub) GetCollectionByNormalizedName(context.Context, string, uint) (*models.Collection, error) {
	return nil, domain.ErrCollectionNotFound
}

func (r *collectionRepoStub) ListCollectionsByUserID(context.Context, uint) ([]models.Collection, error) {
	return nil, nil
}

func (r *collectionRepoStub) CountDocuments(context.Context, uint, uint) (int64, error) {
	return 0, nil
}

type storageStub struct {
	deleteErrByPath map[string]error
}

func (s *storageStub) UploadFile(context.Context, io.Reader, int64, string, string) (string, error) {
	return "", nil
}

func (s *storageStub) OpenFile(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *storageStub) DeleteFile(_ context.Context, objectPath string) error {
	if s.deleteErrByPath == nil {
		return nil
	}
	return s.deleteErrByPath[objectPath]
}

func (s *storageStub) FileExists(context.Context, string) (bool, error) {
	return false, nil
}

type resolverStub struct {
	storage domain.StorageService
	err     error
}

func (r *resolverStub) ResolveStorage(context.Context, uint) (domain.StorageService, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.storage, nil
}

func TestDeleteDocumentReturnsErrorWhenStorageDeletionFailsAndAuditsFailure(t *testing.T) {
	t.Parallel()

	doc := &models.Document{ID: 10, UserID: 3, Title: "Contrato.pdf", FileURL: "docs/contrato.pdf"}
	repo := &documentRepoStub{document: doc}
	useCase := &documentCatalogUseCase{
		documentRepo:    repo,
		storage:         &storageStub{deleteErrByPath: map[string]error{"docs/contrato.pdf": errors.New("boom")}},
		storageResolver: &resolverStub{err: domain.ErrStorageProviderNotConfigured},
	}

	err := useCase.DeleteDocument(context.Background(), 3, 10)
	if !errors.Is(err, domain.ErrDocumentSourceDeleteFailed) {
		t.Fatalf("expected source delete failure, got %v", err)
	}
	if repo.deleteDocumentCalled {
		t.Fatal("expected repository delete not to be called after storage failure")
	}
	if len(repo.recordedAuditEvents) != 1 {
		t.Fatalf("expected one failure audit event, got %d", len(repo.recordedAuditEvents))
	}
	if repo.recordedAuditEvents[0].Status != "FAILED" {
		t.Fatalf("expected failed audit status, got %#v", repo.recordedAuditEvents[0])
	}
}

func TestDeleteDocumentPersistsSuccessAuditInsideDeleteTransaction(t *testing.T) {
	t.Parallel()

	doc := &models.Document{ID: 11, UserID: 4, Title: "Relatorio.pdf", FileURL: "docs/relatorio.pdf"}
	repo := &documentRepoStub{document: doc}
	useCase := &documentCatalogUseCase{
		documentRepo:    repo,
		storage:         &storageStub{},
		storageResolver: &resolverStub{err: domain.ErrStorageProviderNotConfigured},
	}

	err := useCase.DeleteDocument(context.Background(), 4, 11)
	if err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if !repo.deleteDocumentCalled || repo.deleteDocumentAudit == nil {
		t.Fatal("expected repository delete to receive an audit event")
	}
	if repo.deleteDocumentAudit.Status != "SUCCESS" {
		t.Fatalf("expected success audit status, got %#v", repo.deleteDocumentAudit)
	}
}

func TestDeleteCollectionDocumentsMarksPartialRemovalsAsOrphanedAndAuditsFailure(t *testing.T) {
	t.Parallel()

	collectionID := uint(22)
	documents := []models.Document{
		{ID: 1, UserID: 8, Title: "A.pdf", FileURL: "docs/a.pdf", CollectionID: &collectionID},
		{ID: 2, UserID: 8, Title: "B.pdf", FileURL: "docs/b.pdf", CollectionID: &collectionID},
	}
	repo := &documentRepoStub{documentsByCollection: documents}
	useCase := &documentCatalogUseCase{
		collectionRepo:  &collectionRepoStub{collection: &models.Collection{ID: collectionID, UserID: 8, Name: "Lote"}},
		documentRepo:    repo,
		storage:         &storageStub{deleteErrByPath: map[string]error{"docs/b.pdf": errors.New("storage down")}},
		storageResolver: &resolverStub{err: domain.ErrStorageProviderNotConfigured},
	}

	_, _, err := useCase.DeleteCollectionDocuments(context.Background(), 8, collectionID)
	if !errors.Is(err, domain.ErrDocumentSourceDeleteFailed) {
		t.Fatalf("expected source delete failure, got %v", err)
	}
	if repo.deleteCollectionCalled {
		t.Fatal("expected repository bulk delete not to be called after storage failure")
	}
	if len(repo.markOrphanedDocumentIDs) != 1 || repo.markOrphanedDocumentIDs[0] != 1 {
		t.Fatalf("expected first partially removed document to be marked orphaned, got %#v", repo.markOrphanedDocumentIDs)
	}
	if len(repo.recordedAuditEvents) != 2 {
		t.Fatalf("expected failure audits for the failed and partially removed documents, got %d", len(repo.recordedAuditEvents))
	}
}
