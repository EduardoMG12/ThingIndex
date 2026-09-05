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

type documentRepoStub struct {
	document                *models.Document
	documentsByCollection   []models.Document
	createDocumentCalled    bool
	createDocumentAudit     *models.DocumentAuditEvent
	retryDocumentAudit      *models.DocumentAuditEvent
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

func (r *documentRepoStub) CreateDocumentWithOutbox(_ context.Context, doc *models.Document, _ *models.OutboxEvent, auditEvent *models.DocumentAuditEvent) error {
	r.createDocumentCalled = true
	if doc.ID == 0 {
		doc.ID = 99
	}
	if auditEvent != nil {
		copied := *auditEvent
		documentID := doc.ID
		copied.DocumentID = &documentID
		r.createDocumentAudit = &copied
	}
	return nil
}

func (r *documentRepoStub) CreateDocumentAuditEvents(_ context.Context, events []models.DocumentAuditEvent) error {
	r.recordedAuditEvents = append(r.recordedAuditEvents, events...)
	return nil
}

func (r *documentRepoStub) ListDocumentAuditEvents(_ context.Context, _ uint, _ domain.DocumentAuditFilter) ([]models.DocumentAuditEvent, error) {
	return r.recordedAuditEvents, nil
}

func (r *documentRepoStub) QueryDocumentAuditEvents(_ context.Context, _ uint, _ domain.DocumentAuditFilter) (*domain.DocumentAuditQueryResult, error) {
	return &domain.DocumentAuditQueryResult{
		Items: r.recordedAuditEvents,
		PageInfo: domain.AuditPageInfo{
			Limit:    len(r.recordedAuditEvents),
			Returned: len(r.recordedAuditEvents),
		},
		Summary: domain.DocumentAuditSummary{
			TotalEvents: int64(len(r.recordedAuditEvents)),
		},
	}, nil
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

func (r *documentRepoStub) RetryDocumentIndexing(_ context.Context, _ uint, documentID uint, auditEvent *models.DocumentAuditEvent) (*models.Document, error) {
	if auditEvent != nil {
		copied := *auditEvent
		copied.DocumentID = &documentID
		r.retryDocumentAudit = &copied
	}
	if r.document == nil {
		return nil, domain.ErrDocumentNotFound
	}
	return r.document, nil
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
	collection       *models.Collection
	collectionByName *models.Collection
}

type categoryRepoStub struct{}

func (r *categoryRepoStub) CreateCategory(context.Context, *models.DocumentCategory) error {
	return nil
}

func (r *categoryRepoStub) UpdateCategory(context.Context, *models.DocumentCategory) error {
	return nil
}

func (r *categoryRepoStub) GetCategoryByID(context.Context, uint, uint) (*models.DocumentCategory, error) {
	return nil, domain.ErrCategoryNotFound
}

func (r *categoryRepoStub) GetCategoryByNormalizedName(context.Context, string, uint) (*models.DocumentCategory, error) {
	return nil, domain.ErrCategoryNotFound
}

func (r *categoryRepoStub) ListCategoriesByUserID(context.Context, uint) ([]models.DocumentCategory, error) {
	return nil, nil
}

func (r *collectionRepoStub) CreateCollection(context.Context, *models.Collection) error {
	return nil
}

// EnsureCollection espelha CreateCollection no stub: a resolucao da corrida
// acontece no banco, e o teste de concorrencia real vive em
// internal/repository (com a tag integration).
func (r *collectionRepoStub) EnsureCollection(context.Context, *models.Collection) error {
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
	if r.collectionByName == nil {
		return nil, domain.ErrCollectionNotFound
	}
	return r.collectionByName, nil
}

func (r *collectionRepoStub) ListCollectionsByUserID(context.Context, uint) ([]models.Collection, error) {
	return nil, nil
}

func (r *collectionRepoStub) CountDocuments(context.Context, uint, uint) (int64, error) {
	return 0, nil
}

type storageStub struct {
	deleteErrByPath map[string]error
	uploadURL       string
	uploadCalls     int
	openErrByPath   map[string]error
	openBodyByPath  map[string]string
	existsByPath    map[string]bool
	existsErrByPath map[string]error
}

func (s *storageStub) UploadFile(context.Context, io.Reader, int64, string, string) (string, error) {
	s.uploadCalls++
	if s.uploadURL != "" {
		return s.uploadURL, nil
	}
	return "", nil
}

func (s *storageStub) OpenFile(_ context.Context, objectPath string) (io.ReadCloser, error) {
	if s.openErrByPath != nil {
		if err, ok := s.openErrByPath[objectPath]; ok {
			return nil, err
		}
	}
	if s.openBodyByPath != nil {
		if body, ok := s.openBodyByPath[objectPath]; ok {
			return io.NopCloser(strings.NewReader(body)), nil
		}
	}
	return nil, nil
}

func (s *storageStub) DeleteFile(_ context.Context, objectPath string) error {
	if s.deleteErrByPath == nil {
		return nil
	}
	return s.deleteErrByPath[objectPath]
}

func (s *storageStub) FileExists(_ context.Context, objectPath string) (bool, error) {
	if s.existsErrByPath != nil {
		if err, ok := s.existsErrByPath[objectPath]; ok {
			return false, err
		}
	}
	if s.existsByPath != nil {
		if exists, ok := s.existsByPath[objectPath]; ok {
			return exists, nil
		}
	}
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

func TestUploadDocumentPersistsSuccessAuditInsideCreateTransaction(t *testing.T) {
	t.Parallel()

	collectionID := uint(13)
	actorID := uint(4)
	repo := &documentRepoStub{}
	useCase := &documentCatalogUseCase{
		categoryRepo:       &categoryRepoStub{},
		collectionRepo:     &collectionRepoStub{collectionByName: &models.Collection{ID: collectionID, UserID: 4, Name: "Contratos", NormalizedName: "contratos"}},
		documentRepo:       repo,
		storage:            &storageStub{uploadURL: "docs/contrato.pdf"},
		storageResolver:    &resolverStub{err: domain.ErrStorageProviderNotConfigured},
		maxUploadSizeBytes: 10 << 20,
	}

	ctx := domain.WithAuditContext(context.Background(), domain.AuditContext{
		ActorType:     "USER",
		ActorID:       &actorID,
		CorrelationID: "req-upload-001",
		Origin:        "POST /api/v1/documents/upload",
		RequestMethod: "POST",
		RoutePath:     "/api/v1/documents/upload",
		RemoteAddr:    "127.0.0.1",
	})

	doc, err := useCase.UploadDocument(ctx, 4, "Contrato.pdf", io.NopCloser(strings.NewReader("%PDF-1.7 sample")), 15, "application/pdf", "", "", "Contratos")
	if err != nil {
		t.Fatalf("expected upload to succeed, got %v", err)
	}
	if doc == nil || !repo.createDocumentCalled || repo.createDocumentAudit == nil {
		t.Fatal("expected document creation to persist an audit event")
	}
	if repo.createDocumentAudit.Action != "DOCUMENT_UPLOAD" || repo.createDocumentAudit.Status != "SUCCESS" {
		t.Fatalf("unexpected upload audit event: %#v", repo.createDocumentAudit)
	}
	if repo.createDocumentAudit.CorrelationID != "req-upload-001" || repo.createDocumentAudit.Origin != "POST /api/v1/documents/upload" {
		t.Fatalf("expected correlation and origin in upload audit, got %#v", repo.createDocumentAudit)
	}
	if repo.createDocumentAudit.Metadata["mime_type"] != "application/pdf" {
		t.Fatalf("expected mime type metadata, got %#v", repo.createDocumentAudit.Metadata)
	}
}

func TestRetryDocumentIndexingPersistsSuccessAudit(t *testing.T) {
	t.Parallel()

	collectionID := uint(22)
	actorID := uint(9)
	doc := &models.Document{ID: 21, UserID: 9, Title: "Acordao.pdf", CollectionID: &collectionID}
	repo := &documentRepoStub{document: doc}
	useCase := &documentCatalogUseCase{documentRepo: repo}

	ctx := domain.WithAuditContext(context.Background(), domain.AuditContext{
		ActorType:     "USER",
		ActorID:       &actorID,
		CorrelationID: "req-retry-001",
		Origin:        "POST /api/v1/documents/:id/retry",
		RequestMethod: "POST",
		RoutePath:     "/api/v1/documents/:id/retry",
		RemoteAddr:    "127.0.0.1",
	})

	_, err := useCase.RetryDocumentIndexing(ctx, 9, 21)
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if repo.retryDocumentAudit == nil {
		t.Fatal("expected retry audit to be persisted")
	}
	if repo.retryDocumentAudit.Action != "DOCUMENT_RETRY_INDEX" || repo.retryDocumentAudit.Status != "SUCCESS" {
		t.Fatalf("unexpected retry audit event: %#v", repo.retryDocumentAudit)
	}
	if repo.retryDocumentAudit.CorrelationID != "req-retry-001" {
		t.Fatalf("expected retry correlation id, got %#v", repo.retryDocumentAudit)
	}
}

func TestDeleteDocumentPersistsSuccessAuditInsideDeleteTransaction(t *testing.T) {
	t.Parallel()

	doc := &models.Document{ID: 11, UserID: 4, Title: "Relatorio.pdf", FileURL: "docs/relatorio.pdf"}
	repo := &documentRepoStub{document: doc}
	actorID := uint(4)
	ctx := domain.WithAuditContext(context.Background(), domain.AuditContext{
		ActorType:     "USER",
		ActorID:       &actorID,
		CorrelationID: "req-delete-001",
		Origin:        "DELETE /api/v1/documents/:id",
		RequestMethod: "DELETE",
		RoutePath:     "/api/v1/documents/:id",
		RemoteAddr:    "127.0.0.1",
	})
	useCase := &documentCatalogUseCase{
		documentRepo:    repo,
		storage:         &storageStub{},
		storageResolver: &resolverStub{err: domain.ErrStorageProviderNotConfigured},
	}

	err := useCase.DeleteDocument(ctx, 4, 11)
	if err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if !repo.deleteDocumentCalled || repo.deleteDocumentAudit == nil {
		t.Fatal("expected repository delete to receive an audit event")
	}
	if repo.deleteDocumentAudit.Status != "SUCCESS" {
		t.Fatalf("expected success audit status, got %#v", repo.deleteDocumentAudit)
	}
	if repo.deleteDocumentAudit.CorrelationID != "req-delete-001" || repo.deleteDocumentAudit.Origin != "DELETE /api/v1/documents/:id" {
		t.Fatalf("expected correlation and origin metadata, got %#v", repo.deleteDocumentAudit)
	}
	if repo.deleteDocumentAudit.ActorID == nil || *repo.deleteDocumentAudit.ActorID != actorID {
		t.Fatalf("expected actor id in audit event, got %#v", repo.deleteDocumentAudit)
	}
	if repo.deleteDocumentAudit.Metadata["route_path"] != "/api/v1/documents/:id" {
		t.Fatalf("expected route path in audit metadata, got %#v", repo.deleteDocumentAudit.Metadata)
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
