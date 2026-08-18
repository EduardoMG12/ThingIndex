package repository

import (
	"context"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
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
			if err := tx.Create(&result.DocumentPages).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.ParagraphNode{}).Error; err != nil {
			return err
		}
		if len(result.ParagraphNodes) > 0 {
			if err := tx.Create(&result.ParagraphNodes).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.TableSection{}).Error; err != nil {
			return err
		}
		if len(result.TableSections) > 0 {
			if err := tx.Create(&result.TableSections).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.MetadataPair{}).Error; err != nil {
			return err
		}
		if len(result.MetadataPairs) > 0 {
			if err := tx.Create(&result.MetadataPairs).Error; err != nil {
				return err
			}
		}

		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentChunk{}).Error; err != nil {
			return err
		}
		if len(result.Chunks) > 0 {
			if err := tx.Create(&result.Chunks).Error; err != nil {
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

	if err := tx.Create(&nodes).Error; err != nil {
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
