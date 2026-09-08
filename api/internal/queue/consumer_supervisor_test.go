package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Uma queda do RabbitMQ derrubava o worker inteiro: o consumidor devolvia
// ErrDeliveriesChannelClosed e o main chamava log.Fatalf, contando com o
// supervisor do Docker para reiniciar.
//
// Em producao (restart: unless-stopped) isso funciona. Em desenvolvimento o air
// segura o processo, e o resultado observado foi pior do que travar: o
// container continuava "running", o consumidor estava morto, e os eventos
// ficavam PENDING no outbox sem ninguem perceber. Dois documentos ficaram
// QUEUED indefinidamente num teste real de queda.
//
// Mesmo em producao, derrubar o processo por uma queda de broker desperdica:
// perde o dispatcher de outbox e o reaper que rodam no mesmo binario, e ambos
// tem trabalho util a fazer enquanto o broker volta.

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

	// Depois das duas quedas o consumidor precisa estar rodando de novo.
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

// TestSupervisorNaoReconectaEmErroDeProgramacao: reconectar indefinidamente diante
// de um erro que nao e de conexao transformaria bug em laco infinito silencioso.
// Só falha de canal justifica nova tentativa.
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

// TestSupervisorRespeitaCancelamentoDuranteAEspera: o encerramento gracioso nao
// pode ficar preso no intervalo entre tentativas.
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

// TestSupervisorEsperaEntreTentativas evita martelar um broker que esta voltando.
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

// TestSupervisorReconectaEmFalhaDeConexao cobre o caso que quebrou a primeira
// versao desta correcao em producao de desenvolvimento.
//
// Quando o broker cai, a tentativa seguinte nao devolve
// ErrDeliveriesChannelClosed: devolve o erro de dial ou
// "Exception (504) channel/connection is not open". Tratar so o canal fechado
// como transitorio fazia o supervisor desistir na SEGUNDA tentativa e o
// processo morrer -- exatamente o que se queria evitar.
func TestSupervisorReconectaEmFalhaDeConexao(t *testing.T) {
	casos := []struct {
		nome string
		erro error
	}{
		{"canal fechado pelo broker", ErrDeliveriesChannelClosed},
		{"conexao nao esta aberta", errors.New(`consume queue: Exception (504) Reason: "channel/connection is not open"`)},
		{"broker inalcancavel", errors.New("dial rabbitmq: dial tcp 172.29.0.5:5672: connect: connection refused")},
		{"dns do broker sumiu", errors.New("dial rabbitmq: lookup rabbitmq: no such host")},
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
