package queue

import (
	"context"

	"thing-index/internal/domain"
)

// ConsumidorDeIndexacao e o recorte do RabbitMQConsumer usado pelo supervisor.
type ConsumidorDeIndexacao interface {
	SetFailureCallbacks(DocumentIndexFailureCallbacks)
	ConsumeDocumentIndexRequested(context.Context, func(context.Context, domain.DocumentIndexRequestedEvent) error) error
	Close() error
}

// ConsumoComCallbacks monta a funcao que o ConsumerSupervisor reexecuta. As
// callbacks de falha sao instaladas em cada consumidor recriado; instaladas
// so no bootstrap, o consumidor real rodava sem elas.
func ConsumoComCallbacks(
	novoConsumidor func() (ConsumidorDeIndexacao, error),
	callbacks DocumentIndexFailureCallbacks,
	handler func(context.Context, domain.DocumentIndexRequestedEvent) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		consumidor, err := novoConsumidor()
		if err != nil {
			return err
		}
		defer consumidor.Close()

		consumidor.SetFailureCallbacks(callbacks)
		return consumidor.ConsumeDocumentIndexRequested(ctx, handler)
	}
}
