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

// Postgres real porque o que importa e a condicao SQL, em especial
// index_started_at IS NOT NULL.
func TestReapStuckProcessingDocumentsLiberaApenasOsPresos(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormDocumentProcessingRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "reaper")
	category := seedIntegrationCategory(t, db, user.ID, "Auditoria")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Processos")

	agora := time.Now().UTC()
	antigo := agora.Add(-2 * time.Hour)
	recente := agora.Add(-1 * time.Minute)

	preso := seedProcessingDocument(t, db, user.ID, category.ID, collection.ID, "preso.pdf", domain.DocumentStatusProcessing, &antigo)

	// Comecou ha 1 minuto: nao pode ser tocado.
	emAndamento := seedProcessingDocument(t, db, user.ID, category.ID, collection.ID, "em-andamento.pdf", domain.DocumentStatusProcessing, &recente)

	// Sem index_started_at: ignorado.
	semCarimbo := seedProcessingDocument(t, db, user.ID, category.ID, collection.ID, "sem-carimbo.pdf", domain.DocumentStatusProcessing, nil)

	indexado := seedProcessingDocument(t, db, user.ID, category.ID, collection.ID, "indexado.pdf", domain.DocumentStatusIndexed, &antigo)

	cutoff := agora.Add(-30 * time.Minute)
	liberados, err := repo.ReapStuckProcessingDocuments(ctx, cutoff)
	if err != nil {
		t.Fatalf("ReapStuckProcessingDocuments retornou erro: %v", err)
	}

	if liberados != 1 {
		t.Fatalf("esperava exatamente 1 documento liberado, veio %d", liberados)
	}

	assertDocumentStatus(t, db, preso.ID, domain.DocumentStatusFailed, "FAILED_STUCK_IN_PROCESSING")
	assertDocumentStatus(t, db, emAndamento.ID, domain.DocumentStatusProcessing, "")
	assertDocumentStatus(t, db, semCarimbo.ID, domain.DocumentStatusProcessing, "")
	assertDocumentStatus(t, db, indexado.ID, domain.DocumentStatusIndexed, "")

	var liberado models.Document
	if err := db.First(&liberado, preso.ID).Error; err != nil {
		t.Fatalf("recarregar documento liberado: %v", err)
	}
	if liberado.StatusDetail == "" {
		t.Error("o documento liberado ficou sem status_detail explicando o que aconteceu")
	}
	if liberado.LastProcessingError == "" {
		t.Error("o documento liberado ficou sem last_processing_error registrando a causa")
	}
}

func TestReapStuckProcessingDocumentsNaoFazNadaQuandoNaoHaPresos(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormDocumentProcessingRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "reaper-vazio")
	category := seedIntegrationCategory(t, db, user.ID, "Auditoria")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Processos")

	recente := time.Now().UTC().Add(-time.Minute)
	seedProcessingDocument(t, db, user.ID, category.ID, collection.ID, "recente.pdf", domain.DocumentStatusProcessing, &recente)

	liberados, err := repo.ReapStuckProcessingDocuments(ctx, time.Now().UTC().Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("ReapStuckProcessingDocuments retornou erro: %v", err)
	}
	if liberados != 0 {
		t.Fatalf("esperava 0 documentos liberados, veio %d", liberados)
	}
}

func seedProcessingDocument(
	t *testing.T,
	db *gorm.DB,
	userID uint,
	categoryID uint,
	collectionID uint,
	title string,
	status string,
	startedAt *time.Time,
) models.Document {
	t.Helper()

	document := models.Document{
		UserID:          userID,
		Title:           title,
		FileURL:         "documents/" + title,
		Size:            2048,
		MimeType:        "application/pdf",
		Status:          status,
		ProcessingStage: "PARSING_STRUCTURE",
		CollectionID:    &collectionID,
		CategoryID:      &categoryID,
		IndexStartedAt:  startedAt,
	}
	mustCreateRecord(t, db, &document)
	return document
}

func assertDocumentStatus(t *testing.T, db *gorm.DB, documentID uint, wantStatus string, wantStage string) {
	t.Helper()

	var document models.Document
	if err := db.First(&document, documentID).Error; err != nil {
		t.Fatalf("carregar documento %d: %v", documentID, err)
	}

	if document.Status != wantStatus {
		t.Errorf("documento %q: status = %q, esperava %q", document.Title, document.Status, wantStatus)
	}
	if wantStage != "" && document.ProcessingStage != wantStage {
		t.Errorf("documento %q: stage = %q, esperava %q", document.Title, document.ProcessingStage, wantStage)
	}
}
