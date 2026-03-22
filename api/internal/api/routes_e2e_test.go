//go:build e2e

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	providerpkg "thing-index/internal/pkg/provider"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var loadAPIE2EEnvOnce sync.Once

func TestRoutesE2ERegisterLoginUploadListAndDownloadContent(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Routes E2E",
		"email":    "routes-e2e@example.com",
		"password": "secret123",
	}, "")
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d body=%s", registerResponse.Code, registerResponse.Body.String())
	}

	loginResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "routes-e2e@example.com",
		"password": "secret123",
	}, "")
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d body=%s", loginResponse.Code, loginResponse.Body.String())
	}
	token := extractTokenFromResponse(t, loginResponse.Body.Bytes())
	if token == "" {
		t.Fatalf("expected non-empty auth token")
	}

	uploadedBody := []byte("<html><body>ThingIndex E2E</body></html>")
	uploadResponse := performMultipartUploadRequest(t, router, "/api/v1/documents/upload", token, map[string]string{
		"title":      "routes-e2e.html",
		"category":   "Policies",
		"collection": "Operations",
	}, "file", "routes-e2e.html", "text/html", uploadedBody)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}
	uploadedDocument := extractDocumentFromResponse(t, uploadResponse.Body.Bytes())
	if uploadedDocument.ID == 0 {
		t.Fatalf("expected uploaded document ID")
	}
	if uploadedDocument.Status != "QUEUED" {
		t.Fatalf("expected uploaded document status QUEUED, got %q", uploadedDocument.Status)
	}

	listResponse := performAuthorizedRequest(t, router, http.MethodGet, "/api/v1/documents/", token, nil, "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}
	listedDocuments := extractDocumentsFromResponse(t, listResponse.Body.Bytes())
	if len(listedDocuments) != 1 {
		t.Fatalf("expected 1 listed document, got %d", len(listedDocuments))
	}
	if listedDocuments[0].ID != uploadedDocument.ID {
		t.Fatalf("expected listed document ID %d, got %d", uploadedDocument.ID, listedDocuments[0].ID)
	}

	contentResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/documents/%d/content", uploadedDocument.ID), token, nil, "")
	if contentResponse.Code != http.StatusOK {
		t.Fatalf("expected content status 200, got %d body=%s", contentResponse.Code, contentResponse.Body.String())
	}
	if got := contentResponse.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected content type to contain text/html, got %q", got)
	}
	if got := contentResponse.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="routes-e2e.html"`) {
		t.Fatalf("expected content disposition with filename, got %q", got)
	}
	if contentResponse.Body.String() != string(uploadedBody) {
		t.Fatalf("expected downloaded body %q, got %q", string(uploadedBody), contentResponse.Body.String())
	}
}

func TestRoutesE2EAPIDocsExposeFallbackReferenceLinks(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	docsResponse := performAuthorizedRequest(t, router, http.MethodGet, "/docs", "", nil, "")
	if docsResponse.Code != http.StatusOK {
		t.Fatalf("expected docs status 200, got %d body=%s", docsResponse.Code, docsResponse.Body.String())
	}
	if got := docsResponse.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected docs content type text/html, got %q", got)
	}

	body := docsResponse.Body.String()
	for _, expected := range []string{
		"ThingIndex API Reference",
		"http://example.com/docs",
		"http://example.com/swagger/index.html",
		"http://example.com/swagger/doc.json",
		"http://example.com/api/v1",
		"Consume this deployment",
		"Run locally from GitHub",
		"api/docs/swagger.yaml",
		"docs/open-source-onboarding.md",
		"examples/http/thingindex-api.http",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected docs page to contain %q", expected)
		}
	}

	swaggerUIResponse := performAuthorizedRequest(t, router, http.MethodGet, "/swagger/index.html", "", nil, "")
	if swaggerUIResponse.Code != http.StatusOK {
		t.Fatalf("expected swagger ui status 200, got %d body=%s", swaggerUIResponse.Code, swaggerUIResponse.Body.String())
	}
	if got := swaggerUIResponse.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected swagger ui content type text/html, got %q", got)
	}

	swaggerResponse := performAuthorizedRequest(t, router, http.MethodGet, "/swagger/doc.json", "", nil, "")
	if swaggerResponse.Code != http.StatusOK {
		t.Fatalf("expected swagger json status 200, got %d body=%s", swaggerResponse.Code, swaggerResponse.Body.String())
	}
	if !strings.Contains(swaggerResponse.Body.String(), `"swagger": "2.0"`) {
		t.Fatalf("expected swagger json payload, got %s", swaggerResponse.Body.String())
	}
}

func TestRoutesE2ERetryRequeuesDocumentAndAppendsOutboxEvent(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token, uploadedDocument := registerAndUploadHTMLDocument(t, router)
	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	indexedAt := time.Now().UTC().Add(-time.Hour)

	if err := db.Model(&models.Document{}).
		Where("id = ?", uploadedDocument.ID).
		Updates(map[string]any{
			"status":                domain.DocumentStatusFailed,
			"processing_stage":      "FAILED",
			"status_detail":         "pipeline failed",
			"last_processing_error": "pipeline failed",
			"index_started_at":      startedAt,
			"indexed_at":            indexedAt,
		}).Error; err != nil {
		t.Fatalf("seed failed document state: %v", err)
	}

	var outboxCountBefore int64
	if err := db.Model(&models.OutboxEvent{}).Where("aggregate_id = ?", uploadedDocument.ID).Count(&outboxCountBefore).Error; err != nil {
		t.Fatalf("count outbox before retry: %v", err)
	}

	retryResponse := performAuthorizedRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/documents/%d/retry", uploadedDocument.ID), token, nil, "")
	if retryResponse.Code != http.StatusOK {
		t.Fatalf("expected retry status 200, got %d body=%s", retryResponse.Code, retryResponse.Body.String())
	}
	retriedDocument := extractDocumentFromResponse(t, retryResponse.Body.Bytes())
	if retriedDocument.Status != domain.DocumentStatusQueued {
		t.Fatalf("expected retried document status %q, got %q", domain.DocumentStatusQueued, retriedDocument.Status)
	}

	storedDocument := mustLoadRoutesE2EDocument(t, db, uploadedDocument.ID)
	if storedDocument.Status != domain.DocumentStatusQueued {
		t.Fatalf("expected stored status %q, got %q", domain.DocumentStatusQueued, storedDocument.Status)
	}
	if storedDocument.IndexStartedAt != nil || storedDocument.IndexedAt != nil {
		t.Fatalf("expected stored retry timestamps cleared, got started=%v indexed=%v", storedDocument.IndexStartedAt, storedDocument.IndexedAt)
	}

	var outboxCountAfter int64
	if err := db.Model(&models.OutboxEvent{}).Where("aggregate_id = ?", uploadedDocument.ID).Count(&outboxCountAfter).Error; err != nil {
		t.Fatalf("count outbox after retry: %v", err)
	}
	if outboxCountAfter != outboxCountBefore+1 {
		t.Fatalf("expected outbox count %d after retry, got %d", outboxCountBefore+1, outboxCountAfter)
	}
}

func TestRoutesE2EContentMissingMarksDocumentAsOrphaned(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token, uploadedDocument := registerAndUploadHTMLDocument(t, router)
	storedDocument := mustLoadRoutesE2EDocument(t, db, uploadedDocument.ID)
	minioClient := newRoutesE2EMinioClient(t, cfg)
	objectKey := strings.TrimPrefix(storedDocument.FileURL, cfg.MinioBucketName+"/")

	if err := minioClient.RemoveObject(context.Background(), cfg.MinioBucketName, objectKey, minio.RemoveObjectOptions{}); err != nil {
		t.Fatalf("RemoveObject returned error: %v", err)
	}

	contentResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/documents/%d/content", uploadedDocument.ID), token, nil, "")
	if contentResponse.Code != http.StatusGone {
		t.Fatalf("expected content status 410, got %d body=%s", contentResponse.Code, contentResponse.Body.String())
	}

	storedDocument = mustLoadRoutesE2EDocument(t, db, uploadedDocument.ID)
	if storedDocument.Status != domain.DocumentStatusOrphaned {
		t.Fatalf("expected document status %q, got %q", domain.DocumentStatusOrphaned, storedDocument.Status)
	}
	if storedDocument.ProcessingStage != "SOURCE_MISSING" {
		t.Fatalf("expected processing stage SOURCE_MISSING, got %q", storedDocument.ProcessingStage)
	}
	if !strings.Contains(storedDocument.StatusDetail, "nao foi encontrado") {
		t.Fatalf("expected orphan detail to mention missing source, got %q", storedDocument.StatusDetail)
	}
}

func TestRoutesE2EDeleteRemovesDocumentAndSourceObject(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token, uploadedDocument := registerAndUploadHTMLDocument(t, router)
	storedDocument := mustLoadRoutesE2EDocument(t, db, uploadedDocument.ID)

	deleteResponse := performAuthorizedRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/documents/%d", uploadedDocument.ID), token, nil, "")
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}

	listResponse := performAuthorizedRequest(t, router, http.MethodGet, "/api/v1/documents/", token, nil, "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", listResponse.Code, listResponse.Body.String())
	}
	if listedDocuments := extractDocumentsFromResponse(t, listResponse.Body.Bytes()); len(listedDocuments) != 0 {
		t.Fatalf("expected no listed documents after delete, got %d", len(listedDocuments))
	}

	contentResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/documents/%d/content", uploadedDocument.ID), token, nil, "")
	if contentResponse.Code != http.StatusNotFound {
		t.Fatalf("expected content status 404 after delete, got %d body=%s", contentResponse.Code, contentResponse.Body.String())
	}

	var deletedDocument models.Document
	err := db.First(&deletedDocument, uploadedDocument.ID).Error
	if err == nil {
		t.Fatalf("expected deleted document to be absent from database")
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("load deleted document returned unexpected error: %v", err)
	}

	var outboxCount int64
	if err := db.Model(&models.OutboxEvent{}).Where("aggregate_id = ?", uploadedDocument.ID).Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox after delete: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected outbox events to be removed after delete, got %d", outboxCount)
	}

	minioClient := newRoutesE2EMinioClient(t, cfg)
	objectKey := strings.TrimPrefix(storedDocument.FileURL, cfg.MinioBucketName+"/")
	_, err = minioClient.StatObject(context.Background(), cfg.MinioBucketName, objectKey, minio.StatObjectOptions{})
	if err == nil {
		t.Fatalf("expected source object to be removed from minio")
	}
	errorResponse := minio.ToErrorResponse(err)
	if errorResponse.Code != "NoSuchKey" && errorResponse.Code != "NoSuchObject" && errorResponse.Code != "NotFound" {
		t.Fatalf("expected missing-object error after delete, got %v", err)
	}
}

func TestRoutesE2EDocumentAuditEndpointReturnsSummaryAndCursor(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token, uploadedDocument := registerAndUploadHTMLDocument(t, router)

	retryResponse := performAuthorizedRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/documents/%d/retry", uploadedDocument.ID), token, nil, "")
	if retryResponse.Code != http.StatusOK {
		t.Fatalf("expected retry status 200, got %d body=%s", retryResponse.Code, retryResponse.Body.String())
	}

	auditResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/documents/audit?document_id=%d&limit=1", uploadedDocument.ID), token, nil, "")
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected document audit status 200, got %d body=%s", auditResponse.Code, auditResponse.Body.String())
	}

	result := extractDocumentAuditQueryFromResponse(t, auditResponse.Body.Bytes())
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 document audit item on first page, got %d", len(result.Items))
	}
	if result.Summary.TotalEvents < 2 {
		t.Fatalf("expected at least 2 document audit events, got %#v", result.Summary)
	}
	if !result.PageInfo.HasNext || result.PageInfo.NextCursor == "" {
		t.Fatalf("expected document audit cursor, got %#v", result.PageInfo)
	}

	nextPageResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/documents/audit?document_id=%d&limit=1&cursor=%s", uploadedDocument.ID, result.PageInfo.NextCursor), token, nil, "")
	if nextPageResponse.Code != http.StatusOK {
		t.Fatalf("expected second document audit status 200, got %d body=%s", nextPageResponse.Code, nextPageResponse.Body.String())
	}

	nextPage := extractDocumentAuditQueryFromResponse(t, nextPageResponse.Body.Bytes())
	if len(nextPage.Items) != 1 {
		t.Fatalf("expected 1 remaining document audit item, got %d", len(nextPage.Items))
	}
}

func TestRoutesE2ERejectsSpoofedDocumentUpload(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token := registerRoutesE2EUser(t, router)
	uploadResponse := performMultipartUploadRequest(t, router, "/api/v1/documents/upload", token, map[string]string{
		"title":      "spoofed.pdf",
		"category":   "Policies",
		"collection": "Operations",
	}, "file", "spoofed.pdf", "application/pdf", []byte("<html><body>not a pdf</body></html>"))
	if uploadResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected upload status 400 for spoofed mime, got %d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}

	envelope := extractResponseEnvelope(t, uploadResponse.Body.Bytes())
	if envelope.Message != "Tipo de arquivo não permitido" {
		t.Fatalf("expected invalid mime message, got %#v", envelope)
	}

	var count int64
	if err := db.Model(&models.Document{}).Count(&count).Error; err != nil {
		t.Fatalf("count documents after invalid upload: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no persisted documents after invalid upload, got %d", count)
	}
}

func TestRoutesE2ERejectsOversizedDocumentUpload(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	cfg.MaxUploadSizeBytes = 128
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token := registerRoutesE2EUser(t, router)
	uploadResponse := performMultipartUploadRequest(t, router, "/api/v1/documents/upload", token, map[string]string{
		"title":      "oversized.html",
		"category":   "Policies",
		"collection": "Operations",
	}, "file", "oversized.html", "text/html", bytes.Repeat([]byte("A"), 1024))
	if uploadResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected upload status 400 for oversized payload rejected by the API, got %d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}

	envelope := extractResponseEnvelope(t, uploadResponse.Body.Bytes())
	if envelope.Message != "Arquivo excede o limite permitido" {
		t.Fatalf("expected file-too-large message, got %#v", envelope)
	}

	var count int64
	if err := db.Model(&models.Document{}).Count(&count).Error; err != nil {
		t.Fatalf("count documents after oversized upload: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no persisted documents after oversized upload, got %d", count)
	}
}

func TestRoutesE2EProviderChatLifecycleAndStreamWithFakeLLM(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	fakeLLM := newRoutesE2EFakeOpenAICompatibleServer(t)
	originalFactory := providerpkg.NewLLMHTTPClientForTests(nil)
	providerpkg.NewLLMHTTPClientForTests(func() *http.Client {
		client := fakeLLM.Client()
		client.Timeout = 15 * time.Second
		return client
	})
	t.Cleanup(func() {
		providerpkg.NewLLMHTTPClientForTests(originalFactory)
	})

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token, uploadedDocument := registerAndUploadHTMLDocument(t, router)
	storedDocument := mustLoadRoutesE2EDocument(t, db, uploadedDocument.ID)
	if storedDocument.CollectionID == nil {
		t.Fatal("expected uploaded document to be linked to a collection")
	}
	seedRoutesE2EChatContext(t, db, storedDocument.ID)

	providerCredentials := map[string]any{
		"api_key":  "routes-e2e-secret",
		"base_url": fakeLLM.URL,
		"model":    "routes-e2e-model",
	}

	testConnectionResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/providers/test", map[string]any{
		"family":        "LLM",
		"provider_type": "OPENAI",
		"credentials":   providerCredentials,
	}, token)
	if testConnectionResponse.Code != http.StatusOK {
		t.Fatalf("expected provider test-connection status 200, got %d body=%s", testConnectionResponse.Code, testConnectionResponse.Body.String())
	}
	readiness := extractProviderReadinessFromResponse(t, testConnectionResponse.Body.Bytes())
	if readiness.Status != domain.ProviderReadinessStatusReady {
		t.Fatalf("expected ready provider readiness, got %#v", readiness)
	}
	if len(readiness.DiscoveredModels) != 2 || readiness.DiscoveredModels[0] != "routes-e2e-model" {
		t.Fatalf("expected discovered models in readiness report, got %#v", readiness.DiscoveredModels)
	}

	saveProviderResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/providers", map[string]any{
		"family":           "LLM",
		"provider_type":    "OPENAI",
		"credentials":      providerCredentials,
		"enabled_for_chat": false,
	}, token)
	if saveProviderResponse.Code != http.StatusOK {
		t.Fatalf("expected provider save status 200, got %d body=%s", saveProviderResponse.Code, saveProviderResponse.Body.String())
	}

	listProvidersResponse := performAuthorizedRequest(t, router, http.MethodGet, "/api/v1/user/providers", token, nil, "")
	if listProvidersResponse.Code != http.StatusOK {
		t.Fatalf("expected provider list status 200, got %d body=%s", listProvidersResponse.Code, listProvidersResponse.Body.String())
	}
	providers := extractProviderConfigsFromResponse(t, listProvidersResponse.Body.Bytes())
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider config, got %d", len(providers))
	}
	if providers[0].ProviderType != domain.ProviderOpenAI {
		t.Fatalf("expected provider type %q, got %q", domain.ProviderOpenAI, providers[0].ProviderType)
	}
	if providers[0].EnabledForChat {
		t.Fatalf("expected chat provider to start disabled, got %#v", providers[0])
	}
	if providers[0].PublicMetadata["model"] != "routes-e2e-model" {
		t.Fatalf("expected public model metadata routes-e2e-model, got %#v", providers[0].PublicMetadata)
	}

	enableChatResponse := performJSONRequest(t, router, http.MethodPatch, "/api/v1/user/providers/OPENAI", map[string]any{
		"activation_target": "CHAT",
		"enabled":           true,
	}, token)
	if enableChatResponse.Code != http.StatusOK {
		t.Fatalf("expected provider enable-chat status 200, got %d body=%s", enableChatResponse.Code, enableChatResponse.Body.String())
	}

	listProvidersResponse = performAuthorizedRequest(t, router, http.MethodGet, "/api/v1/user/providers", token, nil, "")
	providers = extractProviderConfigsFromResponse(t, listProvidersResponse.Body.Bytes())
	if len(providers) != 1 || !providers[0].EnabledForChat {
		t.Fatalf("expected provider chat activation enabled after patch, got %#v", providers)
	}

	createConversationResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/chat/conversations", map[string]any{
		"collection_id": *storedDocument.CollectionID,
		"title":         "Prazos operacionais",
	}, token)
	if createConversationResponse.Code != http.StatusCreated {
		t.Fatalf("expected create conversation status 201, got %d body=%s", createConversationResponse.Code, createConversationResponse.Body.String())
	}
	conversation := extractConversationFromResponse(t, createConversationResponse.Body.Bytes())
	if conversation.ID == 0 {
		t.Fatal("expected conversation ID to be created")
	}

	listConversationsResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/chat/conversations?collection_id=%d", *storedDocument.CollectionID), token, nil, "")
	if listConversationsResponse.Code != http.StatusOK {
		t.Fatalf("expected list conversations status 200, got %d body=%s", listConversationsResponse.Code, listConversationsResponse.Body.String())
	}
	conversations := extractConversationsFromResponse(t, listConversationsResponse.Body.Bytes())
	if len(conversations) != 1 || conversations[0].ID != conversation.ID {
		t.Fatalf("expected one listed conversation %d, got %#v", conversation.ID, conversations)
	}

	updateConversationResponse := performJSONRequest(t, router, http.MethodPatch, fmt.Sprintf("/api/v1/chat/conversations/%d", conversation.ID), map[string]any{
		"title": "Prazos e atendimento",
	}, token)
	if updateConversationResponse.Code != http.StatusOK {
		t.Fatalf("expected update conversation status 200, got %d body=%s", updateConversationResponse.Code, updateConversationResponse.Body.String())
	}

	streamResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/chat/stream", map[string]any{
		"conversation_id": conversation.ID,
		"collection_id":   *storedDocument.CollectionID,
		"prompt":          "Qual e o prazo do relatorio mensal e do atendimento emergencial?",
		"history":         []map[string]any{},
	}, token)
	if streamResponse.Code != http.StatusOK {
		t.Fatalf("expected chat stream status 200, got %d body=%s", streamResponse.Code, streamResponse.Body.String())
	}
	streamBody := streamResponse.Body.String()
	for _, expected := range []string{
		"event: stage",
		`"stage":"retrieving_context"`,
		`"stage":"selecting_sources"`,
		"event: sources",
		`"docName":"routes-e2e.html"`,
		`"page":1`,
		`"stage":"generating_answer"`,
		"event: delta",
		"O relatorio mensal deve ser entregue ate o dia 5",
		"event: done",
	} {
		if !strings.Contains(streamBody, expected) {
			t.Fatalf("expected chat stream body to contain %q, got %s", expected, streamBody)
		}
	}

	getConversationResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/chat/conversations/%d", conversation.ID), token, nil, "")
	if getConversationResponse.Code != http.StatusOK {
		t.Fatalf("expected get conversation status 200, got %d body=%s", getConversationResponse.Code, getConversationResponse.Body.String())
	}
	conversationDetail := extractConversationDetailFromResponse(t, getConversationResponse.Body.Bytes())
	if conversationDetail.Conversation == nil || conversationDetail.Conversation.Title != "Prazos e atendimento" {
		t.Fatalf("expected updated conversation title, got %#v", conversationDetail.Conversation)
	}
	if len(conversationDetail.Messages) != 2 {
		t.Fatalf("expected 2 persisted messages after stream, got %#v", conversationDetail.Messages)
	}
	if conversationDetail.Messages[0].Role != "user" {
		t.Fatalf("expected first message role user, got %#v", conversationDetail.Messages[0])
	}
	if conversationDetail.Messages[1].Role != "assistant" {
		t.Fatalf("expected second message role assistant, got %#v", conversationDetail.Messages[1])
	}
	if !strings.Contains(conversationDetail.Messages[1].Content, "dia 5") {
		t.Fatalf("expected assistant message to mention dia 5, got %#v", conversationDetail.Messages[1])
	}
	if len(conversationDetail.Messages[1].Sources) == 0 || conversationDetail.Messages[1].Sources[0].DocName != "routes-e2e.html" {
		t.Fatalf("expected persisted grounded sources, got %#v", conversationDetail.Messages[1].Sources)
	}

	deleteConversationResponse := performAuthorizedRequest(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/chat/conversations/%d", conversation.ID), token, nil, "")
	if deleteConversationResponse.Code != http.StatusOK {
		t.Fatalf("expected delete conversation status 200, got %d body=%s", deleteConversationResponse.Code, deleteConversationResponse.Body.String())
	}

	getConversationResponse = performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/chat/conversations/%d", conversation.ID), token, nil, "")
	if getConversationResponse.Code != http.StatusNotFound {
		t.Fatalf("expected deleted conversation to return 404, got %d body=%s", getConversationResponse.Code, getConversationResponse.Body.String())
	}

	listConversationsResponse = performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/chat/conversations?collection_id=%d", *storedDocument.CollectionID), token, nil, "")
	if listConversationsResponse.Code != http.StatusOK {
		t.Fatalf("expected list conversations status 200 after delete, got %d body=%s", listConversationsResponse.Code, listConversationsResponse.Body.String())
	}
	if conversations = extractConversationsFromResponse(t, listConversationsResponse.Body.Bytes()); len(conversations) != 0 {
		t.Fatalf("expected no conversations after delete, got %#v", conversations)
	}
}

func TestRoutesE2EProviderRuntimeOverviewListsActiveAssignments(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token := registerRoutesE2EUser(t, router)

	indexingSaveResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/providers", map[string]any{
		"family":               "LLM",
		"provider_type":        "OPENAI",
		"credentials":          map[string]any{"api_key": "secret-openai", "model": "gpt-4o-mini"},
		"enabled_for_indexing": true,
	}, token)
	if indexingSaveResponse.Code != http.StatusOK {
		t.Fatalf("expected indexing provider save status 200, got %d body=%s", indexingSaveResponse.Code, indexingSaveResponse.Body.String())
	}

	chatSaveResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/providers", map[string]any{
		"family":           "LLM",
		"provider_type":    "OLLAMA",
		"credentials":      map[string]any{"model": "llama3.1:8b"},
		"enabled_for_chat": true,
	}, token)
	if chatSaveResponse.Code != http.StatusOK {
		t.Fatalf("expected chat provider save status 200, got %d body=%s", chatSaveResponse.Code, chatSaveResponse.Body.String())
	}

	runtimeResponse := performAuthorizedRequest(t, router, http.MethodGet, "/api/v1/user/providers/runtime", token, nil, "")
	if runtimeResponse.Code != http.StatusOK {
		t.Fatalf("expected runtime overview status 200, got %d body=%s", runtimeResponse.Code, runtimeResponse.Body.String())
	}

	overview := extractProviderRuntimeOverviewFromResponse(t, runtimeResponse.Body.Bytes())
	if overview.Indexing == nil || overview.Indexing.ProviderType != domain.ProviderOpenAI {
		t.Fatalf("expected active indexing provider %q, got %#v", domain.ProviderOpenAI, overview.Indexing)
	}
	if overview.Indexing.PublicMetadata["model"] != "gpt-4o-mini" {
		t.Fatalf("expected indexing model metadata, got %#v", overview.Indexing)
	}
	if overview.Chat == nil || overview.Chat.ProviderType != domain.ProviderOllama {
		t.Fatalf("expected active chat provider %q, got %#v", domain.ProviderOllama, overview.Chat)
	}
	if overview.Chat.PublicMetadata["model"] != "llama3.1:8b" {
		t.Fatalf("expected chat model metadata, got %#v", overview.Chat)
	}
}

func TestRoutesE2EProviderAuditEndpointReturnsSummaryAndCursor(t *testing.T) {
	db := openRoutesE2EDB(t)
	cfg := newRoutesE2EConfig(t)
	router := gin.New()

	if err := SetupRoutes(router, db, cfg); err != nil {
		t.Fatalf("SetupRoutes returned error: %v", err)
	}

	token := registerRoutesE2EUser(t, router)

	saveProviderResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/providers", map[string]any{
		"family":           "LLM",
		"provider_type":    "OPENAI",
		"credentials":      map[string]any{"api_key": "routes-e2e-secret", "model": "gpt-4o-mini"},
		"enabled_for_chat": false,
	}, token)
	if saveProviderResponse.Code != http.StatusOK {
		t.Fatalf("expected provider save status 200, got %d body=%s", saveProviderResponse.Code, saveProviderResponse.Body.String())
	}

	enableProviderResponse := performJSONRequest(t, router, http.MethodPatch, "/api/v1/user/providers/OPENAI", map[string]any{
		"activation_target": "CHAT",
		"enabled":           true,
	}, token)
	if enableProviderResponse.Code != http.StatusOK {
		t.Fatalf("expected provider enable status 200, got %d body=%s", enableProviderResponse.Code, enableProviderResponse.Body.String())
	}

	auditResponse := performAuthorizedRequest(t, router, http.MethodGet, "/api/v1/user/providers/audit?family=LLM&provider_type=OPENAI&limit=1", token, nil, "")
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected provider audit status 200, got %d body=%s", auditResponse.Code, auditResponse.Body.String())
	}

	result := extractProviderAuditQueryFromResponse(t, auditResponse.Body.Bytes())
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 provider audit item on first page, got %d", len(result.Items))
	}
	if result.Summary.TotalEvents != 2 {
		t.Fatalf("expected 2 provider audit events, got %#v", result.Summary)
	}
	if !result.PageInfo.HasNext || result.PageInfo.NextCursor == "" {
		t.Fatalf("expected provider audit cursor, got %#v", result.PageInfo)
	}

	nextPageResponse := performAuthorizedRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/user/providers/audit?family=LLM&provider_type=OPENAI&limit=1&cursor=%s", result.PageInfo.NextCursor), token, nil, "")
	if nextPageResponse.Code != http.StatusOK {
		t.Fatalf("expected second provider audit status 200, got %d body=%s", nextPageResponse.Code, nextPageResponse.Body.String())
	}

	nextPage := extractProviderAuditQueryFromResponse(t, nextPageResponse.Body.Bytes())
	if len(nextPage.Items) != 1 {
		t.Fatalf("expected 1 remaining provider audit item, got %d", len(nextPage.Items))
	}
}

type routesE2EDocument struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type routesE2EProviderConfig struct {
	ProviderType   string            `json:"provider_type"`
	EnabledForChat bool              `json:"enabled_for_chat"`
	PublicMetadata map[string]string `json:"public_metadata"`
}

type routesE2EProviderRuntimeAssignment struct {
	ActivationTarget string            `json:"activation_target"`
	ProviderType     string            `json:"provider_type"`
	PublicMetadata   map[string]string `json:"public_metadata"`
}

type routesE2EProviderRuntimeOverview struct {
	Storage  *routesE2EProviderRuntimeAssignment `json:"storage"`
	Indexing *routesE2EProviderRuntimeAssignment `json:"indexing"`
	Chat     *routesE2EProviderRuntimeAssignment `json:"chat"`
}

type routesE2EProviderReadinessCheck struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

type routesE2EProviderReadinessReport struct {
	Status           string                            `json:"status"`
	Checks           []routesE2EProviderReadinessCheck `json:"checks"`
	DiscoveredModels []string                          `json:"discovered_models"`
}

type routesE2EConversation struct {
	ID           uint   `json:"id"`
	CollectionID uint   `json:"collection_id"`
	Title        string `json:"title"`
}

type routesE2EConversationMessageSource struct {
	DocName string `json:"doc_name"`
	Page    int    `json:"page"`
	Snippet string `json:"snippet"`
}

type routesE2EConversationMessage struct {
	Role    string                               `json:"role"`
	Content string                               `json:"content"`
	Sources []routesE2EConversationMessageSource `json:"sources"`
}

type routesE2EConversationDetail struct {
	Conversation *routesE2EConversation         `json:"conversation"`
	Messages     []routesE2EConversationMessage `json:"messages"`
}

type routesE2EResponseEnvelope struct {
	Status  int             `json:"status"`
	Type    string          `json:"type"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type routesE2EAuditBucket struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type routesE2EAuditPageInfo struct {
	Limit      int    `json:"limit"`
	Returned   int    `json:"returned"`
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor"`
}

