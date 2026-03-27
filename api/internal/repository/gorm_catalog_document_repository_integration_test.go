//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

func TestCatalogRepositoryCreateDocumentWithOutboxPersistsDerivedRecords(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCatalogDocumentRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "create-outbox")
	category := seedIntegrationCategory(t, db, user.ID, "Policies")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "HR")

	document := &models.Document{
		UserID:       user.ID,
		Title:        "handbook.pdf",
		FileURL:      "documents/handbook.pdf",
		Size:         1024,
		MimeType:     "application/pdf",
		Status:       domain.DocumentStatusQueued,
		CollectionID: &collection.ID,
		CategoryID:   &category.ID,
	}
	event := &models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload: map[string]any{
			"origin": "integration",
		},
		AvailableAt: time.Now().UTC(),
	}
	auditEvent := &models.DocumentAuditEvent{
		UserID:     user.ID,
		Action:     "UPLOAD",
		Status:     "SUCCESS",
		ActorType:  "USER",
		Origin:     "integration-test",
		Detail:     "created by integration test",
		Metadata:   map[string]any{"scope": "catalog"},
		DocumentID: nil,
	}

	if err := repo.CreateDocumentWithOutbox(ctx, document, event, auditEvent); err != nil {
		t.Fatalf("CreateDocumentWithOutbox returned error: %v", err)
	}

	if document.ID == 0 {
		t.Fatalf("expected persisted document ID")
	}
	if event.AggregateID != document.ID {
		t.Fatalf("expected outbox aggregate ID %d, got %d", document.ID, event.AggregateID)
	}
	if auditEvent.DocumentID == nil || *auditEvent.DocumentID != document.ID {
		t.Fatalf("expected audit event document ID %d, got %#v", document.ID, auditEvent.DocumentID)
	}

	var storedDocument models.Document
	if err := db.First(&storedDocument, document.ID).Error; err != nil {
		t.Fatalf("load stored document: %v", err)
	}

	var storedEvent models.OutboxEvent
	if err := db.First(&storedEvent).Error; err != nil {
		t.Fatalf("load stored outbox event: %v", err)
	}
	if storedEvent.AggregateID != document.ID {
		t.Fatalf("expected stored aggregate ID %d, got %d", document.ID, storedEvent.AggregateID)
	}
	if payloadUint(storedEvent.Payload, "document_id") != document.ID {
		t.Fatalf("expected payload document ID %d, got %#v", document.ID, storedEvent.Payload["document_id"])
	}

	var storedAudit models.DocumentAuditEvent
	if err := db.First(&storedAudit).Error; err != nil {
		t.Fatalf("load stored audit event: %v", err)
	}
	if storedAudit.DocumentID == nil || *storedAudit.DocumentID != document.ID {
		t.Fatalf("expected stored audit document ID %d, got %#v", document.ID, storedAudit.DocumentID)
	}
	if storedAudit.CollectionID == nil || *storedAudit.CollectionID != collection.ID {
		t.Fatalf("expected stored audit collection ID %d, got %#v", collection.ID, storedAudit.CollectionID)
	}
	if storedAudit.Title != document.Title {
		t.Fatalf("expected stored audit title %q, got %q", document.Title, storedAudit.Title)
	}
}

