package queue

import (
	"context"
	"errors"
	"testing"

	"thing-index/internal/domain"
)

type consumidorComCallbacksFalso struct {
	callbacks      DocumentIndexFailureCallbacks
	callbacksNoUso bool
	fechado        bool
}

func (c *consumidorComCallbacksFalso) SetFailureCallbacks(cb DocumentIndexFailureCallbacks) {
	c.callbacks = cb
}

func (c *consumidorComCallbacksFalso) ConsumeDocumentIndexRequested(_ context.Context, _ func(context.Context, domain.DocumentIndexRequestedEvent) error) error {
	c.callbacksNoUso = c.callbacks.OnRetryScheduled != nil &&
		c.callbacks.OnMovedToDLQ != nil &&
		c.callbacks.OnPermanentError != nil
	return ErrDeliveriesChannelClosed
}

func (c *consumidorComCallbacksFalso) Close() error {
	c.fechado = true
	return nil
}

// Regressao P1-1: as callbacks de falha iam para um consumidor ocioso do
// bootstrap, e o consumidor que o supervisor recriava consumia sem elas.
func TestConsumoComCallbacksInstalaCallbacksEmCadaConsumidorRecriado(t *testing.T) {
	t.Parallel()

	noop := func(context.Context, domain.DocumentIndexRequestedEvent, DocumentIndexFailureDisposition) {}
	callbacks := DocumentIndexFailureCallbacks{OnRetryScheduled: noop, OnMovedToDLQ: noop, OnPermanentError: noop}

	var criados []*consumidorComCallbacksFalso
	consumir := ConsumoComCallbacks(func() (ConsumidorDeIndexacao, error) {
		c := &consumidorComCallbacksFalso{}
		criados = append(criados, c)
		return c, nil
	}, callbacks, func(context.Context, domain.DocumentIndexRequestedEvent) error { return nil })

	for i := 0; i < 2; i++ {
		if err := consumir(context.Background()); !errors.Is(err, ErrDeliveriesChannelClosed) {
			t.Fatalf("erro do consumidor deveria subir intacto, veio %v", err)
		}
	}

	if len(criados) != 2 {
		t.Fatalf("esperava um consumidor novo por tentativa, vieram %d", len(criados))
	}
	for i, c := range criados {
		if !c.callbacksNoUso {
			t.Fatalf("consumidor %d consumiu sem as callbacks de falha", i)
		}
		if !c.fechado {
			t.Fatalf("consumidor %d nao foi fechado", i)
		}
	}
}

func TestConsumoComCallbacksDevolveErroDeCriacao(t *testing.T) {
	t.Parallel()

	falha := errors.New("dial rabbitmq: connection refused")
	consumir := ConsumoComCallbacks(func() (ConsumidorDeIndexacao, error) {
		return nil, falha
	}, DocumentIndexFailureCallbacks{}, nil)

	if err := consumir(context.Background()); !errors.Is(err, falha) {
		t.Fatalf("esperava o erro de criacao, veio %v", err)
	}
}

func TestRabbitMQConsumerSatisfazConsumidorDeIndexacao(t *testing.T) {
	t.Parallel()

	var _ ConsumidorDeIndexacao = (*RabbitMQConsumer)(nil)
}
