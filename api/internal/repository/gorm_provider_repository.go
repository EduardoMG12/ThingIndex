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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.UserProviderConfig{}).
			Where("user_id = ? AND family = ? AND provider_type <> ?", config.UserID, config.Family, config.ProviderType).
			Update("is_enabled", false).Error; err != nil {
			return err
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "provider_type"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"family",
				"encrypted_data",
				"is_enabled",
				"updated_at",
			}),
		}).Create(config).Error
	})
}

func (r *gormProviderRepository) GetConfig(ctx context.Context, userID uint, providerType string) (*models.UserProviderConfig, error) {
	var config models.UserProviderConfig
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider_type = ?", userID, providerType).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *gormProviderRepository) GetConfigByFamily(ctx context.Context, userID uint, family string) (*models.UserProviderConfig, error) {
	var config models.UserProviderConfig
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND family = ? AND is_enabled = ?", userID, family, true).
		Order("updated_at DESC, id DESC").
		First(&config).Error; err != nil {
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

func (r *gormProviderRepository) SetConfigEnabled(ctx context.Context, userID uint, family, providerType string, enabled bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if enabled {
			if err := tx.Model(&models.UserProviderConfig{}).
				Where("user_id = ? AND family = ? AND provider_type <> ?", userID, family, providerType).
				Update("is_enabled", false).Error; err != nil {
				return err
			}
		}

		result := tx.Model(&models.UserProviderConfig{}).
			Where("user_id = ? AND family = ? AND provider_type = ?", userID, family, providerType).
			Update("is_enabled", enabled)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *gormProviderRepository) DeleteConfig(ctx context.Context, userID uint, providerType string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND provider_type = ?", userID, providerType).Delete(&models.UserProviderConfig{}).Error
}
