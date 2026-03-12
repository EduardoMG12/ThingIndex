package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormProviderRepository struct {
	db *gorm.DB
}

func NewGormProviderRepository(db *gorm.DB) domain.ProviderRepository {
	return &gormProviderRepository{db: db}
}

func (r *gormProviderRepository) SaveConfig(ctx context.Context, config *models.UserProviderConfig) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"encrypted_data", "is_enabled", "updated_at"}),
	}).Create(config).Error
}

func (r *gormProviderRepository) GetConfig(ctx context.Context, userID uint, providerType string) (*models.UserProviderConfig, error) {
	var config models.UserProviderConfig
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider_type = ?", userID, providerType).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *gormProviderRepository) ListConfigs(ctx context.Context, userID uint) ([]models.UserProviderConfig, error) {
	var configs []models.UserProviderConfig
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *gormProviderRepository) DeleteConfig(ctx context.Context, userID uint, providerType string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND provider_type = ?", userID, providerType).Delete(&models.UserProviderConfig{}).Error
}
