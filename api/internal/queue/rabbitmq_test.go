package queue

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"thing-index/internal/domain"
)

func TestClassifyDocumentIndexFailureMarksPermanentErrors(t *testing.T) {
	t.Parallel()

	cases := []error{
		fmtError(domain.ErrDocumentSourceMissing),
		fmtError(domain.ErrDocumentParsingFailed),
	}

	for _, err := range cases {
		err := err
		t.Run(err.Error(), func(t *testing.T) {
			t.Parallel()

			disposition := classifyDocumentIndexFailure(err, 0, 3, 30*time.Second)
			if disposition.Action != documentIndexFailurePermanent {
				t.Fatalf("expected permanent failure action, got %#v", disposition)
			}
			if disposition.NextRetryAttempt != 0 {
				t.Fatalf("expected no retry attempt for permanent failure, got %#v", disposition)
			}
		})
	}
}

func TestClassifyDocumentIndexFailureSchedulesRetryWhenRetriesRemain(t *testing.T) {
	t.Parallel()

	disposition := classifyDocumentIndexFailure(errors.New("temporary network failure"), 1, 3, 45*time.Second)
	if disposition.Action != documentIndexFailureRetry {
		t.Fatalf("expected retry action, got %#v", disposition)
	}
	if disposition.NextRetryAttempt != 2 {
		t.Fatalf("expected next retry attempt 2, got %#v", disposition)
	}
	if disposition.RetryDelay != 45*time.Second {
		t.Fatalf("expected retry delay to be preserved, got %#v", disposition)
	}
}

func TestClassifyDocumentIndexFailureMovesToDLQWhenRetriesAreExhausted(t *testing.T) {
	t.Parallel()

	disposition := classifyDocumentIndexFailure(errors.New("persistent infra failure"), 3, 3, 30*time.Second)
	if disposition.Action != documentIndexFailureDLQ {
		t.Fatalf("expected dlq action, got %#v", disposition)
	}
	if disposition.NextRetryAttempt != 0 {
		t.Fatalf("expected no next retry attempt, got %#v", disposition)
	}
}

func fmtError(base error) error {
	return fmt.Errorf("%w: wrapped", base)
}

func TestHandleDeliveryInterrompeMensagemQueEstouraOPrazo(t *testing.T) {
	t.Parallel()

	consumer := &RabbitMQConsumer{handlerTimeout: 40 * time.Millisecond}

	handlerRetornou := make(chan struct{})
	err := consumer.handleDelivery(context.Background(), func(ctx context.Context, _ domain.DocumentIndexRequestedEvent) error {
		<-ctx.Done()
		close(handlerRetornou)
		return ctx.Err()
	}, domain.DocumentIndexRequestedEvent{DocumentID: 1, UserID: 1})

	select {
	case <-handlerRetornou:
	case <-time.After(2 * time.Second):
		t.Fatal("o handler nao foi cancelado pelo deadline; a fila ficaria travada")
	}

	if err == nil {
		t.Fatal("esperava erro de timeout, veio nil")
	}

	// O timeout tem de ser transiente; se fosse permanente iria direto para a DLQ.
	if !errors.Is(err, domain.ErrDocumentIndexing) {
		t.Fatalf("o timeout deveria ser ErrDocumentIndexing (transiente), veio %v", err)
	}
	disposition := classifyDocumentIndexFailure(err, 0, 5, time.Second)
	if disposition.Action != documentIndexFailureRetry {
		t.Fatalf("timeout deveria virar retry, virou %q", disposition.Action)
	}
}

func TestHandleDeliveryNaoInterfereQuandoOHandlerTerminaATempo(t *testing.T) {
	t.Parallel()

	consumer := &RabbitMQConsumer{handlerTimeout: 5 * time.Second}

	chamadas := 0
	err := consumer.handleDelivery(context.Background(), func(context.Context, domain.DocumentIndexRequestedEvent) error {
		chamadas++
		return nil
	}, domain.DocumentIndexRequestedEvent{DocumentID: 2, UserID: 1})

	if err != nil {
		t.Fatalf("handler rapido nao deveria falhar, veio %v", err)
	}
	if chamadas != 1 {
		t.Fatalf("esperava 1 chamada, veio %d", chamadas)
	}
}

// No shutdown o ctx do handler tambem cancela; isso nao pode contar como timeout.
func TestHandleDeliveryNaoConfundeShutdownComTimeoutDaMensagem(t *testing.T) {
	t.Parallel()

	consumer := &RabbitMQConsumer{handlerTimeout: 10 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())

	err := consumer.handleDelivery(ctx, func(handlerCtx context.Context, _ domain.DocumentIndexRequestedEvent) error {
		cancel() // simula SIGTERM no meio do processamento
		<-handlerCtx.Done()
		return handlerCtx.Err()
	}, domain.DocumentIndexRequestedEvent{DocumentID: 3, UserID: 1})

	if errors.Is(err, domain.ErrDocumentIndexing) {
		t.Fatalf("shutdown nao pode ser reportado como timeout da mensagem, veio %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled vindo do handler, veio %v", err)
	}
}

func TestHandleDeliverySemTimeoutConfiguradoExecutaDireto(t *testing.T) {
	t.Parallel()

	consumer := &RabbitMQConsumer{handlerTimeout: 0}

	var recebido context.Context
	err := consumer.handleDelivery(context.Background(), func(ctx context.Context, _ domain.DocumentIndexRequestedEvent) error {
		recebido = ctx
		return nil
	}, domain.DocumentIndexRequestedEvent{DocumentID: 4, UserID: 1})

	if err != nil {
		t.Fatalf("esperava sucesso, veio %v", err)
	}
	if _, temDeadline := recebido.Deadline(); temDeadline {
		t.Fatal("com handlerTimeout zero o contexto nao deveria ganhar deadline")
	}
}
