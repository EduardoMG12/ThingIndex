package queue

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type consumidorFalso struct {
	mu         sync.Mutex
	chamadas   int32
	erros      []error
	bloqueante bool
}

func (c *consumidorFalso) Consume(ctx context.Context) error {
	n := atomic.AddInt32(&c.chamadas, 1)

	c.mu.Lock()
	var err error
	if int(n) <= len(c.erros) {
		err = c.erros[n-1]
	}
	c.mu.Unlock()

	if err != nil {
		return err
	}

	if c.bloqueante {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (c *consumidorFalso) total() int32 {
	return atomic.LoadInt32(&c.chamadas)
}

func TestSupervisorReconectaAposQuedaDoBroker(t *testing.T) {
	falso := &consumidorFalso{
		erros:      []error{ErrDeliveriesChannelClosed, ErrDeliveriesChannelClosed},
		bloqueante: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sup := NewConsumerSupervisor(falso.Consume, 10*time.Millisecond)

	pronto := make(chan error, 1)
	go func() { pronto <- sup.Run(ctx) }()

	prazo := time.After(2 * time.Second)
	for falso.total() < 3 {
		select {
		case <-prazo:
			t.Fatalf("supervisor nao reconectou: %d tentativa(s)", falso.total())
		case err := <-pronto:
			t.Fatalf("supervisor desistiu cedo demais: %v", err)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()
	select {
	case err := <-pronto:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("erro inesperado ao encerrar: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("supervisor nao respeitou o cancelamento do contexto")
	}
}

func TestSupervisorNaoReconectaEmErroDeProgramacao(t *testing.T) {
	erroFatal := errors.New("credencial invalida")
	falso := &consumidorFalso{erros: []error{erroFatal}}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sup := NewConsumerSupervisor(falso.Consume, 10*time.Millisecond)
	err := sup.Run(ctx)

	if !errors.Is(err, erroFatal) {
		t.Fatalf("esperava o erro original de volta, veio: %v", err)
	}
	if falso.total() != 1 {
		t.Fatalf("supervisor tentou de novo em erro nao transitorio: %d chamadas", falso.total())
	}
}

func TestSupervisorRespeitaCancelamentoDuranteAEspera(t *testing.T) {
	falso := &consumidorFalso{erros: []error{ErrDeliveriesChannelClosed}}

	ctx, cancel := context.WithCancel(context.Background())
	sup := NewConsumerSupervisor(falso.Consume, 5*time.Second)

	pronto := make(chan error, 1)
	go func() { pronto <- sup.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-pronto:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("erro inesperado: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("supervisor ficou preso na espera de 5s em vez de sair no cancelamento")
	}
}

func TestSupervisorEsperaEntreTentativas(t *testing.T) {
	falso := &consumidorFalso{
		erros: []error{
			ErrDeliveriesChannelClosed,
			ErrDeliveriesChannelClosed,
			ErrDeliveriesChannelClosed,
		},
		bloqueante: true,
	}

	const intervalo = 60 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	inicio := time.Now()
	sup := NewConsumerSupervisor(falso.Consume, intervalo)

	pronto := make(chan error, 1)
	go func() { pronto <- sup.Run(ctx) }()

	for falso.total() < 4 {
		if time.Since(inicio) > 1500*time.Millisecond {
			t.Fatalf("nao chegou a quarta tentativa: %d", falso.total())
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Tres esperas entre quatro tentativas.
	if decorrido := time.Since(inicio); decorrido < 3*intervalo {
		t.Fatalf("reconectou rapido demais (%v): sem espera, um broker fora do ar vira laco quente",
			decorrido)
	}

	cancel()
	<-pronto
}

// Apos a queda, a tentativa seguinte devolve erro de dial ou
// "channel/connection is not open", e isso tambem tem de reconectar.
func TestSupervisorReconectaEmFalhaDeConexao(t *testing.T) {
	casos := []struct {
		nome string
		erro error
	}{
		{"canal fechado pelo broker", ErrDeliveriesChannelClosed},
		{"conexao nao esta aberta", errors.New(`consume queue: Exception (504) Reason: "channel/connection is not open"`)},
		{"broker inalcancavel", errors.New("dial rabbitmq: dial tcp 172.29.0.5:5672: connect: connection refused")},
		{"dns do broker sumiu", errors.New("dial rabbitmq: lookup rabbitmq: no such host")},
		// Com o contêiner parado, o DNS interno do Docker responde SERVFAIL, e o
		// erro chega assim; foi o que reiniciou o indexer no teste de caos.
		{"dns do docker sem resposta", fmt.Errorf("dial rabbitmq: %w", &net.OpError{
			Op: "dial", Net: "tcp",
			Err: &net.DNSError{Err: "server misbehaving", Name: "rabbitmq", Server: "127.0.0.11:53"},
		})},
		{"qualquer erro de rede no dial", fmt.Errorf("dial rabbitmq: %w", &net.OpError{
			Op: "dial", Net: "tcp", Err: errors.New("network is unreachable"),
		})},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			falso := &consumidorFalso{erros: []error{caso.erro}, bloqueante: true}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			sup := NewConsumerSupervisor(falso.Consume, 10*time.Millisecond)
			pronto := make(chan error, 1)
			go func() { pronto <- sup.Run(ctx) }()

			prazo := time.After(time.Second)
			for falso.total() < 2 {
				select {
				case err := <-pronto:
					t.Fatalf("supervisor desistiu em erro transitorio (%v): %v", caso.erro, err)
				case <-prazo:
					t.Fatalf("nao houve segunda tentativa apos %v", caso.erro)
				default:
					time.Sleep(5 * time.Millisecond)
				}
			}

			cancel()
			<-pronto
		})
	}
}