type routesE2EDocumentAuditEvent struct {
	ID         uint   `json:"id"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	DocumentID *uint  `json:"document_id"`
}

type routesE2EDocumentAuditQueryResponse struct {
	Items   []routesE2EDocumentAuditEvent `json:"items"`
	Summary struct {
		TotalEvents     int64                  `json:"total_events"`
		ActionBreakdown []routesE2EAuditBucket `json:"action_breakdown"`
		StatusBreakdown []routesE2EAuditBucket `json:"status_breakdown"`
	} `json:"summary"`
	PageInfo routesE2EAuditPageInfo `json:"page_info"`
}

type routesE2EProviderAuditEvent struct {
	ID           uint   `json:"id"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	ProviderType string `json:"provider_type"`
}

type routesE2EProviderAuditQueryResponse struct {
	Items   []routesE2EProviderAuditEvent `json:"items"`
	Summary struct {
		TotalEvents     int64                  `json:"total_events"`
		ActionBreakdown []routesE2EAuditBucket `json:"action_breakdown"`
		StatusBreakdown []routesE2EAuditBucket `json:"status_breakdown"`
	} `json:"summary"`
	PageInfo routesE2EAuditPageInfo `json:"page_info"`
}

func performJSONRequest(t *testing.T, router http.Handler, method, path string, payload any, token string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	return performAuthorizedRequest(t, router, method, path, token, bytes.NewReader(body), "application/json")
}

