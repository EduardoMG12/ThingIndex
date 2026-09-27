package repository

import (
	"context"
	appdb "thing-index/db"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormStructuredDataRepository struct {
	db *gorm.DB
}

func NewGormStructuredDataRepository(db *gorm.DB) domain.StructuredDataRepository {
	return &GormStructuredDataRepository{db: db}
}

func (r *GormStructuredDataRepository) ReplaceDocumentPages(ctx context.Context, documentID uint, pages []models.DocumentPage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentPage{}).Error; err != nil {
			return err
		}
		if len(pages) == 0 {
			return nil
		}
		return tx.CreateInBatches(&pages, appdb.InsertBatchSize).Error
	})
}

func (r *GormStructuredDataRepository) ReplaceDocumentChunks(ctx context.Context, documentID uint, chunks []models.DocumentChunk) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentChunk{}).Error; err != nil {
			return err
		}
		if len(chunks) == 0 {
			return nil
		}
		return tx.CreateInBatches(&chunks, appdb.InsertBatchSize).Error
	})
}

func (r *GormStructuredDataRepository) ReplaceOutlineNodes(ctx context.Context, documentID uint, nodes []models.DocumentOutlineNode) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentOutlineNode{}).Error; err != nil {
			return err
		}
		if len(nodes) == 0 {
			return nil
		}
		return createStructuredOutlineNodesWithParents(tx, nodes)
	})
}

func (r *GormStructuredDataRepository) GetDocumentPagesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.DocumentPage, error) {
	pages := make([]models.DocumentPage, 0)
	if len(documentIDs) == 0 {
		return pages, nil
	}

	if err := r.db.WithContext(ctx).
		Where("document_id IN ?", documentIDs).
		Order("document_id ASC, page_number ASC").
		Find(&pages).Error; err != nil {
		return nil, err
	}

	return pages, nil
}

func (r *GormStructuredDataRepository) GetParagraphNodesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.ParagraphNode, error) {
	nodes := make([]models.ParagraphNode, 0)
	if len(documentIDs) == 0 {
		return nodes, nil
	}

	if err := r.db.WithContext(ctx).
		Where("document_id IN ?", documentIDs).
		Order("document_id ASC, sequence_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	return nodes, nil
}

func (r *GormStructuredDataRepository) GetTableSectionsByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.TableSection, error) {
	sections := make([]models.TableSection, 0)
	if len(documentIDs) == 0 {
		return sections, nil
	}

	if err := r.db.WithContext(ctx).
		Where("document_id IN ?", documentIDs).
		Order("document_id ASC, sequence_order ASC").
		Find(&sections).Error; err != nil {
		return nil, err
	}

	return sections, nil
}

func (r *GormStructuredDataRepository) GetMetadataPairsByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.MetadataPair, error) {
	pairs := make([]models.MetadataPair, 0)
	if len(documentIDs) == 0 {
		return pairs, nil
	}

	if err := r.db.WithContext(ctx).
		Where("document_id IN ?", documentIDs).
		Order("document_id ASC, sequence_order ASC").
		Find(&pairs).Error; err != nil {
		return nil, err
	}

	return pairs, nil
}

func (r *GormStructuredDataRepository) GetDocumentChunksByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.DocumentChunk, error) {
	chunks := make([]models.DocumentChunk, 0)
	if len(documentIDs) == 0 {
		return chunks, nil
	}

	if err := r.db.WithContext(ctx).
		Where("document_id IN ?", documentIDs).
		Order("document_id ASC, sequence_order ASC").
		Find(&chunks).Error; err != nil {
		return nil, err
	}

	return chunks, nil
}

func (r *GormStructuredDataRepository) GetOutlineNodesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.DocumentOutlineNode, error) {
	nodes := make([]models.DocumentOutlineNode, 0)
	if len(documentIDs) == 0 {
		return nodes, nil
	}

	if err := r.db.WithContext(ctx).
		Where("document_id IN ?", documentIDs).
		Order("document_id ASC, sequence_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	return nodes, nil
}

func createStructuredOutlineNodesWithParents(tx *gorm.DB, nodes []models.DocumentOutlineNode) error {
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
