package processing

import (
	"context"
	"errors"
	"log"
	"time"

	"thing-index/internal/domain"
)

// StuckDocumentReaper marca FAILED documentos presos em PROCESSING porque o
// worker morreu (OOM, SIGKILL); worker lento e coberto pelo timeout por mensagem.
// Tambem republica (PENDING no outbox) documentos em QUEUED cujo evento foi
// PUBLISHED ha mais de queuedAfter: a mensagem se perdeu depois do broker.
type StuckDocumentReaper struct {
	repo domain.DocumentProcessingRepository
	// Precisa ser bem maior que o timeout por mensagem.
	stuckAfter time.Duration
	// Quanto tempo com o ultimo evento PUBLISHED e o documento ainda em QUEUED
	// caracteriza mensagem perdida. Zero desliga a reconciliacao.
	queuedAfter  time.Duration
	pollInterval time.Duration
}

func NewStuckDocumentReaper(
	repo domain.DocumentProcessingRepository,
	stuckAfter time.Duration,
	queuedAfter time.Duration,
	pollInterval time.Duration,
) *StuckDocumentReaper {
	return &StuckDocumentReaper{
		repo:         repo,
		stuckAfter:   stuckAfter,
		queuedAfter:  queuedAfter,
		pollInterval: pollInterval,
	}
}

func (r *StuckDocumentReaper) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		if err := r.reapOnce(ctx); err != nil && ctx.Err() == nil {
			// Falha transitoria de banco nao derruba o worker.
			log.Printf("stuck document reaper: %v", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *StuckDocumentReaper) reapOnce(ctx context.Context) error {
	errProcessing := r.reapProcessing(ctx)
	errQueued := r.requeueQueued(ctx)
	// Uma falha num dos dois nao impede o outro de rodar neste ciclo.
	return errors.Join(errProcessing, errQueued)
}

func (r *StuckDocumentReaper) requeueQueued(ctx context.Context) error {
	if r.queuedAfter <= 0 {
		return nil
	}

	cutoff := time.Now().UTC().Add(-r.queuedAfter)
	requeued, err := r.repo.RequeueStuckQueuedDocuments(ctx, cutoff)
	if err != nil {
		return err
	}
	if requeued > 0 {
		log.Printf("stuck document reaper: %d documento(s) em QUEUED com evento PUBLISHED ha mais de %s ganharam um evento PENDING novo no outbox", requeued, r.queuedAfter)
	}
	return nil
}

func (r *StuckDocumentReaper) reapProcessing(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-r.stuckAfter)

	reaped, err := r.repo.ReapStuckProcessingDocuments(ctx, cutoff)
	if err != nil {
		return err
	}

	if reaped > 0 {
		// So loga quando ha algo: numero crescente aqui = worker morrendo.
		log.Printf("stuck document reaper: %d documento(s) presos em PROCESSING ha mais de %s foram marcados como FAILED", reaped, r.stuckAfter)
	}

	return nil
}
