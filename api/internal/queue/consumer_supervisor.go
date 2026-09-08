package queue

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
)

// ConsumerSupervisor reexecuta o consumidor quando o RabbitMQ fecha o canal de
// entregas, em vez de deixar o processo morrer.
//
// MOTIVO: uma queda do broker derrubava o worker inteiro. O main chamava
// log.Fatalf e contava com o supervisor do Docker (restart: unless-stopped).
// Isso funciona em producao, mas tem dois problemas.
//
// O primeiro apareceu num teste real de queda: em desenvolvimento o air segura
// o processo, entao o container continuava "running" com o consumidor morto.
// Os eventos ficaram PENDING no outbox e dois documentos travaram em QUEUED
// indefinidamente -- pior que falhar, porque nada indicava o problema.
//
// O segundo vale tambem em producao: o mesmo binario hospeda o dispatcher de
// outbox e o reaper de documentos presos. Derrubar tudo por uma queda de broker
// interrompe dois servicos que teriam trabalho util a fazer enquanto o
// RabbitMQ volta.
type ConsumerSupervisor struct {
	consumir func(context.Context) error
	espera   time.Duration
}

// NewConsumerSupervisor cria o supervisor.
//
// A espera entre tentativas e obrigatoria: sem ela, um broker fora do ar vira
// laco quente consumindo CPU e enchendo o log.
func NewConsumerSupervisor(consumir func(context.Context) error, espera time.Duration) *ConsumerSupervisor {
	if espera <= 0 {
		espera = time.Second
	}
	return &ConsumerSupervisor{consumir: consumir, espera: espera}
}

// erroDeConexaoTransitoria diz se vale tentar de novo.
//
// ErrDeliveriesChannelClosed cobre so o primeiro momento da queda: o broker
// fecha o canal de entregas. A tentativa SEGUINTE falha de outro jeito -- erro
// de dial enquanto o RabbitMQ nao voltou, ou "channel/connection is not open"
// se a conexao morta for reaproveitada.
//
// Tratar apenas o canal fechado como transitorio fez o supervisor desistir na
// segunda tentativa e o processo morrer, que era justamente o que a supervisao
// deveria evitar.
//
// A comparacao e por texto porque a biblioteca amqp devolve *amqp.Error com
// codigo numerico e os erros de rede vem do pacote net -- um errors.As para
// cada tipo acoplaria este arquivo as duas bibliotecas sem ganhar precisao.
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

// Run executa o consumidor ate o contexto ser cancelado.
//
// Reconecta apenas em falha de conexao. Qualquer outro erro sobe: tentar de
// novo diante de credencial invalida ou fila inexistente transformaria um bug
// de configuracao em laco infinito silencioso, e o operador nunca veria a causa.
func (s *ConsumerSupervisor) Run(ctx context.Context) error {
	for {
		err := s.consumir(ctx)

		switch {
		case err == nil:
			return nil
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return err
		case !erroDeConexaoTransitoria(err):
			// Erro que nova tentativa nao resolve.
			return err
		}

		// Antes de esperar, confere se ja fomos cancelados: sem isso o
		// encerramento gracioso fica preso ate o fim do intervalo.
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
