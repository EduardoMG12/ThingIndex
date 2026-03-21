package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"thing-index/internal/api/middleware"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/gin-gonic/gin"
)

type documentUseCaseAuditCaptureStub struct {
	lastUploadAuditContext           domain.AuditContext
	lastRetryAuditContext            domain.AuditContext
	lastDeleteAuditContext           domain.AuditContext
	lastCollectionDeleteAuditContext domain.AuditContext
	lastAuditFilter                  domain.DocumentAuditFilter
	documentContent                  *domain.DocumentContent
	documentContentErr               error
	uploadErr                        error
	uploadCalled                     bool
	retryCalled                      bool
	deleteDocumentCalled             bool
	deleteCollectionCalled           bool
}

func (s *documentUseCaseAuditCaptureStub) UploadDocument(ctx context.Context, _ uint, _ string, _ io.Reader, _ int64, _ string, _ string, _ string, _ string) (*models.Document, error) {
	s.uploadCalled = true
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastUploadAuditContext = auditCtx
	}
	if s.uploadErr != nil {
		return nil, s.uploadErr
	}
	return &models.Document{ID: 1, Title: "Arquivo.pdf"}, nil
}

func (s *documentUseCaseAuditCaptureStub) CreateCategory(context.Context, uint, string, string) (*models.DocumentCategory, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) UpdateCategory(context.Context, uint, uint, *string, *string) (*models.DocumentCategory, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) CreateCollection(context.Context, uint, string, *string, map[string]any) (*models.Collection, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) GetCollection(context.Context, uint, uint) (*models.Collection, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) UpdateCollectionCategory(context.Context, uint, string, *string, string) (int64, *models.Collection, error) {
	return 0, nil, nil
}

func (s *documentUseCaseAuditCaptureStub) ListCategories(context.Context, uint) ([]models.DocumentCategory, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) ListDocuments(context.Context, uint) ([]models.Document, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) ListDocumentAuditEvents(context.Context, uint, domain.DocumentAuditFilter) ([]models.DocumentAuditEvent, error) {
	return []models.DocumentAuditEvent{
		{
			ID:        1,
			Action:    "DOCUMENT_UPLOAD",
			Status:    "SUCCESS",
			ActorType: "USER",
			Title:     "Arquivo.pdf",
		},
	}, nil
}

func (s *documentUseCaseAuditCaptureStub) QueryDocumentAuditEvents(_ context.Context, _ uint, filter domain.DocumentAuditFilter) (*domain.DocumentAuditQueryResult, error) {
	s.lastAuditFilter = filter
	if filter.Cursor == "not-valid" {
		return nil, domain.ErrAuditCursorInvalid
	}
	return &domain.DocumentAuditQueryResult{
		Items: []models.DocumentAuditEvent{
			{
				ID:            1,
				Action:        "DOCUMENT_UPLOAD",
				Status:        "SUCCESS",
				ActorType:     "USER",
				Title:         "Arquivo.pdf",
				CorrelationID: "req-http-upload-001",
			},
		},
		Summary: domain.DocumentAuditSummary{
			TotalEvents: 1,
			ActionBreakdown: []domain.AuditCountBucket{
				{Key: "DOCUMENT_UPLOAD", Count: 1},
			},
			StatusBreakdown: []domain.AuditCountBucket{
				{Key: "SUCCESS", Count: 1},
			},
		},
		PageInfo: domain.AuditPageInfo{
			Limit:    1,
			Returned: 1,
		},
	}, nil
}

func (s *documentUseCaseAuditCaptureStub) GetDocument(context.Context, uint, uint) (*models.Document, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) GetDocumentContent(context.Context, uint, uint) (*domain.DocumentContent, error) {
	if s.documentContentErr != nil {
		return nil, s.documentContentErr
	}
	return s.documentContent, nil
}

func (s *documentUseCaseAuditCaptureStub) GetDocumentIndex(context.Context, uint, uint, string) (*domain.DocumentIndex, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) GetDocumentMonitoringSummary(context.Context, uint) (*domain.DocumentMonitoringSummary, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) RetryDocumentIndexing(ctx context.Context, _ uint, _ uint) (*models.Document, error) {
	s.retryCalled = true
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastRetryAuditContext = auditCtx
	}
	return &models.Document{ID: 9, Title: "Arquivo.pdf"}, nil
}

func (s *documentUseCaseAuditCaptureStub) DeleteDocument(ctx context.Context, _ uint, _ uint) error {
	s.deleteDocumentCalled = true
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastDeleteAuditContext = auditCtx
	}
	return nil
}