func performMultipartUploadRequest(t *testing.T, router http.Handler, path, token string, fields map[string]string, fileField, fileName, contentType string, fileBody []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s) returned error: %v", key, err)
		}
	}

	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     fileField,
		"filename": fileName,
	}))
	partHeader.Set("Content-Type", contentType)

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := part.Write(fileBody); err != nil {
		t.Fatalf("part.Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept-Language", "pt-BR")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func performAuthorizedRequest(t *testing.T, router http.Handler, method, path, token string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Accept-Language", "pt-BR")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func extractTokenFromResponse(t *testing.T, body []byte) string {
	t.Helper()

	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal token response returned error: %v", err)
	}

	return envelope.Data.Token
}

func extractDocumentFromResponse(t *testing.T, body []byte) routesE2EDocument {
	t.Helper()

	var envelope struct {
		Data routesE2EDocument `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal document response returned error: %v", err)
	}

	return envelope.Data
}

func extractResponseEnvelope(t *testing.T, body []byte) routesE2EResponseEnvelope {
	t.Helper()

	var envelope routesE2EResponseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal response envelope returned error: %v", err)
	}

	return envelope
}

func extractDocumentsFromResponse(t *testing.T, body []byte) []routesE2EDocument {
	t.Helper()

	var envelope struct {
		Data []routesE2EDocument `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal documents response returned error: %v", err)
	}

	return envelope.Data
}

