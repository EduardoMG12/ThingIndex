package queue

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
)

// ConsumerSupervisor reexecuta o consumidor quando o broker cai, em vez de
// derrubar o processo (que tambem hospeda o dispatcher do outbox e o reaper).
type ConsumerSupervisor struct {
	consumir func(context.Context) error
	espera   time.Duration
}

// A espera entre tentativas evita laco quente com o broker fora do ar.
func NewConsumerSupervisor(consumir func(context.Context) error, espera time.Duration) *ConsumerSupervisor {
	if espera <= 0 {
		espera = time.Second
	}
	return &ConsumerSupervisor{consumir: consumir, espera: espera}
}

// Depois da queda, a tentativa seguinte falha com erro de dial ou
// "channel/connection is not open", nao com ErrDeliveriesChannelClosed.
// Comparacao por texto porque os erros vem do amqp e do pacote net.
func erroDeConexaoTransitoria(err error) bool {
	if errors.Is(err, ErrDeliveriesChannelClosed) {
		return true
	}

	texto := strings.ToLower(err.Error())
	sinais := []string{
		"channel/connection is not open",
		"connection refused",
		"no such host",
		"connection reset",
		"broken pipe",
		"unexpected eof",
		"i/o timeout",
	}

	for _, sinal := range sinais {
		if strings.Contains(texto, sinal) {
			return true
		}
	}
	return false
}

// Run so reconecta em falha de conexao; qualquer outro erro sobe.
func (s *ConsumerSupervisor) Run(ctx context.Context) error {
	for {
		err := s.consumir(ctx)

		switch {
		case err == nil:
			return nil
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return err
		case !erroDeConexaoTransitoria(err):
			return err
		}

		// Sem isso o shutdown espera o intervalo inteiro.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Printf("consumidor perdeu a conexao (%v), reconectando em %v", err, s.espera)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.espera):
		}
	}
}
