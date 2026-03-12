package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormCollectionContextRepository struct {
	db *gorm.DB
}

func NewGormCollectionContextRepository(db *gorm.DB) domain.CollectionContextRepository {
	return &GormCollectionContextRepository{db: db}
}

func (r *GormCollectionContextRepository) GetCollectionContext(ctx context.Context, userID uint, collectionID uint) (*domain.CollectionContext, error) {
	var collection models.Collection
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", collectionID, userID).
		First(&collection).Error; err != nil {
		return nil, err
	}

	var documents []models.Document
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Order("id ASC").
		Find(&documents).Error; err != nil {
		return nil, err
	}

	documentIDs := make([]uint, 0, len(documents))
	for _, doc := range documents {
		documentIDs = append(documentIDs, doc.ID)
	}

	paragraphNodes := make([]models.ParagraphNode, 0)
	tableSections := make([]models.TableSection, 0)
	metadataPairs := make([]models.MetadataPair, 0)

	if len(documentIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Find(&paragraphNodes).Error; err != nil {
			return nil, err
		}

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Find(&tableSections).Error; err != nil {
			return nil, err
		}

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Find(&metadataPairs).Error; err != nil {
			return nil, err
		}
	}

	return &domain.CollectionContext{
		Collection:     &collection,
		Documents:      documents,
		ParagraphNodes: paragraphNodes,
		TableSections:  tableSections,
		MetadataPairs:  metadataPairs,
	}, nil
}
