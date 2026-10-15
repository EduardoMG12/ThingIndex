//go:build integration

package queue

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Queda do broker tem de devolver erro, nao nil: com nil o processo sai com
// codigo 0 e o Docker nao reinicia o worker.
func TestConsumerDistingueQuedaDoBrokerDeEncerramentoLimpo(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	consumer := newIntegrationConsumer(t, cfg)

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.ConsumeDocumentIndexRequested(context.Background(), nil)
	}()

	// Deixa o consumidor entrar no laco antes de derrubar o canal.
	time.Sleep(500 * time.Millisecond)

	if err := consumer.channel.Close(); err != nil {
		t.Fatalf("fechar canal para simular queda do broker: %v", err)
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrDeliveriesChannelClosed) {
			t.Fatalf("esperava ErrDeliveriesChannelClosed para que o supervisor reinicie o worker, veio: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("o consumidor nao retornou apos a queda do canal")
	}
}

// Controle do teste acima: no SIGTERM o erro continua sendo context.Canceled.
func TestConsumerEncerraLimpoQuandoContextoECancelado(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	consumer := newIntegrationConsumer(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.ConsumeDocumentIndexRequested(ctx, nil)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("encerramento limpo deveria devolver context.Canceled, veio: %v", err)
		}
		if errors.Is(err, ErrDeliveriesChannelClosed) {
			t.Fatal("encerramento limpo nao pode ser confundido com queda do broker")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("o consumidor nao retornou apos o cancelamento do contexto")
	}
}
