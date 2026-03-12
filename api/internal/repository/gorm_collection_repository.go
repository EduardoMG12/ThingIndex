package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormCollectionRepository struct {
	db *gorm.DB
}

func NewGormCollectionRepository(db *gorm.DB) domain.CollectionRepository {
	return &GormCollectionRepository{db: db}
}

func (r *GormCollectionRepository) CreateCollection(ctx context.Context, collection *models.Collection) error {
	return r.db.WithContext(ctx).Create(collection).Error
}

func (r *GormCollectionRepository) UpdateCollection(ctx context.Context, collection *models.Collection) error {
	return r.db.WithContext(ctx).Save(collection).Error
}

func (r *GormCollectionRepository) GetCollectionByID(ctx context.Context, id uint, userID uint) (*models.Collection, error) {
	var collection models.Collection
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&collection).Error; err != nil {
		return nil, err
	}

	return &collection, nil
}

func (r *GormCollectionRepository) GetCollectionByNormalizedName(ctx context.Context, normalizedName string, userID uint) (*models.Collection, error) {
	var collection models.Collection
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("normalized_name = ? AND user_id = ?", normalizedName, userID).
		First(&collection).Error; err != nil {
		return nil, err
	}

	return &collection, nil
}

func (r *GormCollectionRepository) ListCollectionsByUserID(ctx context.Context, userID uint) ([]models.Collection, error) {
	var collections []models.Collection
	if err := r.db.WithContext(ctx).
		Model(&models.Collection{}).
		Preload("Category").
		Select("collections.*, COUNT(documents.id) AS document_count").
		Joins("LEFT JOIN documents ON documents.collection_id = collections.id AND documents.deleted_at IS NULL").
		Where("collections.user_id = ?", userID).
		Group("collections.id").
		Order("collections.name ASC").
		Find(&collections).Error; err != nil {
		return nil, err
	}

	return collections, nil
}

func (r *GormCollectionRepository) CountDocuments(ctx context.Context, userID uint, collectionID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
