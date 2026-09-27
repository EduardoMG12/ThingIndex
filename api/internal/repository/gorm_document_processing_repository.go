package repository

import (
	"context"
	appdb "thing-index/db"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormDocumentProcessingRepository struct {
	db *gorm.DB
}

func NewGormDocumentProcessingRepository(db *gorm.DB) domain.DocumentProcessingRepository {
	return &gormDocumentProcessingRepository{db: db}
}

func (r *gormDocumentProcessingRepository) GetDocumentByIDForProcessing(ctx context.Context, id uint) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		First(&doc, id).Error; err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *gormDocumentProcessingRepository) UpdateDocumentProcessingState(ctx context.Context, documentID uint, state domain.DocumentProcessingState) error {
	updates := map[string]any{
		"status": state.Status,
	}
	if state.Stage != "" {
		updates["processing_stage"] = state.Stage
	}
	if state.StatusDetail != "" || state.Stage != "" {
		updates["status_detail"] = state.StatusDetail
	}
	if state.LastError != "" || state.Status == domain.DocumentStatusIndexed {
		updates["last_processing_error"] = state.LastError
	}
	if state.StartedAt != nil {
		updates["index_started_at"] = state.StartedAt.UTC()
	}
	if state.CompletedAt != nil {
		updates["indexed_at"] = state.CompletedAt.UTC()
	}

	return r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("id = ?", documentID).
		Updates(updates).Error
}

// ReapStuckProcessingDocuments libera documentos que ficaram presos em
// PROCESSING.
//
// O timeout por mensagem resolve o worker VIVO que demora demais. Nao resolve o
// worker que MORREU no meio do trabalho (OOM kill, SIGKILL, queda do no): nesse
// caso ninguem executa o codigo que marcaria FAILED, e o documento fica
// PROCESSING para sempre. Do lado do usuario e o pior estado possivel -- a
// interface mostra "processando" indefinidamente, sem erro e sem opcao de
// reenviar.
//
// A condicao usa index_started_at, gravado por UpdateDocumentProcessingState no
// inicio do Process. Documentos sem esse campo sao ignorados de proposito: sem
// referencia temporal nao da para afirmar que estao presos, e marcar FAILED por
// engano um documento que acabou de entrar seria pior que esperar.
func (r *gormDocumentProcessingRepository) ReapStuckProcessingDocuments(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("status = ?", domain.DocumentStatusProcessing).
		Where("index_started_at IS NOT NULL AND index_started_at < ?", olderThan.UTC()).
		Updates(map[string]any{
			"status":                domain.DocumentStatusFailed,
			"processing_stage":      "FAILED_STUCK_IN_PROCESSING",
			"status_detail":         "O processamento foi interrompido sem concluir. Reenvie o documento para tentar de novo.",
			"last_processing_error": "worker encerrou durante o processamento (documento preso em PROCESSING)",
			"updated_at":            time.Now().UTC(),
		})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// RequeueStuckQueuedDocuments reenfileira documentos presos em QUEUED cuja
// mensagem se perdeu DEPOIS do outbox.
//
// O outbox garante que o evento chega ao broker; publisher confirms garantem
// que o broker o persistiu. Ainda assim a mensagem pode sumir depois disso: fila
// apagada e recriada, purge manual, volume do RabbitMQ perdido. Nesse caso o
// evento esta PUBLISHED, o documento segue QUEUED e nada no sistema volta a
// olhar para ele.
//
// Criterio (todos juntos):
//   - documento em QUEUED, nao excluido, sem mudanca desde antes de olderThan;
//   - o evento MAIS RECENTE do documento no outbox esta PUBLISHED, com
//     published_at anterior a olderThan. Se o ultimo evento ainda esta PENDING,
//     IN_FLIGHT ou FAILED, o despachante continua responsavel por ele e nao ha
//     o que reconciliar -- criar outro seria duplicar.
//
// Para cada um, na mesma transacao: um evento PENDING novo (com requested_at
// novo, entao o consumidor nao o confunde com reentrega) e updated_at do
// documento avancado, o que impede o proximo ciclo de reenfileira-lo de novo
// antes do novo evento ter a chance de ser processado. FOR UPDATE SKIP LOCKED
// evita que duas replicas do indexer reenfileirem o mesmo documento.
func (r *gormDocumentProcessingRepository) RequeueStuckQueuedDocuments(ctx context.Context, olderThan time.Time) (int64, error) {
	cutoff := olderThan.UTC()
	var requeued int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var documents []models.Document
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("documents.status = ? AND documents.updated_at < ?", domain.DocumentStatusQueued, cutoff).
			Where(`EXISTS (
				SELECT 1 FROM outbox_events latest
				WHERE latest.aggregate_type = ? AND latest.aggregate_id = documents.id
				  AND latest.id = (
					SELECT MAX(e.id) FROM outbox_events e
					WHERE e.aggregate_type = ? AND e.aggregate_id = documents.id
				  )
				  AND latest.status = ?
				  AND latest.published_at IS NOT NULL AND latest.published_at < ?
			)`, domain.OutboxAggregateDocument, domain.OutboxAggregateDocument, domain.OutboxEventStatusPublished, cutoff).
			Order("documents.id ASC").
			Find(&documents).Error; err != nil {
			return err
		}
		if len(documents) == 0 {
			return nil
		}

		now := time.Now().UTC()
		ids := make([]uint, 0, len(documents))
		events := make([]models.OutboxEvent, 0, len(documents))
		for _, document := range documents {
			ids = append(ids, document.ID)
			events = append(events, models.OutboxEvent{
				AggregateType: domain.OutboxAggregateDocument,
				AggregateID:   document.ID,
				EventType:     domain.OutboxEventDocumentIndexRequested,
				Status:        domain.OutboxEventStatusPending,
				Payload: map[string]any{
					"document_id":  document.ID,
					"user_id":      document.UserID,
					"requested_at": now,
				},
				AvailableAt: now,
			})
		}

		if err := tx.Create(&events).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Document{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status_detail": "Documento reenfileirado automaticamente: a solicitacao anterior de indexacao nao foi processada.",
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}

		requeued = int64(len(documents))
		return nil
	})
	if err != nil {
		return 0, err
	}

	return requeued, nil
}