func (s *documentUseCaseAuditCaptureStub) DeleteCollectionDocuments(ctx context.Context, _ uint, _ uint) (int64, *models.Collection, error) {
	s.deleteCollectionCalled = true
	if auditCtx, ok := domain.ExtractAuditContext(ctx); ok {
		s.lastCollectionDeleteAuditContext = auditCtx
	}
	return 2, &models.Collection{ID: 7, Name: "Colecao"}, nil
}

func (s *documentUseCaseAuditCaptureStub) ListDocumentsByCategory(context.Context, uint, uint) ([]models.Document, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) ListCollections(context.Context, uint) ([]models.Collection, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) GetCollectionContext(context.Context, uint, uint, string) (*domain.CollectionContext, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) StoreParagraphNodes(context.Context, uint, uint, []models.ParagraphNode) ([]models.ParagraphNode, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) StoreTableSections(context.Context, uint, uint, []models.TableSection) ([]models.TableSection, error) {
	return nil, nil
}

func (s *documentUseCaseAuditCaptureStub) StoreMetadataPairs(context.Context, uint, uint, []models.MetadataPair) ([]models.MetadataPair, error) {
	return nil, nil
}

func TestDeleteDocumentEndpointAddsAuditContextFromRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(42))
		c.Next()
	})
	router.DELETE("/api/v1/documents/:id", handler.DeleteDocument)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/9", nil)
	req.Header.Set("X-Request-ID", "req-http-delete-001")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if !useCase.deleteDocumentCalled {
		t.Fatal("expected delete document use case to be called")
	}
	if useCase.lastDeleteAuditContext.CorrelationID != "req-http-delete-001" {
		t.Fatalf("expected request id to propagate, got %#v", useCase.lastDeleteAuditContext)
	}
	if useCase.lastDeleteAuditContext.Origin != "DELETE /api/v1/documents/:id" {
		t.Fatalf("expected origin to include method and route, got %#v", useCase.lastDeleteAuditContext)
	}
	if useCase.lastDeleteAuditContext.ActorID == nil || *useCase.lastDeleteAuditContext.ActorID != 42 {
		t.Fatalf("expected actor id to propagate, got %#v", useCase.lastDeleteAuditContext)
	}
}

func TestUploadDocumentEndpointAddsAuditContextFromRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.POST("/api/v1/documents/upload", handler.Upload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nArquivo\r\n--x\r\nContent-Disposition: form-data; name=\"collection\"\r\n\r\nColecao\r\n--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"arquivo.pdf\"\r\nContent-Type: application/pdf\r\n\r\n%PDF-1.7\r\n--x--\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req.Header.Set("X-Request-ID", "req-http-upload-001")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", rec.Code)
	}
	if !useCase.uploadCalled {
		t.Fatal("expected upload use case to be called")
	}
	if useCase.lastUploadAuditContext.CorrelationID != "req-http-upload-001" {
		t.Fatalf("expected request id to propagate, got %#v", useCase.lastUploadAuditContext)
	}
	if useCase.lastUploadAuditContext.Origin != "POST /api/v1/documents/upload" {
		t.Fatalf("expected upload origin to include route, got %#v", useCase.lastUploadAuditContext)
	}
}

func TestUploadDocumentEndpointReturnsInvalidMimeError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{uploadErr: domain.ErrDocumentInvalidMime}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.POST("/api/v1/documents/upload", handler.Upload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nArquivo\r\n--x\r\nContent-Disposition: form-data; name=\"collection\"\r\n\r\nColecao\r\n--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"arquivo.exe\"\r\nContent-Type: application/octet-stream\r\n\r\nMZ\r\n--x--\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", rec.Code)
	}
}

func TestUploadDocumentEndpointReturnsPayloadTooLargeWhenBodyExceedsLimit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 64)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.POST("/api/v1/documents/upload", handler.Upload)

	body := "--x\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nArquivo\r\n--x\r\nContent-Disposition: form-data; name=\"collection\"\r\n\r\nColecao\r\n--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"arquivo.pdf\"\r\nContent-Type: application/pdf\r\n\r\n%PDF-1.7 lorem ipsum dolor sit amet\r\n--x--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 response, got %d", rec.Code)
	}
	if useCase.uploadCalled {
		t.Fatal("expected upload use case not to be called when body exceeds limit")
	}
}

func TestRetryDocumentEndpointAddsAuditContextFromRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(31))
		c.Next()
	})
	router.POST("/api/v1/documents/:id/retry", handler.RetryIndexing)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/9/retry", nil)
	req.Header.Set("X-Request-ID", "req-http-retry-001")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if !useCase.retryCalled {
		t.Fatal("expected retry use case to be called")
	}
	if useCase.lastRetryAuditContext.CorrelationID != "req-http-retry-001" {
		t.Fatalf("expected request id to propagate, got %#v", useCase.lastRetryAuditContext)
	}
	if useCase.lastRetryAuditContext.Origin != "POST /api/v1/documents/:id/retry" {
		t.Fatalf("expected retry origin to include route, got %#v", useCase.lastRetryAuditContext)
	}
}