func TestCatalogRepositoryRetryDocumentIndexingResetsStateAndEnqueuesEvent(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCatalogDocumentRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "retry")
	category := seedIntegrationCategory(t, db, user.ID, "Finance")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Invoices")
	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	indexedAt := time.Now().UTC().Add(-time.Hour)
	document := &models.Document{
		UserID:              user.ID,
		Title:               "invoice.pdf",
		FileURL:             "documents/invoice.pdf",
		Size:                2048,
		MimeType:            "application/pdf",
		Status:              domain.DocumentStatusFailed,
		ProcessingStage:     "FAILED",
		StatusDetail:        "timeout while indexing",
		LastProcessingError: "timeout while indexing",
		IndexStartedAt:      &startedAt,
		IndexedAt:           &indexedAt,
		CollectionID:        &collection.ID,
		CategoryID:          &category.ID,
	}
	mustCreateRecord(t, db, document)

	auditEvent := &models.DocumentAuditEvent{
		UserID:    user.ID,
		Action:    "RETRY",
		Status:    "SUCCESS",
		ActorType: "USER",
		Origin:    "integration-test",
	}

	retriedDocument, err := repo.RetryDocumentIndexing(ctx, user.ID, document.ID, auditEvent)
	if err != nil {
		t.Fatalf("RetryDocumentIndexing returned error: %v", err)
	}

	if retriedDocument.Status != domain.DocumentStatusQueued {
		t.Fatalf("expected status %q, got %q", domain.DocumentStatusQueued, retriedDocument.Status)
	}
	if retriedDocument.ProcessingStage != "QUEUED" {
		t.Fatalf("expected processing stage QUEUED, got %q", retriedDocument.ProcessingStage)
	}
	if retriedDocument.LastProcessingError != "" {
		t.Fatalf("expected cleared processing error, got %q", retriedDocument.LastProcessingError)
	}
	if retriedDocument.IndexStartedAt != nil || retriedDocument.IndexedAt != nil {
		t.Fatalf("expected indexing timestamps to be cleared, got started=%v indexed=%v", retriedDocument.IndexStartedAt, retriedDocument.IndexedAt)
	}
	if retriedDocument.CollectionRef == nil || retriedDocument.CollectionRef.ID != collection.ID {
		t.Fatalf("expected preloaded collection %d, got %#v", collection.ID, retriedDocument.CollectionRef)
	}
	if retriedDocument.CollectionRef.Category == nil || retriedDocument.CollectionRef.Category.ID != category.ID {
		t.Fatalf("expected preloaded category %d, got %#v", category.ID, retriedDocument.CollectionRef.Category)
	}

	var storedDocument models.Document
	if err := db.First(&storedDocument, document.ID).Error; err != nil {
		t.Fatalf("load retried document: %v", err)
	}
	if storedDocument.Status != domain.DocumentStatusQueued {
		t.Fatalf("expected stored status %q, got %q", domain.DocumentStatusQueued, storedDocument.Status)
	}
	if storedDocument.IndexStartedAt != nil || storedDocument.IndexedAt != nil {
		t.Fatalf("expected stored timestamps to be cleared, got started=%v indexed=%v", storedDocument.IndexStartedAt, storedDocument.IndexedAt)
	}

	var events []models.OutboxEvent
	if err := db.Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("load outbox events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}
	if events[0].EventType != domain.OutboxEventDocumentIndexRequested {
		t.Fatalf("expected event type %q, got %q", domain.OutboxEventDocumentIndexRequested, events[0].EventType)
	}
	if payloadUint(events[0].Payload, "document_id") != document.ID {
		t.Fatalf("expected payload document ID %d, got %#v", document.ID, events[0].Payload["document_id"])
	}
	if payloadUint(events[0].Payload, "user_id") != user.ID {
		t.Fatalf("expected payload user ID %d, got %#v", user.ID, events[0].Payload["user_id"])
	}

	var audits []models.DocumentAuditEvent
	if err := db.Order("id ASC").Find(&audits).Error; err != nil {
		t.Fatalf("load audit events: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audits))
	}
	if audits[0].DocumentID == nil || *audits[0].DocumentID != document.ID {
		t.Fatalf("expected audit document ID %d, got %#v", document.ID, audits[0].DocumentID)
	}
}

func TestCatalogRepositoryDeleteDocumentsByCollectionRemovesDerivedData(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCatalogDocumentRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "delete-collection")
	category := seedIntegrationCategory(t, db, user.ID, "Knowledge Base")
	targetCollection := seedIntegrationCollection(t, db, user.ID, category.ID, "Policies")
	otherCollection := seedIntegrationCollection(t, db, user.ID, category.ID, "Guides")

	firstDocument := seedIntegrationDocument(t, db, user.ID, category.ID, targetCollection.ID, "policy-a.pdf")
	secondDocument := seedIntegrationDocument(t, db, user.ID, category.ID, targetCollection.ID, "policy-b.pdf")
	untouchedDocument := seedIntegrationDocument(t, db, user.ID, category.ID, otherCollection.ID, "guide.pdf")

	seedIntegrationDerivedData(t, db, firstDocument.ID)
	seedIntegrationDerivedData(t, db, secondDocument.ID)
	seedIntegrationDerivedData(t, db, untouchedDocument.ID)
	seedIntegrationOutboxEvent(t, db, firstDocument.ID, user.ID)
	seedIntegrationOutboxEvent(t, db, secondDocument.ID, user.ID)
	seedIntegrationOutboxEvent(t, db, untouchedDocument.ID, user.ID)

	deletedCount, err := repo.DeleteDocumentsByCollection(ctx, user.ID, targetCollection.ID, []models.DocumentAuditEvent{
		{
			UserID:       user.ID,
			DocumentID:   &firstDocument.ID,
			CollectionID: &targetCollection.ID,
			Action:       "DELETE",
			Status:       "SUCCESS",
			ActorType:    "USER",
			Origin:       "integration-test",
			Title:        firstDocument.Title,
		},
		{
			UserID:       user.ID,
			DocumentID:   &secondDocument.ID,
			CollectionID: &targetCollection.ID,
			Action:       "DELETE",
			Status:       "SUCCESS",
			ActorType:    "USER",
			Origin:       "integration-test",
			Title:        secondDocument.Title,
		},
	})
	if err != nil {
		t.Fatalf("DeleteDocumentsByCollection returned error: %v", err)
	}
	if deletedCount != 2 {
		t.Fatalf("expected deleted count 2, got %d", deletedCount)
	}

	assertDocumentMissing(t, db, firstDocument.ID)
	assertDocumentMissing(t, db, secondDocument.ID)
	assertDocumentPresent(t, db, untouchedDocument.ID)
	assertDerivedDataMissing(t, db, firstDocument.ID)
	assertDerivedDataMissing(t, db, secondDocument.ID)
	assertDerivedDataPresent(t, db, untouchedDocument.ID)

	var outboxCount int64
	if err := db.Model(&models.OutboxEvent{}).
		Where("aggregate_type = ? AND aggregate_id IN ?", domain.OutboxAggregateDocument, []uint{firstDocument.ID, secondDocument.ID}).
		Count(&outboxCount).Error; err != nil {
		t.Fatalf("count target outbox events: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected target outbox events to be deleted, got %d", outboxCount)
	}

	var untouchedOutbox models.OutboxEvent
	if err := db.Where("aggregate_id = ?", untouchedDocument.ID).First(&untouchedOutbox).Error; err != nil {
		t.Fatalf("expected untouched outbox event to remain: %v", err)
	}

	var auditCount int64
	if err := db.Model(&models.DocumentAuditEvent{}).Count(&auditCount).Error; err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("expected 2 audit events, got %d", auditCount)
	}
}