func extractProviderConfigsFromResponse(t *testing.T, body []byte) []routesE2EProviderConfig {
	t.Helper()

	var envelope struct {
		Data []routesE2EProviderConfig `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal providers response returned error: %v", err)
	}

	return envelope.Data
}

func extractProviderRuntimeOverviewFromResponse(t *testing.T, body []byte) routesE2EProviderRuntimeOverview {
	t.Helper()

	var envelope struct {
		Data routesE2EProviderRuntimeOverview `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal provider runtime overview returned error: %v", err)
	}

	return envelope.Data
}

func extractProviderReadinessFromResponse(t *testing.T, body []byte) routesE2EProviderReadinessReport {
	t.Helper()

	var envelope struct {
		Data routesE2EProviderReadinessReport `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal provider readiness response returned error: %v", err)
	}

	return envelope.Data
}

func extractDocumentAuditQueryFromResponse(t *testing.T, body []byte) routesE2EDocumentAuditQueryResponse {
	t.Helper()

	var envelope struct {
		Data routesE2EDocumentAuditQueryResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal document audit query response returned error: %v", err)
	}

	return envelope.Data
}

func extractProviderAuditQueryFromResponse(t *testing.T, body []byte) routesE2EProviderAuditQueryResponse {
	t.Helper()

	var envelope struct {
		Data routesE2EProviderAuditQueryResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal provider audit query response returned error: %v", err)
	}

	return envelope.Data
}

func extractConversationFromResponse(t *testing.T, body []byte) routesE2EConversation {
	t.Helper()

	var envelope struct {
		Data routesE2EConversation `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal conversation response returned error: %v", err)
	}

	return envelope.Data
}

