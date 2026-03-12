package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormDocumentRepository struct {
	db *gorm.DB
}

func NewGormDocumentRepository(db *gorm.DB) domain.DocumentRepository {
	return &GormDocumentRepository{db: db}
}

func (r *GormDocumentRepository) CreateCategory(ctx context.Context, category *models.DocumentCategory) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *GormDocumentRepository) GetCategoryByID(ctx context.Context, id uint) (*models.DocumentCategory, error) {
	var category models.DocumentCategory
	if err := r.db.WithContext(ctx).First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *GormDocumentRepository) GetCategoryByName(ctx context.Context, name string) (*models.DocumentCategory, error) {
	var category models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *GormDocumentRepository) CreateDocument(ctx context.Context, doc *models.Document) error {
	return r.db.WithContext(ctx).Create(doc).Error
}