func (r *gormDocumentProcessingRepository) CreateProcessingAttempt(ctx context.Context, attempt *models.DocumentProcessingAttempt) error {
	if attempt == nil {
		return nil
	}

	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *gormDocumentProcessingRepository) ReplaceIndexingResult(ctx context.Context, documentID uint, result domain.IndexingResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentPage{}).Error; err != nil {
			return err
		}
		if len(result.DocumentPages) > 0 {
			if err := tx.CreateInBatches(&result.DocumentPages, appdb.InsertBatchSize).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.ParagraphNode{}).Error; err != nil {
			return err
		}
		if len(result.ParagraphNodes) > 0 {
			if err := tx.CreateInBatches(&result.ParagraphNodes, appdb.InsertBatchSize).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.TableSection{}).Error; err != nil {
			return err
		}
		if len(result.TableSections) > 0 {
			if err := tx.CreateInBatches(&result.TableSections, appdb.InsertBatchSize).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.MetadataPair{}).Error; err != nil {
			return err
		}
		if len(result.MetadataPairs) > 0 {
			if err := tx.CreateInBatches(&result.MetadataPairs, appdb.InsertBatchSize).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentChunk{}).Error; err != nil {
			return err
		}
		if len(result.Chunks) > 0 {
			if err := tx.CreateInBatches(&result.Chunks, appdb.InsertBatchSize).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentOutlineNode{}).Error; err != nil {
			return err
		}
		if len(result.OutlineNodes) > 0 {
			if err := createOutlineNodesWithParents(tx, result.OutlineNodes); err != nil {
				return err
			}
		}

		completedAt := time.Now().UTC()
		return tx.Model(&models.Document{}).
			Where("id = ?", documentID).
			Updates(map[string]any{
				"status":                domain.DocumentStatusIndexed,
				"processing_stage":      "COMPLETED",
				"status_detail":         "Indexacao concluida e pronta para consulta.",
				"indexed_at":            completedAt,
				"last_processing_error": "",
			}).Error
	})
}

func createOutlineNodesWithParents(tx *gorm.DB, nodes []models.DocumentOutlineNode) error {
	for idx := range nodes {
		nodes[idx].ParentID = nil
	}

	if err := tx.CreateInBatches(&nodes, appdb.InsertBatchSize).Error; err != nil {
		return err
	}

	nodeIDToID := make(map[string]uint, len(nodes))
	for _, node := range nodes {
		nodeIDToID[node.NodeID] = node.ID
	}

	for _, node := range nodes {
		parentNodeID, ok := node.Metadata["parent_node_id"].(string)
		if !ok || parentNodeID == "" {
			continue
		}

		parentID, ok := nodeIDToID[parentNodeID]
		if !ok {
			continue
		}

		if err := tx.Model(&models.DocumentOutlineNode{}).
			Where("id = ?", node.ID).
			Update("parent_id", parentID).Error; err != nil {
			return err
		}
	}

	return nil
}