func extractConversationsFromResponse(t *testing.T, body []byte) []routesE2EConversation {
	t.Helper()

	var envelope struct {
		Data []routesE2EConversation `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal conversations response returned error: %v", err)
	}

	return envelope.Data
}

func extractConversationDetailFromResponse(t *testing.T, body []byte) routesE2EConversationDetail {
	t.Helper()

	var envelope struct {
		Data routesE2EConversationDetail `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Unmarshal conversation detail response returned error: %v", err)
	}

	return envelope.Data
}

func registerRoutesE2EUser(t *testing.T, router http.Handler) string {
	t.Helper()

	registerResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Routes E2E",
		"email":    fmt.Sprintf("routes-e2e-%d@example.com", time.Now().UnixNano()),
		"password": "secret123",
	}, "")
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d body=%s", registerResponse.Code, registerResponse.Body.String())
	}

	var registerEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(registerResponse.Body.Bytes(), &registerEnvelope); err != nil {
		t.Fatalf("Unmarshal register response returned error: %v", err)
	}

	return registerEnvelope.Data.Token
}

func registerAndUploadHTMLDocument(t *testing.T, router http.Handler) (string, routesE2EDocument) {
	t.Helper()

	token := registerRoutesE2EUser(t, router)

	uploadResponse := performMultipartUploadRequest(t, router, "/api/v1/documents/upload", token, map[string]string{
		"title":      "routes-e2e.html",
		"category":   "Policies",
		"collection": "Operations",
	}, "file", "routes-e2e.html", "text/html", []byte("<html><body>ThingIndex E2E</body></html>"))
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}

	return token, extractDocumentFromResponse(t, uploadResponse.Body.Bytes())
}

