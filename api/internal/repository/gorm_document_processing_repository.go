package repository

import (
	"context"

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

func (r *gormDocumentProcessingRepository) UpdateDocumentStatus(ctx context.Context, documentID uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("id = ?", documentID).
		Update("status", status).Error
}

func (r *gormDocumentProcessingRepository) ReplaceIndexingResult(ctx context.Context, documentID uint, result domain.IndexingResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
			if err := tx.Create(&result.OutlineNodes).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.Document{}).
			Where("id = ?", documentID).
			Update("status", domain.DocumentStatusIndexed).Error
	})
}
