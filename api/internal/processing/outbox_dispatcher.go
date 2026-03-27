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
}

func NewOutboxDispatcher(repo domain.OutboxRepository, publisher domain.MessagePublisher, pollInterval time.Duration, batchSize int) *OutboxDispatcher {
	return &OutboxDispatcher{
		repo:         repo,
		publisher:    publisher,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

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
	events, err := d.repo.ClaimPendingEvents(ctx, d.batchSize, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}

	for _, event := range events {
		requestedEvent, err := decodeDocumentIndexRequestedEvent(event.Payload)
		if err != nil {
			_ = d.repo.MarkEventFailed(ctx, event.ID, err.Error(), nextBackoffTime(event.Attempts))
			continue
		}

		if err := d.publisher.PublishDocumentIndexRequested(ctx, requestedEvent); err != nil {
			_ = d.repo.MarkEventFailed(ctx, event.ID, err.Error(), nextBackoffTime(event.Attempts))
			continue
		}

		if err := d.repo.MarkEventPublished(ctx, event.ID, time.Now().UTC()); err != nil {
			return fmt.Errorf("mark outbox event published: %w", err)
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
