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

// ClaimPendingEvents reivindica eventos de fato: alem de incrementar a
// tentativa, move o evento para IN_FLIGHT e empurra available_at para frente
// pelo visibilityTimeout, tudo na mesma transacao da selecao.
//
// Antes disso a funcao so incrementava `attempts`: o status seguia PENDING e o
// available_at nao avancava. Como o `FOR UPDATE SKIP LOCKED` solta o lock no
// commit — e a publicacao no broker acontece DEPOIS, ja fora da transacao —
// existia uma janela em que o mesmo evento era visivel para qualquer outro
// dispatcher. Efeitos reais:
//
//   - com duas replicas do indexer, o mesmo documento era publicado 2x e
//     indexado 2x: custo de LLM dobrado e ReplaceIndexingResult concorrente;
//   - com uma replica, se MarkEventPublished falhasse, o evento ja publicado
//     seguia PENDING e era republicado a cada poll, indefinidamente.
//
// O visibilityTimeout tambem serve de rede: se o processo morrer entre o claim
// e a publicacao, o evento volta a ficar elegivel sozinho, sem intervencao.
func (r *gormOutboxRepository) ClaimPendingEvents(ctx context.Context, limit int, now time.Time, visibilityTimeout time.Duration) ([]models.OutboxEvent, error) {
	events := make([]models.OutboxEvent, 0, limit)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"status IN ? AND available_at <= ?",
				[]string{
					domain.OutboxEventStatusPending,
					domain.OutboxEventStatusFailed,
					// IN_FLIGHT entra aqui de proposito: se available_at ja
					// venceu, o worker que reivindicou morreu antes de publicar
					// e o evento precisa voltar para a fila.
					domain.OutboxEventStatusInFlight,
				},
				now,
			).
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

		if err := tx.Model(&models.OutboxEvent{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":       domain.OutboxEventStatusInFlight,
				"attempts":     gorm.Expr("attempts + 1"),
				"available_at": now.Add(visibilityTimeout),
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}

		// Os eventos devolvidos ao chamador precisam refletir o que foi
		// gravado: quem consome usa Attempts para calcular o backoff.
		for index := range events {
			events[index].Status = domain.OutboxEventStatusInFlight
			events[index].Attempts++
			events[index].AvailableAt = now.Add(visibilityTimeout)
			events[index].UpdatedAt = now
		}

		return nil
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