func TestDeleteCollectionDocumentsEndpointAddsAuditContextFromRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(17))
		c.Next()
	})
	router.DELETE("/api/v1/documents/collections/:id/documents", handler.DeleteCollectionDocuments)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/collections/7/documents", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if !useCase.deleteCollectionCalled {
		t.Fatal("expected delete collection documents use case to be called")
	}
	if useCase.lastCollectionDeleteAuditContext.CorrelationID == "" {
		t.Fatalf("expected generated request id, got %#v", useCase.lastCollectionDeleteAuditContext)
	}
	if useCase.lastCollectionDeleteAuditContext.Origin != "DELETE /api/v1/documents/collections/:id/documents" {
		t.Fatalf("expected origin to include collection delete route, got %#v", useCase.lastCollectionDeleteAuditContext)
	}
	if useCase.lastCollectionDeleteAuditContext.ActorID == nil || *useCase.lastCollectionDeleteAuditContext.ActorID != 17 {
		t.Fatalf("expected actor id to propagate, got %#v", useCase.lastCollectionDeleteAuditContext)
	}
}

func TestGetDocumentContentReturnsGoneForMissingSource(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{documentContentErr: domain.ErrDocumentSourceMissing}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(17))
		c.Next()
	})
	router.GET("/api/v1/documents/:id/content", handler.GetDocumentContent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/7/content", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 response, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "nao foi encontrado no storage") {
		t.Fatalf("expected orphaned source message, got %q", rec.Body.String())
	}
}

func TestGetDocumentContentStreamsInlinePDFResponse(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{
		documentContent: &domain.DocumentContent{
			Document: &models.Document{
				ID:       7,
				Title:    "Contrato.pdf",
				MimeType: "application/pdf",
				Size:     12,
			},
			Reader: io.NopCloser(strings.NewReader("pdf-content")),
		},
	}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(17))
		c.Next()
	})
	router.GET("/api/v1/documents/:id/content", handler.GetDocumentContent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/7/content", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected pdf content type, got %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "inline") || !strings.Contains(got, "Contrato.pdf") {
		t.Fatalf("expected inline content disposition for pdf, got %q", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("expected no-referrer policy, got %q", got)
	}
	if rec.Body.String() != "pdf-content" {
		t.Fatalf("unexpected streamed body: %q", rec.Body.String())
	}
}

func TestUploadDocumentEndpointReturnsConflictForUnavailableStorageProvider(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{uploadErr: domain.ErrStorageProviderUnavailable}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.POST("/api/v1/documents/upload", handler.Upload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", strings.NewReader("--x\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nArquivo\r\n--x\r\nContent-Disposition: form-data; name=\"collection\"\r\n\r\nColecao\r\n--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"arquivo.pdf\"\r\nContent-Type: application/pdf\r\n\r\n%PDF-1.7\r\n--x--\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 response, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "storage") {
		t.Fatalf("expected storage provider message, got %q", rec.Body.String())
	}
}

func TestListDocumentAuditEventsEndpointAcceptsFilters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.GET("/api/v1/documents/audit", handler.ListDocumentAuditEvents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/audit?action=document_upload&status=success&document_id=7&collection_id=3&actor_type=user&actor_id=9&correlation_id=req-77&q=arquivo&created_from=2026-03-20T10:00:00Z&created_to=2026-03-20T12:00:00Z&limit=5", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "DOCUMENT_UPLOAD") {
		t.Fatalf("expected document audit payload, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"page_info\"") {
		t.Fatalf("expected document audit page info, got %q", rec.Body.String())
	}
	if useCase.lastAuditFilter.Action != "document_upload" || useCase.lastAuditFilter.DocumentID == nil || *useCase.lastAuditFilter.DocumentID != 7 {
		t.Fatalf("expected audit filters to propagate, got %#v", useCase.lastAuditFilter)
	}
	if useCase.lastAuditFilter.ActorID == nil || *useCase.lastAuditFilter.ActorID != 9 {
		t.Fatalf("expected actor id filter to propagate, got %#v", useCase.lastAuditFilter)
	}
	if useCase.lastAuditFilter.CreatedFrom == nil || useCase.lastAuditFilter.CreatedTo == nil {
		t.Fatalf("expected created_at range filters to propagate, got %#v", useCase.lastAuditFilter)
	}
}

func TestListDocumentAuditEventsEndpointRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "pt_BR")
		c.Set("user_id", uint(23))
		c.Next()
	})
	router.GET("/api/v1/documents/audit", handler.ListDocumentAuditEvents)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/audit?cursor=not-valid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "cursor") {
		t.Fatalf("expected invalid cursor message, got %q", rec.Body.String())
	}
}