func mustLoadRoutesE2EDocument(t *testing.T, db *gorm.DB, documentID uint) models.Document {
	t.Helper()

	var document models.Document
	if err := db.First(&document, documentID).Error; err != nil {
		t.Fatalf("load document %d returned error: %v", documentID, err)
	}

	return document
}

func seedRoutesE2EChatContext(t *testing.T, db *gorm.DB, documentID uint) {
	t.Helper()

	pageNumber := 1
	if err := db.Create(&models.DocumentPage{
		DocumentID:    documentID,
		PageNumber:    pageNumber,
		Content:       "O atendimento emergencial deve ocorrer em ate 4 horas. O relatorio mensal deve ser entregue ate o dia 5 de cada mes.",
		Summary:       "Prazos operacionais e de atendimento do contrato.",
		TokenEstimate: 32,
	}).Error; err != nil {
		t.Fatalf("seed document page: %v", err)
	}

	if err := db.Create(&models.DocumentOutlineNode{
		DocumentID:    documentID,
		SequenceOrder: 0,
		NodeID:        "prazos-operacionais",
		Level:         1,
		Title:         "Prazos Operacionais",
		Summary:       "Atendimento emergencial em ate 4 horas e relatorio mensal ate o dia 5.",
		StartPage:     &pageNumber,
		EndPage:       &pageNumber,
		NodePath:      "prazos-operacionais",
		Metadata: map[string]any{
			"confidence_score": 95,
		},
	}).Error; err != nil {
		t.Fatalf("seed outline node: %v", err)
	}

	if err := db.Create(&models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: 0,
		Key:           "document_summary",
		Value:         "Contrato com manutencao predial, prazo de atendimento emergencial e entrega de relatorio mensal.",
	}).Error; err != nil {
		t.Fatalf("seed metadata pair: %v", err)
	}
}

