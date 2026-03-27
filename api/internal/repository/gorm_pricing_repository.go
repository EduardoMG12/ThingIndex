package repository

import (
	"context"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormPricingRepository struct {
	db *gorm.DB
}

func NewGormPricingRepository(db *gorm.DB) domain.PricingSourceRepository {
	return &gormPricingRepository{db: db}
}

func (r *gormPricingRepository) SaveSource(ctx context.Context, source *models.PricingSource) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"source_url",
			"is_enabled",
			"updated_at",
		}),
	}).Create(source).Error
}

func (r *gormPricingRepository) GetSource(ctx context.Context, userID, sourceID uint) (*models.PricingSource, error) {
	var source models.PricingSource
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sourceID, userID).
		First(&source).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *gormPricingRepository) ListSources(ctx context.Context, userID uint) ([]models.PricingSource, error) {
	var sources []models.PricingSource
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider_type ASC, id ASC").
		Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *gormPricingRepository) ListEnabledSources(ctx context.Context, userID uint) ([]models.PricingSource, error) {
	var sources []models.PricingSource
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_enabled = ?", userID, true).
		Order("provider_type ASC, id ASC").
		Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *gormPricingRepository) ListAllEnabledSources(ctx context.Context) ([]models.PricingSource, error) {
	var sources []models.PricingSource
	if err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("user_id ASC, provider_type ASC, id ASC").
		Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *gormPricingRepository) ReplacePrices(ctx context.Context, source *models.PricingSource, prices []models.ModelPrice, syncedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		history := make([]models.ModelPriceHistory, 0, len(prices))
		for i := range prices {
			history = append(history, models.ModelPriceHistory{
				UserID:                       source.UserID,
				PricingSourceID:              source.ID,
				ProviderType:                 source.ProviderType,
				ModelKey:                     prices[i].ModelKey,
				ModelName:                    prices[i].ModelName,
				Unit:                         prices[i].Unit,
				Currency:                     prices[i].Currency,
				FlatPriceUSD:                 prices[i].FlatPriceUSD,
				InputCostPerMillionUSD:       prices[i].InputCostPerMillionUSD,
				OutputCostPerMillionUSD:      prices[i].OutputCostPerMillionUSD,
				CachedInputCostPerMillionUSD: prices[i].CachedInputCostPerMillionUSD,
				ContextCostPerMillionUSD:     prices[i].ContextCostPerMillionUSD,
				PriceNotes:                   prices[i].PriceNotes,
				CapturedAt:                   syncedAt,
			})
		}

		if err := tx.Where("user_id = ? AND pricing_source_id = ?", source.UserID, source.ID).
			Delete(&models.ModelPrice{}).Error; err != nil {
			return err
		}

		for i := range prices {
			prices[i].UserID = source.UserID
			prices[i].PricingSourceID = source.ID
			prices[i].ProviderType = source.ProviderType
			prices[i].LastSyncedAt = syncedAt
		}
		if len(prices) > 0 {
			if err := tx.Create(&prices).Error; err != nil {
				return err
			}
		}
		if len(history) > 0 {
			if err := tx.Create(&history).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *gormPricingRepository) MarkSourceSyncSucceeded(ctx context.Context, source *models.PricingSource, finishedAt time.Time, durationMs int64, priceCount int) error {
	return r.db.WithContext(ctx).
		Model(&models.PricingSource{}).
		Where("id = ? AND user_id = ?", source.ID, source.UserID).
		Updates(map[string]any{
			"last_synced_at":        finishedAt.UTC(),
			"last_sync_status":      "SUCCESS",
			"last_sync_error":       "",
			"last_sync_duration_ms": durationMs,
			"last_price_count":      priceCount,
		}).Error
}

func (r *gormPricingRepository) MarkSourceSyncFailed(ctx context.Context, source *models.PricingSource, finishedAt time.Time, durationMs int64, failure string) error {
	return r.db.WithContext(ctx).
		Model(&models.PricingSource{}).
		Where("id = ? AND user_id = ?", source.ID, source.UserID).
		Updates(map[string]any{
			"last_synced_at":        finishedAt.UTC(),
			"last_sync_status":      "FAILED",
			"last_sync_error":       failure,
			"last_sync_duration_ms": durationMs,
		}).Error
}

func (r *gormPricingRepository) ListPrices(ctx context.Context, userID uint) ([]models.ModelPrice, error) {
	var prices []models.ModelPrice
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider_type ASC, model_name ASC").
		Find(&prices).Error; err != nil {
		return nil, err
	}
	return prices, nil
}

func (r *gormPricingRepository) ListPriceHistory(ctx context.Context, userID uint, limit int) ([]models.ModelPriceHistory, error) {
	if limit <= 0 {
		limit = 100
	}

	var history []models.ModelPriceHistory
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("captured_at DESC, id DESC").
		Limit(limit).
		Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}
