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
		switch config.Family {
		case domain.FamilyStorage:
			if config.IsEnabled {
				if err := tx.Model(&models.UserProviderConfig{}).
					Where("user_id = ? AND family = ? AND provider_type <> ?", config.UserID, config.Family, config.ProviderType).
					Update("is_enabled", false).Error; err != nil {
					return err
				}
			}
		case domain.FamilyLLM:
			if config.EnabledForIndexing {
				if err := tx.Model(&models.UserProviderConfig{}).
					Where("user_id = ? AND family = ? AND provider_type <> ?", config.UserID, config.Family, config.ProviderType).
					Update("enabled_for_indexing", false).Error; err != nil {
					return err
				}
			}
			if config.EnabledForChat {
				if err := tx.Model(&models.UserProviderConfig{}).
					Where("user_id = ? AND family = ? AND provider_type <> ?", config.UserID, config.Family, config.ProviderType).
					Update("enabled_for_chat", false).Error; err != nil {
					return err
				}
			}
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "provider_type"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"family",
				"encrypted_data",
				"is_enabled",
				"enabled_for_indexing",
				"enabled_for_chat",
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

func (r *gormProviderRepository) GetConfigByLLMUsage(ctx context.Context, userID uint, usage string) (*models.UserProviderConfig, error) {
	var config models.UserProviderConfig

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND family = ?", userID, domain.FamilyLLM)

	switch usage {
	case domain.LLMUsageIndexing:
		query = query.Where("enabled_for_indexing = ?", true)
	case domain.LLMUsageChat:
		query = query.Where("enabled_for_chat = ?", true)
	default:
		return nil, gorm.ErrRecordNotFound
	}

	if err := query.Order("updated_at DESC, id DESC").First(&config).Error; err != nil {
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

func (r *gormProviderRepository) ListAllConfigs(ctx context.Context) ([]models.UserProviderConfig, error) {
	var configs []models.UserProviderConfig
	if err := r.db.WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *gormProviderRepository) ListModelCatalog(ctx context.Context, scopeKey, providerType string) ([]models.ProviderModelCatalog, error) {
	var items []models.ProviderModelCatalog
	if err := r.db.WithContext(ctx).
		Where("scope_key = ? AND provider_type = ? AND is_active = ?", scopeKey, providerType, true).
		Order("display_name ASC, model_id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *gormProviderRepository) ReplaceModelCatalog(ctx context.Context, scopeKey string, userID *uint, providerType string, entries []models.ProviderModelCatalog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ProviderModelCatalog{}).
			Where("scope_key = ? AND provider_type = ?", scopeKey, providerType).
			Update("is_active", false).Error; err != nil {
			return err
		}

		for i := range entries {
			entries[i].ScopeKey = scopeKey
			entries[i].UserID = userID
			entries[i].ProviderType = providerType
			entries[i].IsActive = true
		}

		if len(entries) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "scope_key"}, {Name: "provider_type"}, {Name: "model_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"user_id",
				"display_name",
				"metadata_json",
				"fetched_at",
				"is_active",
				"updated_at",
			}),
		}).Create(&entries).Error
	})
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

func (r *gormProviderRepository) SetLLMConfigUsage(ctx context.Context, userID uint, providerType, usage string, enabled bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		column := ""
		switch usage {
		case domain.LLMUsageIndexing:
			column = "enabled_for_indexing"
		case domain.LLMUsageChat:
			column = "enabled_for_chat"
		default:
			return gorm.ErrRecordNotFound
		}

		if enabled {
			if err := tx.Model(&models.UserProviderConfig{}).
				Where("user_id = ? AND family = ? AND provider_type <> ?", userID, domain.FamilyLLM, providerType).
				Update(column, false).Error; err != nil {
				return err
			}
		}

		result := tx.Model(&models.UserProviderConfig{}).
			Where("user_id = ? AND family = ? AND provider_type = ?", userID, domain.FamilyLLM, providerType).
			Update(column, enabled)
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
