//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestOutboxRepositoryClaimPendingEventsFiltersAndIncrementsAttempts(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormOutboxRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	eligiblePending := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   10,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 10},
		Attempts:      0,
		AvailableAt:   now.Add(-2 * time.Minute),
	}
	eligibleFailed := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   11,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusFailed,
		Payload:       map[string]any{"document_id": 11},
		Attempts:      2,
		AvailableAt:   now.Add(-time.Minute),
	}
	futurePending := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   12,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 12},
		Attempts:      0,
		AvailableAt:   now.Add(time.Minute),
	}

	mustCreateRecord(t, db, &eligiblePending)
	mustCreateRecord(t, db, &eligibleFailed)
	mustCreateRecord(t, db, &futurePending)

	claimed, err := repo.ClaimPendingEvents(ctx, 10, now)
	if err != nil {
		t.Fatalf("ClaimPendingEvents returned error: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed events, got %d", len(claimed))
	}
	if claimed[0].ID != eligiblePending.ID || claimed[1].ID != eligibleFailed.ID {
		t.Fatalf("expected claims in available_at order, got IDs %d and %d", claimed[0].ID, claimed[1].ID)
	}

	var storedPending models.OutboxEvent
	if err := db.First(&storedPending, eligiblePending.ID).Error; err != nil {
		t.Fatalf("load claimed pending event: %v", err)
	}
	if storedPending.Attempts != 1 {
		t.Fatalf("expected pending attempts 1, got %d", storedPending.Attempts)
	}

	var storedFailed models.OutboxEvent
	if err := db.First(&storedFailed, eligibleFailed.ID).Error; err != nil {
		t.Fatalf("load claimed failed event: %v", err)
	}
	if storedFailed.Attempts != 3 {
		t.Fatalf("expected failed attempts 3, got %d", storedFailed.Attempts)
	}

	var storedFuture models.OutboxEvent
	if err := db.First(&storedFuture, futurePending.ID).Error; err != nil {
		t.Fatalf("load future pending event: %v", err)
	}
	if storedFuture.Attempts != 0 {
		t.Fatalf("expected future event attempts to remain 0, got %d", storedFuture.Attempts)
	}
}
