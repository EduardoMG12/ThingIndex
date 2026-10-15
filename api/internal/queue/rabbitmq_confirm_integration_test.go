//go:build integration

package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"thing-index/internal/domain"
)

// Sem mandatory e confirms o broker descartava a mensagem sem rota e o
// outbox marcava PUBLISHED.
func TestPublisherRecusaMensagemSemRota(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	publisher := newIntegrationPublisher(t, cfg)
	defer publisher.Close()

	publisher.routingKey = cfg.DocumentIndexRoutingKey + ".sem-fila"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := publisher.PublishDocumentIndexRequested(ctx, domain.DocumentIndexRequestedEvent{DocumentID: 501, UserID: 7})
	if !errors.Is(err, ErrMensagemSemRota) {
		t.Fatalf("esperava ErrMensagemSemRota, veio %v", err)
	}

	// O retorno anterior nao pode contaminar a proxima publicacao.
	publisher.routingKey = cfg.DocumentIndexRoutingKey
	expected := domain.DocumentIndexRequestedEvent{DocumentID: 502, UserID: 7}
	if err := publisher.PublishDocumentIndexRequested(ctx, expected); err != nil {
		t.Fatalf("publicacao com rota valida falhou depois de uma devolucao: %v", err)
	}

	channel := newInspectionRabbitChannel(t, cfg)
	defer channel.Close()
	delivery := waitForRabbitMessage(t, channel, cfg.DocumentIndexQueue, 5*time.Second)
	assertRabbitEventEquals(t, delivery.Body, expected)
}

// A reconexao precisa religar confirm e NotifyReturn.
func TestPublisherReconectadoMantemConfirmacao(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	publisher := newIntegrationPublisher(t, cfg)
	defer publisher.Close()

	// Fecha o canal por baixo; a proxima publicacao falha e reconecta.
	if err := publisher.channel.Close(); err != nil {
		t.Fatalf("fechar canal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := publisher.PublishDocumentIndexRequested(ctx, domain.DocumentIndexRequestedEvent{DocumentID: 601, UserID: 8}); err != nil {
		t.Fatalf("publicacao apos reconexao falhou: %v", err)
	}

	publisher.routingKey = cfg.DocumentIndexRoutingKey + ".sem-fila"
	err := publisher.PublishDocumentIndexRequested(ctx, domain.DocumentIndexRequestedEvent{DocumentID: 602, UserID: 8})
	if !errors.Is(err, ErrMensagemSemRota) {
		t.Fatalf("canal reconectado aceitou mensagem sem rota: %v", err)
	}
}