func TestCatalogRepositoryListDocumentAuditEventsAppliesFilters(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCatalogDocumentRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "list-document-audit")
	category := seedIntegrationCategory(t, db, user.ID, "Ops")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Policies")
	document := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, "policy.pdf")
	otherDocument := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, "archive.pdf")

	mustCreateRecord(t, db, &models.DocumentAuditEvent{
		UserID:       user.ID,
		DocumentID:   &document.ID,
		CollectionID: &collection.ID,
		Action:       "DOCUMENT_UPLOAD",
		Status:       "SUCCESS",
		ActorType:    "USER",
		Title:        document.Title,
		Detail:       "uploaded",
	})
	mustCreateRecord(t, db, &models.DocumentAuditEvent{
		UserID:       user.ID,
		DocumentID:   &otherDocument.ID,
		CollectionID: &collection.ID,
		Action:       "DOCUMENT_DELETE",
		Status:       "FAILED",
		ActorType:    "USER",
		Title:        otherDocument.Title,
		Detail:       "delete failed",
	})

	events, err := repo.ListDocumentAuditEvents(ctx, user.ID, domain.DocumentAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:  10,
			Status: "success",
		},
		Action:     "document_upload",
		DocumentID: &document.ID,
	})
	if err != nil {
		t.Fatalf("ListDocumentAuditEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(events))
	}
	if events[0].Title != document.Title || events[0].Action != "DOCUMENT_UPLOAD" {
		t.Fatalf("unexpected filtered event: %#v", events[0])
	}
}

