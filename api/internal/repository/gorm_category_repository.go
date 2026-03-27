package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormCategoryRepository struct {
	db *gorm.DB
}

func NewGormCategoryRepository(db *gorm.DB) domain.CategoryCatalogRepository {
	return &GormCategoryRepository{db: db}
}

func (r *GormCategoryRepository) CreateCategory(ctx context.Context, category *models.DocumentCategory) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *GormCategoryRepository) UpdateCategory(ctx context.Context, category *models.DocumentCategory) error {
	return r.db.WithContext(ctx).Save(category).Error
}

func (r *GormCategoryRepository) GetCategoryByID(ctx context.Context, id uint, userID uint) (*models.DocumentCategory, error) {
	var category models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *GormCategoryRepository) GetCategoryByNormalizedName(ctx context.Context, normalizedName string, userID uint) (*models.DocumentCategory, error) {
	var category models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("normalized_name = ? AND user_id = ?", normalizedName, userID).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *GormCategoryRepository) ListCategoriesByUserID(ctx context.Context, userID uint) ([]models.DocumentCategory, error) {
	var categories []models.DocumentCategory
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}
