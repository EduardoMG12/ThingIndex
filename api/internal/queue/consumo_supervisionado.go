package queue

import (
	"context"

	"thing-index/internal/domain"
)

// ConsumidorDeIndexacao e o recorte do RabbitMQConsumer que o supervisor usa.
// Existe para que o wiring do indexer seja testavel sem broker.
type ConsumidorDeIndexacao interface {
	SetFailureCallbacks(DocumentIndexFailureCallbacks)
	ConsumeDocumentIndexRequested(context.Context, func(context.Context, domain.DocumentIndexRequestedEvent) error) error
	Close() error
}

// ConsumoComCallbacks monta a funcao que o ConsumerSupervisor reexecuta: cria
// um consumidor novo, instala as callbacks de falha NELE e consome.
//
// MOTIVO: as callbacks eram instaladas num consumidor criado no bootstrap,
// enquanto quem consumia de fato era outro, criado dentro do supervisor e sem
// callbacks. MarkRetryScheduled, MarkMovedToDLQ e MarkPermanentFailure nunca
// rodavam: nenhuma tentativa de processamento era gravada e o status nunca
// virava RETRY_SCHEDULED nem MOVED_TO_DLQ. Amarrar as callbacks aqui garante
// que todo consumidor recriado pelo supervisor as recebe.
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
