//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// Regressao P1-4: ReplaceIndexingResult fazia um unico INSERT com todos os
// paragrafos. ParagraphNode tem 8 colunas, entao acima de ~8191 paragrafos o
// statement passava de 65535 parametros, o pgx recusava e o documento ia para
// FAILED_PERSISTING_INDEX. A conexao deste teste NAO tem CreateBatchSize: o
// que precisa segurar e o lote explicito do repositorio.
func TestReplaceIndexingResultPersisteDocumentoComMaisDeNoveMilParagrafos(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormDocumentProcessingRepository{db: db}

	user := seedIntegrationUser(t, db, "lote")
	category := seedIntegrationCategory(t, db, user.ID, "Grandes")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Volumes")
	document := seedProcessingDocument(t, db, user.ID, category.ID, collection.ID, "grande.pdf", domain.DocumentStatusProcessing, nil)

	const total = 9500
	paragrafos := make([]models.ParagraphNode, 0, total)
	for i := 0; i < total; i++ {
		pagina := i/20 + 1
		paragrafos = append(paragrafos, models.ParagraphNode{
			DocumentID:    document.ID,
			SequenceOrder: i,
			PageNumber:    &pagina,
			Content:       fmt.Sprintf("paragrafo %d", i),
		})
	}

	if err := repo.ReplaceIndexingResult(context.Background(), document.ID, domain.IndexingResult{ParagraphNodes: paragrafos}); err != nil {
		t.Fatalf("persistir %d paragrafos falhou: %v", total, err)
	}

	var gravados int64
	if err := db.Model(&models.ParagraphNode{}).Where("document_id = ?", document.ID).Count(&gravados).Error; err != nil {
		t.Fatalf("contar paragrafos: %v", err)
	}
	if gravados != total {
		t.Fatalf("esperava %d paragrafos gravados, vieram %d", total, gravados)
	}
}
