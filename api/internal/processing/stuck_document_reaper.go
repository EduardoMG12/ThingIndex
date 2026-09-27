package processing

import (
	"context"
	"errors"
	"log"
	"time"

	"thing-index/internal/domain"
)

// StuckDocumentReaper libera documentos que ficaram presos em PROCESSING.
//
// Ha dois modos de falha distintos, e cada um precisa da sua defesa:
//
//   - O worker esta VIVO mas demora demais (PDF gigante, LLM que nao responde).
//     Quem resolve isso e o timeout por mensagem no consumidor, que interrompe
//     o handler e devolve a mensagem para o fluxo de retry.
//   - O worker MORREU no meio do trabalho (OOM kill, SIGKILL, queda do no).
//     Aqui nenhum codigo do processo roda: ninguem marca FAILED, ninguem
//     agenda retry. O documento fica PROCESSING para sempre.
//
// Este reaper existe para o segundo caso. Sem ele, o usuario ve "processando"
// indefinidamente -- sem mensagem de erro e sem forma de reenviar.
//
// No mesmo laco ele reconcilia documentos presos em QUEUED cujo evento ja foi
// PUBLISHED ha mais de queuedAfter: a mensagem se perdeu depois do broker (fila
// apagada, purge, volume perdido). Esses ganham um evento PENDING novo no
// outbox, em vez de FAILED: nada falhou no documento, so a entrega.
type StuckDocumentReaper struct {
	repo domain.DocumentProcessingRepository
	// Quanto tempo em PROCESSING caracteriza documento preso. Precisa ser
	// folgadamente maior que o timeout por mensagem, senao o reaper marca
	// FAILED um documento que o worker ainda esta processando legitimamente.
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
			// Loga e segue: uma falha transitoria de banco nao pode derrubar o
			// worker inteiro, que continua consumindo a fila normalmente.
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
		// Silencio quando nao ha nada a fazer, log quando ha: um numero
		// crescente aqui e sinal de worker morrendo com frequencia, e essa
		// informacao se perde se o log for sempre igual.
		log.Printf("stuck document reaper: %d documento(s) presos em PROCESSING ha mais de %s foram marcados como FAILED", reaped, r.stuckAfter)
	}

	return nil
}
