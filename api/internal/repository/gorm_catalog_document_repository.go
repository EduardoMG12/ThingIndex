package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormCatalogDocumentRepository struct {
	db *gorm.DB
}

func NewGormCatalogDocumentRepository(db *gorm.DB) domain.CatalogDocumentRepository {
	return &GormCatalogDocumentRepository{db: db}
}

func (r *GormCatalogDocumentRepository) CreateDocument(ctx context.Context, doc *models.Document) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *GormCatalogDocumentRepository) GetDocumentByID(ctx context.Context, id uint, userID uint) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&doc).Error; err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentsByUserID(ctx context.Context, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentsByCategoryID(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Joins("JOIN collections ON collections.id = documents.collection_id").
		Where("documents.user_id = ? AND collections.category_id = ?", userID, categoryID).
		Order("documents.created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentsByCollectionID(ctx context.Context, collectionID uint, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Order("created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}
