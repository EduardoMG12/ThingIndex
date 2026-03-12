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

func (r *GormDocumentRepository) GetCategoryByID(ctx context.Context, id uint, userID uint) (*models.DocumentCategory, error) {
	var category models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *GormDocumentRepository) GetCategoryByName(ctx context.Context, name string, userID uint) (*models.DocumentCategory, error) {
	var category models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("name = ? AND user_id = ?", name, userID).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *GormDocumentRepository) GetCategoriesByUserID(ctx context.Context, userID uint) ([]models.DocumentCategory, error) {
	var categories []models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *GormDocumentRepository) CreateDocument(ctx context.Context, doc *models.Document) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *GormDocumentRepository) GetDocumentsByUserID(ctx context.Context, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).Preload("Category").Where("user_id = ?", userID).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *GormDocumentRepository) GetDocumentsByCategoryID(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).Preload("Category").Where("category_id = ? AND user_id = ?", categoryID, userID).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}
