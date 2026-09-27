//go:build integration

package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"thing-index/internal/domain"
)

// TestPublisherRecusaMensagemSemRota prova que publicar para uma routing key
// sem fila vinculada devolve erro, em vez de nil.
//
// Antes (mandatory=false, sem publisher confirms) o broker descartava a
// mensagem em silencio, PublishDocumentIndexRequested devolvia nil e o outbox
// marcava o evento PUBLISHED: o documento ficava QUEUED para sempre.
//
// Verificado RED -> GREEN: com o PublishWithContext antigo este teste falha em
// "esperava ErrMensagemSemRota, veio <nil>".
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

	// Depois de uma devolucao, a proxima publicacao com rota valida precisa
	// voltar a funcionar: o retorno antigo nao pode contaminar a seguinte.
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

// TestPublisherReconectadoMantemConfirmacao garante que a reconexao religa o
// modo de confirmacao e o aviso de devolucao: um canal novo sem Confirm
// voltaria a aceitar mensagem sem rota como sucesso.
func TestPublisherReconectadoMantemConfirmacao(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	publisher := newIntegrationPublisher(t, cfg)
	defer publisher.Close()

	// Simula a queda: fecha o canal por baixo do publicador. A proxima
	// publicacao falha com "channel/connection is not open" e reconecta.
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
