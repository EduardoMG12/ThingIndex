package processing

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/queue"
)

type processingRepoStub struct {
	document      *models.Document
	updatedStates []domain.DocumentProcessingState
}

func (r *processingRepoStub) GetDocumentByIDForProcessing(context.Context, uint) (*models.Document, error) {
	if r.document == nil {
		return nil, domain.ErrDocumentNotFound
	}
	return r.document, nil
}

func (r *processingRepoStub) UpdateDocumentProcessingState(_ context.Context, _ uint, state domain.DocumentProcessingState) error {
	r.updatedStates = append(r.updatedStates, state)
	return nil
}

func (r *processingRepoStub) ReplaceIndexingResult(context.Context, uint, domain.IndexingResult) error {
	return nil
}

func TestMarkRetryScheduledPersistsQueuedOperationalState(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:                  7,
			Status:              domain.DocumentStatusFailed,
			ProcessingStage:     "FAILED_PARSING_STRUCTURE",
			LastProcessingError: "parse pdf: broken xref table",
		},
	}
	processor := NewDocumentProcessor(repo, nil, nil, nil, nil, nil)

	processor.MarkRetryScheduled(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 7}, queue.DocumentIndexFailureDisposition{
		NextRetryAttempt: 1,
		MaxRetries:       3,
		RetryDelay:       30 * time.Second,
		Err:              fmt.Errorf("%w: parse pdf", domain.ErrDocumentIndexing),
	})

	if len(repo.updatedStates) != 1 {
		t.Fatalf("expected one state update, got %d", len(repo.updatedStates))
	}

	state := repo.updatedStates[0]
	if state.Status != domain.DocumentStatusQueued || state.Stage != "RETRY_SCHEDULED" {
		t.Fatalf("unexpected retry scheduled state: %#v", state)
	}
	if !strings.Contains(state.StatusDetail, "tentativa 1 de 3") {
		t.Fatalf("expected retry detail to mention attempt count, got %#v", state)
	}
	if state.LastError != "parse pdf: broken xref table" {
		t.Fatalf("expected original processing error to be preserved, got %#v", state)
	}
}

func TestMarkMovedToDLQPersistsFailedOperationalState(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:                  8,
			Status:              domain.DocumentStatusFailed,
			ProcessingStage:     "FAILED_PERSISTING_INDEX",
			LastProcessingError: "database unavailable",
		},
	}
	processor := NewDocumentProcessor(repo, nil, nil, nil, nil, nil)

	processor.MarkMovedToDLQ(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 8}, queue.DocumentIndexFailureDisposition{
		RetryCount: 3,
		MaxRetries: 3,
		Err:        fmt.Errorf("%w: persist indexing result", domain.ErrDocumentIndexing),
	})

	if len(repo.updatedStates) != 1 {
		t.Fatalf("expected one state update, got %d", len(repo.updatedStates))
	}

	state := repo.updatedStates[0]
	if state.Status != domain.DocumentStatusFailed || state.Stage != "MOVED_TO_DLQ" {
		t.Fatalf("unexpected dlq state: %#v", state)
	}
	if !strings.Contains(state.StatusDetail, "3 retry(s)") {
		t.Fatalf("expected dlq detail to mention retry exhaustion, got %#v", state)
	}
}

func TestMarkPermanentFailureKeepsOrphanedStateWhenSourceIsMissing(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:                  9,
			Status:              domain.DocumentStatusOrphaned,
			ProcessingStage:     "SOURCE_MISSING",
			LastProcessingError: "source file missing in configured storage",
		},
	}
	processor := NewDocumentProcessor(repo, nil, nil, nil, nil, nil)

	processor.MarkPermanentFailure(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 9}, queue.DocumentIndexFailureDisposition{
		Err: fmt.Errorf("%w: source file missing", domain.ErrDocumentSourceMissing),
	})

	if len(repo.updatedStates) != 1 {
		t.Fatalf("expected one state update, got %d", len(repo.updatedStates))
	}

	state := repo.updatedStates[0]
	if state.Status != domain.DocumentStatusOrphaned || state.Stage != "SOURCE_MISSING" {
		t.Fatalf("unexpected permanent failure state: %#v", state)
	}
	if !strings.Contains(state.StatusDetail, "Nenhuma nova tentativa automatica") {
		t.Fatalf("expected permanent failure detail to mention no automatic retry, got %#v", state)
	}
}