func TestCatalogRepositoryQueryDocumentAuditEventsReturnsSummaryAndCursor(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCatalogDocumentRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "query-document-audit")
	category := seedIntegrationCategory(t, db, user.ID, "Ops")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Policies")
	document := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, "policy.pdf")

	now := time.Now().UTC()
	actorID := user.ID
	for index, item := range []struct {
		action string
		status string
		title  string
		detail string
	}{
		{action: "DOCUMENT_UPLOAD", status: "SUCCESS", title: "policy.pdf", detail: "uploaded successfully"},
		{action: "DOCUMENT_RETRY_INDEX", status: "FAILED", title: "policy.pdf", detail: "retry failed"},
		{action: "DOCUMENT_DELETE", status: "SUCCESS", title: "policy.pdf", detail: "deleted"},
	} {
		createdAt := now.Add(time.Duration(index) * time.Minute)
		mustCreateRecord(t, db, &models.DocumentAuditEvent{
			UserID:        user.ID,
			DocumentID:    &document.ID,
			CollectionID:  &collection.ID,
			ActorType:     "USER",
			ActorID:       &actorID,
			Action:        item.action,
			Status:        item.status,
			CorrelationID: "req-document-query-77",
			Origin:        "POST /api/v1/documents/upload",
			Title:         item.title,
			Detail:        item.detail,
			CreatedAt:     createdAt,
			UpdatedAt:     createdAt,
		})
	}

	result, err := repo.QueryDocumentAuditEvents(ctx, user.ID, domain.DocumentAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:         2,
			Query:         "policy",
			ActorType:     "user",
			ActorID:       &actorID,
			CorrelationID: "req-document-query-77",
		},
		DocumentID: &document.ID,
	})
	if err != nil {
		t.Fatalf("QueryDocumentAuditEvents returned error: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(result.Items))
	}
	if !result.PageInfo.HasNext || result.PageInfo.NextCursor == "" {
		t.Fatalf("expected next cursor on first page, got %#v", result.PageInfo)
	}
	if result.Summary.TotalEvents != 3 {
		t.Fatalf("expected total events 3, got %d", result.Summary.TotalEvents)
	}
	if result.Summary.UniqueDocuments != 1 || result.Summary.UniqueCollections != 1 {
		t.Fatalf("unexpected document summary: %#v", result.Summary)
	}
	if len(result.Summary.ActionBreakdown) == 0 || len(result.Summary.StatusBreakdown) == 0 {
		t.Fatalf("expected action and status breakdowns, got %#v", result.Summary)
	}

	nextPage, err := repo.QueryDocumentAuditEvents(ctx, user.ID, domain.DocumentAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:         2,
			Cursor:        result.PageInfo.NextCursor,
			Query:         "policy",
			ActorType:     "user",
			ActorID:       &actorID,
			CorrelationID: "req-document-query-77",
		},
		DocumentID: &document.ID,
	})
	if err != nil {
		t.Fatalf("QueryDocumentAuditEvents second page returned error: %v", err)
	}
	if len(nextPage.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(nextPage.Items))
	}
	if nextPage.PageInfo.HasNext {
		t.Fatalf("expected no further pages, got %#v", nextPage.PageInfo)
	}
}

func seedIntegrationUser(t *testing.T, db *gorm.DB, suffix string) models.User {
	t.Helper()

	user := models.User{
		Name:     "Integration " + suffix,
		Email:    "integration-" + suffix + "@example.com",
		Password: "hashed-password",
	}
	mustCreateRecord(t, db, &user)
	return user
}

func seedIntegrationCategory(t *testing.T, db *gorm.DB, userID uint, name string) models.DocumentCategory {
	t.Helper()

	category := models.DocumentCategory{
		UserID:         userID,
		Name:           name,
		NormalizedName: sanitizeIdentifier(name),
		Description:    "integration category",
	}
	mustCreateRecord(t, db, &category)
	return category
}

func seedIntegrationCollection(t *testing.T, db *gorm.DB, userID uint, categoryID uint, name string) models.Collection {
	t.Helper()

	collection := models.Collection{
		UserID:         userID,
		CategoryID:     &categoryID,
		Name:           name,
		NormalizedName: sanitizeIdentifier(name),
		Metadata:       map[string]any{"source": "integration"},
	}
	mustCreateRecord(t, db, &collection)
	return collection
}

func seedIntegrationDocument(t *testing.T, db *gorm.DB, userID uint, categoryID uint, collectionID uint, title string) models.Document {
	t.Helper()

	document := models.Document{
		UserID:       userID,
		Title:        title,
		FileURL:      "documents/" + title,
		Size:         2048,
		MimeType:     "application/pdf",
		Status:       domain.DocumentStatusIndexed,
		CollectionID: &collectionID,
		CategoryID:   &categoryID,
	}
	mustCreateRecord(t, db, &document)
	return document
}

