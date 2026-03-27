package repository

import (
	"context"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormOutboxRepository struct {
	db *gorm.DB
}

func NewGormOutboxRepository(db *gorm.DB) domain.OutboxRepository {
	return &gormOutboxRepository{db: db}
}

func (r *gormOutboxRepository) ClaimPendingEvents(ctx context.Context, limit int, now time.Time) ([]models.OutboxEvent, error) {
	events := make([]models.OutboxEvent, 0, limit)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ? AND available_at <= ?", []string{domain.OutboxEventStatusPending, domain.OutboxEventStatusFailed}, now).
			Order("available_at ASC, id ASC").
			Limit(limit).
			Find(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.ID)
		}

		return tx.Model(&models.OutboxEvent{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"attempts":   gorm.Expr("attempts + 1"),
				"updated_at": now,
			}).Error
	})
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (r *gormOutboxRepository) MarkEventPublished(ctx context.Context, eventID uint, publishedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]any{
			"status":       domain.OutboxEventStatusPublished,
			"published_at": publishedAt,
			"last_error":   "",
			"updated_at":   publishedAt,
		}).Error
}

func (r *gormOutboxRepository) MarkEventFailed(ctx context.Context, eventID uint, errMsg string, availableAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]any{
			"status":       domain.OutboxEventStatusFailed,
			"last_error":   errMsg,
			"available_at": availableAt,
			"updated_at":   time.Now().UTC(),
		}).Error
}
