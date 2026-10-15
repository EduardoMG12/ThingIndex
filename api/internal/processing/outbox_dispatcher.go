package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"thing-index/internal/domain"
)

type OutboxDispatcher struct {
	repo         domain.OutboxRepository
	publisher    domain.MessagePublisher
	pollInterval time.Duration
	batchSize    int
	// Por quanto tempo um evento reivindicado fica invisivel para outros
	// dispatchers. Precisa ser folgadamente maior que o tempo de publicar no
	// broker; se o processo morrer nesse intervalo, o evento volta sozinho.
	visibilityTimeout time.Duration
}

func NewOutboxDispatcher(
	repo domain.OutboxRepository,
	publisher domain.MessagePublisher,
	pollInterval time.Duration,
	batchSize int,
	visibilityTimeout time.Duration,
) *OutboxDispatcher {
	if visibilityTimeout <= 0 {
		visibilityTimeout = defaultOutboxVisibilityTimeout
	}

	return &OutboxDispatcher{
		repo:              repo,
		publisher:         publisher,
		pollInterval:      pollInterval,
		batchSize:         batchSize,
		visibilityTimeout: visibilityTimeout,
	}
}

const defaultOutboxVisibilityTimeout = 2 * time.Minute

func (d *OutboxDispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		if err := d.dispatchOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("outbox dispatcher error: %v", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (d *OutboxDispatcher) dispatchOnce(ctx context.Context) error {
	events, err := d.repo.ClaimPendingEvents(ctx, d.batchSize, time.Now().UTC(), d.visibilityTimeout)
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}

	for _, event := range events {
		requestedEvent, err := decodeDocumentIndexRequestedEvent(event.Payload)
		if err != nil {
			if markErr := d.repo.MarkEventFailed(ctx, event.ID, err.Error(), nextBackoffTime(event.Attempts)); markErr != nil {
				log.Printf("outbox: falha ao marcar evento %d como FAILED apos decode invalido: %v", event.ID, markErr)
			}
			continue
		}

		if err := d.publisher.PublishDocumentIndexRequested(ctx, requestedEvent); err != nil {
			if markErr := d.repo.MarkEventFailed(ctx, event.ID, err.Error(), nextBackoffTime(event.Attempts)); markErr != nil {
				log.Printf("outbox: falha ao marcar evento %d como FAILED apos erro de publicacao: %v", event.ID, markErr)
			}
			continue
		}

		if err := d.repo.MarkEventPublished(ctx, event.ID, time.Now().UTC()); err != nil {
			// Nao aborta o lote: os seguintes ja foram reivindicados e ficariam
			// em IN_FLIGHT ate o visibility timeout.
			log.Printf("outbox: evento %d publicado no broker mas nao marcado como PUBLISHED: %v", event.ID, err)
		}
	}

	return nil
}

func decodeDocumentIndexRequestedEvent(payload map[string]any) (domain.DocumentIndexRequestedEvent, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.DocumentIndexRequestedEvent{}, err
	}

	var event domain.DocumentIndexRequestedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return domain.DocumentIndexRequestedEvent{}, err
	}
	if event.DocumentID == 0 || event.UserID == 0 {
		return domain.DocumentIndexRequestedEvent{}, fmt.Errorf("invalid event payload")
	}
	return event, nil
}

func nextBackoffTime(attempt int) time.Time {
	if attempt < 1 {
		attempt = 1
	}

	backoff := time.Duration(attempt*attempt) * time.Second
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}

	return time.Now().UTC().Add(backoff)
}
