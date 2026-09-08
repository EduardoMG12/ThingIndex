package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type GormParagraphHighlightRepository struct {
	db *gorm.DB
}

func NewGormParagraphHighlightRepository(db *gorm.DB) domain.ParagraphHighlightRepository {
	return &GormParagraphHighlightRepository{db: db}
}

func (r *GormParagraphHighlightRepository) ListHighlights(ctx context.Context, userID, collectionID uint) ([]models.ParagraphHighlight, error) {
	var highlights []models.ParagraphHighlight
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Order("document_id ASC, paragraph_node_id ASC").
		Find(&highlights).Error
	return highlights, err
}

// UpsertHighlight valida a cadeia inteira numa consulta so -- paragrafo vivo,
// de documento vivo, de colecao viva do usuario -- antes de gravar. E o que a
// constituicao pede para escopo transitivo, e e o que impede marcar o
// paragrafo de outra pessoa sabendo so o id.
func (r *GormParagraphHighlightRepository) UpsertHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint, color string) (*models.ParagraphHighlight, error) {
	var owner struct{ DocumentID uint }
	err := r.db.WithContext(ctx).
		Table("paragraph_nodes AS p").
		Select("p.document_id").
		Joins("JOIN documents d ON d.id = p.document_id AND d.deleted_at IS NULL").
		Joins("JOIN collections c ON c.id = d.collection_id AND c.deleted_at IS NULL").
		Where("p.id = ? AND p.deleted_at IS NULL AND c.id = ? AND c.user_id = ?", paragraphNodeID, collectionID, userID).
		Take(&owner).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrParagraphNotInCollection
	}
	if err != nil {
		return nil, err
	}

	highlight := models.ParagraphHighlight{
		UserID:          userID,
		CollectionID:    collectionID,
		DocumentID:      owner.DocumentID,
		ParagraphNodeID: paragraphNodeID,
		Color:           color,
	}
	err = r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "paragraph_node_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"color", "collection_id", "document_id", "updated_at", "deleted_at"}),
		}).
		Create(&highlight).Error
	if err != nil {
		return nil, err
	}
	// O Create com ON CONFLICT nao devolve a linha atualizada com o id antigo;
	// releitura barata para o chamador ter o registro real.
	var saved models.ParagraphHighlight
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND paragraph_node_id = ?", userID, paragraphNodeID).
		Take(&saved).Error; err != nil {
		return nil, err
	}
	return &saved, nil
}

func (r *GormParagraphHighlightRepository) DeleteHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint) error {
	// Apagar de verdade, nao soft delete: a marca nao tem trilha de auditoria
	// e um soft delete bloquearia o indice unico na proxima marca.
	return r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ? AND collection_id = ? AND paragraph_node_id = ?", userID, collectionID, paragraphNodeID).
		Delete(&models.ParagraphHighlight{}).Error
}

// UpdateCollectionMetadataKey grava uma unica chave em collections.metadata
// sem tocar nas outras, no proprio banco (jsonb ||), para duas escritas
// concorrentes em chaves diferentes nao se apagarem.
func (r *GormCollectionRepository) UpdateCollectionMetadataKey(ctx context.Context, userID, collectionID uint, key string, valueJSON []byte) error {
	result := r.db.WithContext(ctx).
		Model(&models.Collection{}).
		Where("id = ? AND user_id = ?", collectionID, userID).
		Updates(map[string]any{
			"metadata":   gorm.Expr("COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(?::text, ?::jsonb)", key, string(valueJSON)),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrCollectionNotFound
	}
	return nil
}