func newRoutesE2EFakeOpenAICompatibleServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			var payload struct {
				Stream bool `json:"stream"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if payload.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				flusher, ok := w.(http.Flusher)
				if !ok {
					http.Error(w, "streaming unsupported", http.StatusInternalServerError)
					return
				}

				_, _ = fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"O relatorio mensal deve ser entregue ate o dia 5, e o atendimento emergencial deve ocorrer em ate 4 horas."}}]}`)
				flusher.Flush()
				_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]string{
							"content": "OK",
						},
					},
				},
			})
			return
		case "/models":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "routes-e2e-model"},
					{"id": "routes-e2e-alt-model"},
				},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))

	t.Cleanup(server.Close)
	return server
}

func openRoutesE2EDB(t *testing.T) *gorm.DB {
	t.Helper()

	loadAPIE2EEnvOnce.Do(func() {
		_ = godotenv.Load("../../../.env")
	})

	cfg := routesE2EDBConfigFromEnv()
	adminDB, err := gorm.Open(postgres.Open(routesE2EDSNS(cfg, "")), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open e2e admin database: %v", err)
	}

	schemaName := fmt.Sprintf("routes_e2e_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName)).Error; err != nil {
		t.Fatalf("create e2e schema: %v", err)
	}

	testDB, err := gorm.Open(postgres.Open(routesE2EDSNS(cfg, schemaName)), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open e2e schema database: %v", err)
	}

	if err := testDB.AutoMigrate(
		&models.User{},
		&models.DocumentCategory{},
		&models.Collection{},
		&models.Document{},
		&models.DocumentPage{},
		&models.ParagraphNode{},
		&models.TableSection{},
		&models.MetadataPair{},
		&models.DocumentChunk{},
		&models.DocumentOutlineNode{},
		&models.OutboxEvent{},
		&models.DocumentProcessingAttempt{},
		&models.DocumentAuditEvent{},
		&models.UserProviderConfig{},
		&models.ProviderModelCatalog{},
		&models.ProviderAuditEvent{},
		&models.PricingSource{},
		&models.ModelPrice{},
		&models.ModelPriceHistory{},
		&models.ChatConversation{},
		&models.ChatConversationEntry{},
	); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := testDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}

		if err := adminDB.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName)).Error; err != nil {
			t.Fatalf("drop e2e schema: %v", err)
		}

		adminSQLDB, err := adminDB.DB()
		if err == nil {
			_ = adminSQLDB.Close()
		}
	})

	return testDB
}

