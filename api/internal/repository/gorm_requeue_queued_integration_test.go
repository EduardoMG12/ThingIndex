//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

// Mensagem perdida depois do broker: o evento esta PUBLISHED, o documento
// segue QUEUED e nada volta a olhar para ele. O reconciliador cria um evento
// PENDING novo so para esse caso; documentos cuja entrega ainda esta sob
// responsabilidade do despachante (ultimo evento PENDING/IN_FLIGHT/FAILED) ou
// que sairam de QUEUED nao podem ganhar evento duplicado.
//
// Contra Postgres real porque o que precisa estar certo e a condicao SQL do
// "ultimo evento do documento".
func TestRequeueStuckQueuedDocumentsReenfileiraApenasMensagensPerdidas(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormDocumentProcessingRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "reconcilia")
	category := seedIntegrationCategory(t, db, user.ID, "Auditoria")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Processos")

	agora := time.Now().UTC()
	antigo := agora.Add(-2 * time.Hour)
	recente := agora.Add(-time.Minute)

	perdido := seedQueuedDocument(t, db, user.ID, category.ID, collection.ID, "perdido.pdf", domain.DocumentStatusQueued, antigo)
	seedOutboxEvent(t, db, perdido.ID, domain.OutboxEventStatusPublished, &antigo)

	// Publicado ha pouco: o worker pode so estar atrasado.
	publicadoRecente := seedQueuedDocument(t, db, user.ID, category.ID, collection.ID, "recente.pdf", domain.DocumentStatusQueued, antigo)
	seedOutboxEvent(t, db, publicadoRecente.ID, domain.OutboxEventStatusPublished, &recente)

	// Evento antigo PUBLISHED mas o mais recente ainda PENDING (reindexacao ja
	// pedida): o despachante cuida dele.
	comPendente := seedQueuedDocument(t, db, user.ID, category.ID, collection.ID, "pendente.pdf", domain.DocumentStatusQueued, antigo)
	seedOutboxEvent(t, db, comPendente.ID, domain.OutboxEventStatusPublished, &antigo)
	seedOutboxEvent(t, db, comPendente.ID, domain.OutboxEventStatusPending, nil)

	// Ja indexado: nao e QUEUED.
	indexado := seedQueuedDocument(t, db, user.ID, category.ID, collection.ID, "indexado.pdf", domain.DocumentStatusIndexed, antigo)
	seedOutboxEvent(t, db, indexado.ID, domain.OutboxEventStatusPublished, &antigo)

	// QUEUED com documento atualizado ha pouco (retry agendado pelo consumidor).
	retryRecente := seedQueuedDocument(t, db, user.ID, category.ID, collection.ID, "retry.pdf", domain.DocumentStatusQueued, recente)
	seedOutboxEvent(t, db, retryRecente.ID, domain.OutboxEventStatusPublished, &antigo)

	cutoff := agora.Add(-30 * time.Minute)
	reenfileirados, err := repo.RequeueStuckQueuedDocuments(ctx, cutoff)
	if err != nil {
		t.Fatalf("RequeueStuckQueuedDocuments retornou erro: %v", err)
	}
	if reenfileirados != 1 {
		t.Fatalf("esperava 1 documento reenfileirado, veio %d", reenfileirados)
	}

	assertOutboxEventCount(t, db, perdido.ID, domain.OutboxEventStatusPending, 1)
	assertOutboxEventCount(t, db, publicadoRecente.ID, domain.OutboxEventStatusPending, 0)
	assertOutboxEventCount(t, db, comPendente.ID, domain.OutboxEventStatusPending, 1)
	assertOutboxEventCount(t, db, indexado.ID, domain.OutboxEventStatusPending, 0)
	assertOutboxEventCount(t, db, retryRecente.ID, domain.OutboxEventStatusPending, 0)

	var novo models.OutboxEvent
	if err := db.Where("aggregate_id = ? AND status = ?", perdido.ID, domain.OutboxEventStatusPending).First(&novo).Error; err != nil {
		t.Fatalf("carregar evento novo: %v", err)
	}
	if novo.EventType != domain.OutboxEventDocumentIndexRequested || novo.AggregateType != domain.OutboxAggregateDocument {
		t.Fatalf("evento novo com tipo errado: %#v", novo)
	}
	if got, _ := novo.Payload["document_id"].(float64); uint(got) != perdido.ID {
		t.Fatalf("payload sem document_id correto: %#v", novo.Payload)
	}
	if got, _ := novo.Payload["user_id"].(float64); uint(got) != user.ID {
		t.Fatalf("payload sem user_id correto: %#v", novo.Payload)
	}
	if _, ok := novo.Payload["requested_at"]; !ok {
		t.Fatalf("payload sem requested_at: o consumidor trataria como evento legado")
	}

	// Segundo ciclo logo em seguida: o ultimo evento agora e PENDING, entao
	// nada pode ser duplicado.
	deNovo, err := repo.RequeueStuckQueuedDocuments(ctx, cutoff)
	if err != nil {
		t.Fatalf("segundo ciclo retornou erro: %v", err)
	}
	if deNovo != 0 {
		t.Fatalf("segundo ciclo duplicou %d evento(s)", deNovo)
	}
}

func seedQueuedDocument(t *testing.T, db *gorm.DB, userID, categoryID, collectionID uint, title, status string, updatedAt time.Time) models.Document {
	t.Helper()

	document := models.Document{
		UserID:          userID,
		Title:           title,
		FileURL:         "documents/" + title,
		Size:            2048,
		MimeType:        "application/pdf",
		Status:          status,
		ProcessingStage: "QUEUED",
		CollectionID:    &collectionID,
		CategoryID:      &categoryID,
	}
	mustCreateRecord(t, db, &document)
	if err := db.Model(&models.Document{}).Where("id = ?", document.ID).UpdateColumn("updated_at", updatedAt).Error; err != nil {
		t.Fatalf("ajustar updated_at: %v", err)
	}
	return document
}

func seedOutboxEvent(t *testing.T, db *gorm.DB, documentID uint, status string, publishedAt *time.Time) models.OutboxEvent {
	t.Helper()

	event := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   documentID,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        status,
		Payload:       map[string]any{"document_id": documentID},
		AvailableAt:   time.Now().UTC(),
		PublishedAt:   publishedAt,
	}
	mustCreateRecord(t, db, &event)
	return event
}

func assertOutboxEventCount(t *testing.T, db *gorm.DB, documentID uint, status string, want int64) {
	t.Helper()

	var got int64
	if err := db.Model(&models.OutboxEvent{}).
		Where("aggregate_type = ? AND aggregate_id = ? AND status = ?", domain.OutboxAggregateDocument, documentID, status).
		Count(&got).Error; err != nil {
		t.Fatalf("contar eventos: %v", err)
	}
	if got != want {
		t.Errorf("documento %d: %d evento(s) %s, esperava %d", documentID, got, status, want)
	}
}
