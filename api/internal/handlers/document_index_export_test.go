package handlers

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"thing-index/internal/api/middleware"
	"thing-index/internal/domain"
	"thing-index/internal/indexio"
	"thing-index/internal/models"

	"github.com/gin-gonic/gin"
)

func TestExportDocumentIndexReturnsVersionedAttachment(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{
		documentIndex: &domain.DocumentIndex{
			Document: &models.Document{
				ID:       7,
				Title:    "Contrato final.pdf",
				MimeType: "application/pdf",
				Status:   domain.DocumentStatusIndexed,
			},
			DocumentPages: []models.DocumentPage{
				{DocumentID: 7, PageNumber: 1, Content: "Page 1"},
			},
			ParagraphNodes: []models.ParagraphNode{
				{DocumentID: 7, SequenceOrder: 1, Content: "Paragraph A"},
			},
			DocumentSummary: "Resumo",
		},
	}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(middleware.RequestContextMiddleware())
	router.Use(func(c *gin.Context) {
		c.Set("language", "en-US")
		c.Set("user_id", uint(42))
		c.Next()
	})
	router.GET("/api/v1/documents/:id/index/export", handler.ExportDocumentIndex)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/7/index/export", nil)
	req.Header.Set(indexio.DocumentIndexExportOriginHeader, "https://thingindex.dev/documents/7")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d with body %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, "Contrato") {
		t.Fatalf("expected attachment content disposition, got %q", got)
	}

	var exported indexio.DocumentIndexExportEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export response: %v", err)
	}
	if err := exported.Validate(); err != nil {
		t.Fatalf("expected valid export payload, got %v", err)
	}
	if exported.Source.Origin != "https://thingindex.dev" {
		t.Fatalf("expected normalized origin, got %q", exported.Source.Origin)
	}
	if exported.Source.ExportRoute != "/api/v1/documents/7/index/export" {
		t.Fatalf("unexpected export route %q", exported.Source.ExportRoute)
	}
	if exported.DocumentIndex == nil || exported.DocumentIndex.Document == nil || exported.DocumentIndex.Document.Title != "Contrato final.pdf" {
		t.Fatalf("unexpected exported document %#v", exported.DocumentIndex)
	}
}

func TestExportDocumentIndexUsesHostFallbackForOrigin(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	useCase := &documentUseCaseAuditCaptureStub{
		documentIndex: &domain.DocumentIndex{
			Document: &models.Document{
				ID:       11,
				Title:    "Policy.txt",
				MimeType: "text/plain",
				Status:   domain.DocumentStatusIndexed,
			},
		},
	}
	handler := NewDocumentHandler(useCase, 10<<20)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("language", "en-US")
		c.Set("user_id", uint(42))
		c.Next()
	})
	router.GET("/api/v1/documents/:id/index/export", handler.ExportDocumentIndex)

	req := httptest.NewRequest(http.MethodGet, "https://api.thingindex.dev/api/v1/documents/11/index/export", nil)
	req.Host = "api.thingindex.dev"
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	var exported indexio.DocumentIndexExportEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export response: %v", err)
	}
	if exported.Source.Origin != "https://api.thingindex.dev" {
		t.Fatalf("expected host-based origin fallback, got %q", exported.Source.Origin)
	}
}