type routesE2EDBConfig struct {
	host     string
	port     string
	user     string
	password string
	dbName   string
	sslMode  string
}

func routesE2EDBConfigFromEnv() routesE2EDBConfig {
	return routesE2EDBConfig{
		host:     firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_DB_HOST"), os.Getenv("DB_HOST"), "127.0.0.1"),
		port:     firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_DB_PORT"), os.Getenv("DB_PORT"), "5434"),
		user:     firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_DB_USER"), os.Getenv("DB_USER"), os.Getenv("POSTGRES_USER")),
		password: firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_DB_PASSWORD"), os.Getenv("DB_PASSWORD"), os.Getenv("POSTGRES_PASSWORD")),
		dbName:   firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_DB_NAME"), os.Getenv("DB_NAME"), os.Getenv("POSTGRES_DB")),
		sslMode:  firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_DB_SSLMODE"), os.Getenv("DB_SSLMODE"), "disable"),
	}
}

func routesE2EDSNS(cfg routesE2EDBConfig, searchPath string) string {
	parts := []string{
		fmt.Sprintf("host=%s", cfg.host),
		fmt.Sprintf("port=%s", cfg.port),
		fmt.Sprintf("user=%s", cfg.user),
		fmt.Sprintf("password=%s", cfg.password),
		fmt.Sprintf("dbname=%s", cfg.dbName),
		fmt.Sprintf("sslmode=%s", cfg.sslMode),
	}
	if searchPath != "" {
		parts = append(parts, fmt.Sprintf("search_path=%s", searchPath))
	}

	return strings.Join(parts, " ")
}

func newRoutesE2EConfig(t *testing.T) *config.Config {
	t.Helper()

	loadAPIE2EEnvOnce.Do(func() {
		_ = godotenv.Load("../../../.env")
	})

	minioEndpoint := normalizeRoutesE2EMinioEndpoint(firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_ENDPOINT"), os.Getenv("MINIO_ENDPOINT")))
	if minioEndpoint == "" {
		minioEndpoint = "127.0.0.1:9000"
	}

	cfg := &config.Config{
		Environment:                  "test",
		AllowPrivateStorageEndpoints: true,
		CORSAllowedOrigins:           "http://localhost:3000",
		JWTSecret:                    "routes-e2e-secret",
		MinioEndpoint:                minioEndpoint,
		MinioAccessKey:               firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_ACCESS_KEY"), os.Getenv("MINIO_ACCESS_KEY"), "admin"),
		MinioSecretKey:               firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_SECRET_KEY"), os.Getenv("MINIO_SECRET_KEY"), "admin1234"),
		MinioUseSSL:                  firstRoutesE2ENonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_USE_SSL"), os.Getenv("MINIO_USE_SSL"), "false"),
		MinioBucketName:              fmt.Sprintf("thingindex-routes-e2e-%d", time.Now().UnixNano()),
		MaxUploadSizeBytes:           1 << 20,
		MasterEncryptionKey:          "0123456789abcdef0123456789abcdef",
		AuthRateLimitRequests:        10,
		AuthRateLimitWindow:          time.Minute,
		SMTPHost:                     "localhost",
		SMTPPort:                     "1025",
		SMTPSecurityMode:             "none",
		SMTPFrom:                     "no-reply@example.com",
		ResetPasswordURL:             "http://localhost:3000/reset-password",
		RabbitMQURL:                  "amqp://guest:guest@127.0.0.1:5672/",
		DocumentIndexExchange:        "routes-e2e-unused",
		DocumentIndexQueue:           "routes-e2e-unused",
		DocumentIndexRoutingKey:      "routes-e2e-unused",
		DocumentIndexRetryQueue:      "routes-e2e-unused",
		DocumentIndexDLQ:             "routes-e2e-unused",
		DocumentIndexRetryDelay:      time.Second,
		DocumentIndexMaxRetries:      1,
		OutboxPollInterval:           time.Second,
		OutboxBatchSize:              1,
		ConsumerPrefetchCount:        1,
		PricingSyncInterval:          time.Hour,
		ModelCatalogSyncInterval:     0,
		ServerReadTimeout:            time.Second,
		ServerWriteTimeout:           time.Second,
		ServerIdleTimeout:            time.Second,
	}

	t.Cleanup(func() {
		minioClient, err := minio.New(cfg.MinioEndpoint, &minio.Options{
			Creds:        credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
			Secure:       cfg.MinioUseSSL == "true",
			BucketLookup: minio.BucketLookupPath,
		})
		if err != nil {
			t.Fatalf("cleanup minio.New returned error: %v", err)
		}

		cleanupCtx := context.Background()
		for object := range minioClient.ListObjects(cleanupCtx, cfg.MinioBucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				t.Fatalf("cleanup ListObjects returned error: %v", object.Err)
			}
			if err := minioClient.RemoveObject(cleanupCtx, cfg.MinioBucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Fatalf("cleanup RemoveObject returned error: %v", err)
			}
		}
		if err := minioClient.RemoveBucket(cleanupCtx, cfg.MinioBucketName); err != nil {
			t.Fatalf("cleanup RemoveBucket returned error: %v", err)
		}
	})

	return cfg
}

func newRoutesE2EMinioClient(t *testing.T, cfg *config.Config) *minio.Client {
	t.Helper()

	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure:       cfg.MinioUseSSL == "true",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		t.Fatalf("minio.New returned error: %v", err)
	}

	return client
}

func normalizeRoutesE2EMinioEndpoint(endpoint string) string {
	switch strings.TrimSpace(endpoint) {
	case "", "minio", "minio:9000":
		return "127.0.0.1:9000"
	default:
		if strings.HasPrefix(strings.TrimSpace(endpoint), "minio:") {
			return "127.0.0.1" + strings.TrimPrefix(strings.TrimSpace(endpoint), "minio")
		}
		return strings.TrimSpace(endpoint)
	}
}

func firstRoutesE2ENonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
