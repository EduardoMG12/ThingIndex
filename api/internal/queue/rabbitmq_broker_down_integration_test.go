//go:build integration

package queue

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestConsumerDistingueQuedaDoBrokerDeEncerramentoLimpo prova que o consumidor
// nao confunde "o RabbitMQ caiu" com "o processo esta encerrando".
//
// O bug original: quando o broker fechava o canal de entregas, o laco devolvia
// `nil` -- exatamente o mesmo valor do encerramento limpo. Como
// cmd/indexer/main.go so chama log.Fatalf se o erro for diferente de nil, o
// processo terminava com codigo 0; para o Docker isso e uma parada
// bem-sucedida e `restart: unless-stopped` nao reinicia nada. Resultado: o
// worker sumia sem alarme e todo upload ficava preso em QUEUED.
//
// Verificado RED -> GREEN: com `return nil` no lugar de
// ErrDeliveriesChannelClosed, este teste falha em "esperava
// ErrDeliveriesChannelClosed".
func TestConsumerDistingueQuedaDoBrokerDeEncerramentoLimpo(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	consumer := newIntegrationConsumer(t, cfg)

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.ConsumeDocumentIndexRequested(context.Background(), nil)
	}()

	// Deixa o consumidor entrar no laco antes de derrubar o canal.
	time.Sleep(500 * time.Millisecond)

	// Simula a queda do broker fechando o canal por baixo do consumidor.
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

// TestConsumerEncerraLimpoQuandoContextoECancelado é o controle do teste
// acima: no encerramento legitimo (SIGTERM -> contexto cancelado) o erro
// precisa continuar sendo context.Canceled, que main.go trata como saida
// normal. Sem este controle, "sempre devolver erro" passaria no teste anterior
// e transformaria todo deploy num falso alarme.
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