func seedIntegrationDerivedData(t *testing.T, db *gorm.DB, documentID uint) {
	t.Helper()

	pageNumber := 1
	paragraphStart := 0
	paragraphEnd := 1
	mustCreateRecord(t, db, &models.DocumentPage{
		DocumentID:    documentID,
		PageNumber:    1,
		Content:       "page content",
		TokenEstimate: 10,
	})
	mustCreateRecord(t, db, &models.ParagraphNode{
		DocumentID:    documentID,
		SequenceOrder: 1,
		PageNumber:    &pageNumber,
		Content:       "paragraph content",
	})
	mustCreateRecord(t, db, &models.TableSection{
		DocumentID:    documentID,
		SequenceOrder: 1,
		Title:         "table",
		Headers:       []string{"name"},
		Rows:          [][]string{{"value"}},
	})
	mustCreateRecord(t, db, &models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: 1,
		Key:           "department",
		Value:         "finance",
	})
	mustCreateRecord(t, db, &models.DocumentChunk{
		DocumentID:           documentID,
		SequenceOrder:        1,
		ChunkType:            "paragraph",
		Title:                "chunk",
		Content:              "chunk content",
		SourceParagraphStart: &paragraphStart,
		SourceParagraphEnd:   &paragraphEnd,
		TokenEstimate:        10,
	})
	mustCreateRecord(t, db, &models.DocumentOutlineNode{
		DocumentID:    documentID,
		SequenceOrder: 1,
		NodeID:        "root_" + titleSuffix(documentID),
		Level:         1,
		Title:         "Root",
		NodePath:      "root_" + titleSuffix(documentID),
	})
}

func seedIntegrationOutboxEvent(t *testing.T, db *gorm.DB, documentID uint, userID uint) {
	t.Helper()

	mustCreateRecord(t, db, &models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   documentID,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload: map[string]any{
			"document_id": documentID,
			"user_id":     userID,
		},
		AvailableAt: time.Now().UTC(),
	})
}

func assertDocumentMissing(t *testing.T, db *gorm.DB, documentID uint) {
	t.Helper()

	var document models.Document
	err := db.First(&document, documentID).Error
	if err == nil {
		t.Fatalf("expected document %d to be deleted", documentID)
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("load deleted document %d: %v", documentID, err)
	}
}

func assertDocumentPresent(t *testing.T, db *gorm.DB, documentID uint) {
	t.Helper()

	var document models.Document
	if err := db.First(&document, documentID).Error; err != nil {
		t.Fatalf("expected document %d to remain: %v", documentID, err)
	}
}

func assertDerivedDataMissing(t *testing.T, db *gorm.DB, documentID uint) {
	t.Helper()

	assertModelCount(t, db, &models.DocumentPage{}, documentID, 0)
	assertModelCount(t, db, &models.ParagraphNode{}, documentID, 0)
	assertModelCount(t, db, &models.TableSection{}, documentID, 0)
	assertModelCount(t, db, &models.MetadataPair{}, documentID, 0)
	assertModelCount(t, db, &models.DocumentChunk{}, documentID, 0)
	assertModelCount(t, db, &models.DocumentOutlineNode{}, documentID, 0)
}

func assertDerivedDataPresent(t *testing.T, db *gorm.DB, documentID uint) {
	t.Helper()

	assertModelCount(t, db, &models.DocumentPage{}, documentID, 1)
	assertModelCount(t, db, &models.ParagraphNode{}, documentID, 1)
	assertModelCount(t, db, &models.TableSection{}, documentID, 1)
	assertModelCount(t, db, &models.MetadataPair{}, documentID, 1)
	assertModelCount(t, db, &models.DocumentChunk{}, documentID, 1)
	assertModelCount(t, db, &models.DocumentOutlineNode{}, documentID, 1)
}

func assertModelCount(t *testing.T, db *gorm.DB, model any, documentID uint, want int64) {
	t.Helper()

	var count int64
	if err := db.Model(model).Where("document_id = ?", documentID).Count(&count).Error; err != nil {
		t.Fatalf("count %T for document %d: %v", model, documentID, err)
	}
	if count != want {
		t.Fatalf("expected %d rows of %T for document %d, got %d", want, model, documentID, count)
	}
}

func mustCreateRecord(t *testing.T, db *gorm.DB, model any) {
	t.Helper()

	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create %T: %v", model, err)
	}
}

func payloadUint(payload map[string]any, key string) uint {
	if payload == nil {
		return 0
	}

	switch value := payload[key].(type) {
	case uint32:
		return uint(value)
	case uint:
		return value
	case uint64:
		return uint(value)
	case int:
		return uint(value)
	case int32:
		return uint(value)
	case int64:
		return uint(value)
	case float64:
		return uint(value)
	default:
		return 0
	}
}

func titleSuffix(id uint) string {
	return fmt.Sprintf("%d", id)
}
