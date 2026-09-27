package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

const retryHeader = "x-retry-count"

type DocumentIndexFailureAction string

const (
	documentIndexFailureRetry     DocumentIndexFailureAction = "retry"
	documentIndexFailureDLQ       DocumentIndexFailureAction = "dlq"
	documentIndexFailurePermanent DocumentIndexFailureAction = "permanent"
)

type DocumentIndexFailureDisposition struct {
	Action           DocumentIndexFailureAction
	RetryCount       int
	NextRetryAttempt int
	MaxRetries       int
	RetryDelay       time.Duration
	Err              error
}

type DocumentIndexFailureCallbacks struct {
	OnRetryScheduled func(context.Context, domain.DocumentIndexRequestedEvent, DocumentIndexFailureDisposition)
	OnMovedToDLQ     func(context.Context, domain.DocumentIndexRequestedEvent, DocumentIndexFailureDisposition)
	OnPermanentError func(context.Context, domain.DocumentIndexRequestedEvent, DocumentIndexFailureDisposition)
}

type RabbitMQPublisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string

	// Guardada para reabrir a conexao quando o broker cai e volta. Sem isso,
	// todo Publish apos uma queda falhava com "channel/connection is not open"
	// para sempre, e os eventos do outbox iam para FAILED mesmo com o RabbitMQ
	// ja no ar.
	cfg *config.Config
	// Protege a troca de conn/channel: o dispatcher publica em lote e a
	// reconexao nao pode acontecer no meio de outra publicacao.
	mu sync.Mutex

	// retornos recebe as mensagens que o broker devolveu por falta de rota
	// (publicacao com mandatory=true). E trocado junto com o canal.
	retornos chan amqp.Return
	// publicando serializa publicar+aguardar confirmacao: o basic.return de
	// uma mensagem sem rota chega antes do basic.ack dela, e so com uma
	// publicacao em voo por vez da para atribuir o retorno a mensagem certa.
	publicando sync.Mutex
	sequencia  uint64
}

// ErrMensagemSemRota: o broker aceitou a publicacao mas nao havia fila
// vinculada a routing key (binding apagado, fila removida). Sem mandatory=true
// o RabbitMQ descartava a mensagem em silencio e o outbox a marcava PUBLISHED.
var ErrMensagemSemRota = errors.New("rabbitmq devolveu a mensagem: nenhuma fila vinculada a routing key")

// ErrPublicacaoNaoConfirmada: o broker respondeu basic.nack, ou o canal fechou
// antes da confirmacao. Em ambos os casos nao ha garantia de que a mensagem foi
// persistida, entao o evento do outbox nao pode ser marcado PUBLISHED.
var ErrPublicacaoNaoConfirmada = errors.New("rabbitmq nao confirmou a publicacao")

// publisherConfirmTimeout limita a espera pelo ack quando o contexto do chamador
// nao tem prazo. Sem isso, um broker que aceita o frame e nunca confirma
// prenderia o despachante do outbox para sempre.
const publisherConfirmTimeout = 30 * time.Second

type RabbitMQConsumer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string
	queue      string
	retryQueue string
	dlq        string
	retryDelay time.Duration
	maxRetries int
	// Prazo maximo de uma unica mensagem. O consumidor e sequencial, entao
	// sem isso um PDF gigante ou um LLM que nao responde trava a fila inteira
	// -- e o ctx do handler era o do processo, sem deadline nenhum.
	handlerTimeout time.Duration

	failureCallbacks DocumentIndexFailureCallbacks
}

func NewRabbitMQPublisher(cfg *config.Config) (*RabbitMQPublisher, error) {
	conn, channel, retornos, err := abrirConexaoPublicador(cfg)
	if err != nil {
		return nil, err
	}

	return &RabbitMQPublisher{
		cfg:        cfg,
		conn:       conn,
		channel:    channel,
		retornos:   retornos,
		exchange:   cfg.DocumentIndexExchange,
		routingKey: cfg.DocumentIndexRoutingKey,
	}, nil
}

// abrirConexaoPublicador abre conexao e canal do publicador, declara a
// topologia e liga o modo de confirmacao (publisher confirms) e o aviso de
// mensagem devolvida. Usado na criacao e na reconexao, para que o canal novo
// tenha exatamente as mesmas garantias do original.
func abrirConexaoPublicador(cfg *config.Config) (*amqp.Connection, *amqp.Channel, chan amqp.Return, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("create rabbitmq channel: %w", err)
	}

	if err := declareTopology(channel, cfg); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, nil, nil, err
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	// Com buffer: a biblioteca entrega o retorno de forma bloqueante na
	// goroutine de leitura, e um canal sem buffer travaria a conexao ate
	// alguem ler.
	retornos := channel.NotifyReturn(make(chan amqp.Return, 16))

	return conn, channel, retornos, nil
}

func NewRabbitMQConsumer(cfg *config.Config) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("create rabbitmq channel: %w", err)
	}

	if err := declareTopology(channel, cfg); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := channel.Qos(cfg.ConsumerPrefetchCount, 0, false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("configure rabbitmq qos: %w", err)
	}

	return &RabbitMQConsumer{
		conn:           conn,
		channel:        channel,
		exchange:       cfg.DocumentIndexExchange,
		routingKey:     cfg.DocumentIndexRoutingKey,
		queue:          cfg.DocumentIndexQueue,
		retryQueue:     cfg.DocumentIndexRetryQueue,
		dlq:            cfg.DocumentIndexDLQ,
		retryDelay:     cfg.DocumentIndexRetryDelay,
		maxRetries:     cfg.DocumentIndexMaxRetries,
		handlerTimeout: cfg.DocumentIndexHandlerTimeout,
	}, nil
}

func declareTopology(channel *amqp.Channel, cfg *config.Config) error {
	if err := channel.ExchangeDeclare(cfg.DocumentIndexExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    cfg.DocumentIndexExchange,
		"x-dead-letter-routing-key": cfg.DocumentIndexDLQ,
	}
	if _, err := channel.QueueDeclare(cfg.DocumentIndexQueue, true, false, false, false, mainArgs); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := channel.QueueBind(cfg.DocumentIndexQueue, cfg.DocumentIndexRoutingKey, cfg.DocumentIndexExchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	retryArgs := amqp.Table{
		"x-message-ttl":             int32(cfg.DocumentIndexRetryDelay / time.Millisecond),
		"x-dead-letter-exchange":    cfg.DocumentIndexExchange,
		"x-dead-letter-routing-key": cfg.DocumentIndexRoutingKey,
	}
	if _, err := channel.QueueDeclare(cfg.DocumentIndexRetryQueue, true, false, false, false, retryArgs); err != nil {
		return fmt.Errorf("declare retry queue: %w", err)
	}
	if err := channel.QueueBind(cfg.DocumentIndexRetryQueue, cfg.DocumentIndexRetryQueue, cfg.DocumentIndexExchange, false, nil); err != nil {
		return fmt.Errorf("bind retry queue: %w", err)
	}

	if _, err := channel.QueueDeclare(cfg.DocumentIndexDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := channel.QueueBind(cfg.DocumentIndexDLQ, cfg.DocumentIndexDLQ, cfg.DocumentIndexExchange, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	return nil
}

func (p *RabbitMQPublisher) PublishDocumentIndexRequested(ctx context.Context, event domain.DocumentIndexRequestedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	p.publicando.Lock()
	defer p.publicando.Unlock()

	return publicarComReconexao(ctx, func(ctx context.Context) error {
		p.mu.Lock()
		canal := p.channel
		retornos := p.retornos
		p.mu.Unlock()

		p.sequencia++
		messageID := fmt.Sprintf("doc-%d-%d", event.DocumentID, p.sequencia)

		return publicarConfirmado(ctx, canal, retornos, p.exchange, p.routingKey, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Type:         domain.OutboxEventDocumentIndexRequested,
			MessageId:    messageID,
			Headers: amqp.Table{
				retryHeader: int32(0),
			},
		})
	}, p.reconectar)
}

// publicarConfirmado publica com mandatory=true e so devolve nil depois do
// basic.ack do broker, sem retorno de "sem rota" para esta mensagem.
//
// Antes, PublishWithContext devolvia nil assim que o frame saia pelo socket:
// se o broker caisse antes de persistir, ou se nao houvesse fila vinculada, a
// mensagem sumia e o outbox a marcava PUBLISHED -- documento preso em QUEUED
// para sempre.
func publicarConfirmado(
	ctx context.Context,
	canal *amqp.Channel,
	retornos chan amqp.Return,
	exchange, routingKey string,
	msg amqp.Publishing,
) error {
	// Descarta retornos antigos (de uma publicacao que desistiu por timeout)
	// para nao atribui-los a esta mensagem.
	drenarRetornos(retornos)

	confirmacao, err := canal.PublishWithDeferredConfirmWithContext(ctx, exchange, routingKey, true, false, msg)
	if err != nil {
		return err
	}

	esperaCtx := ctx
	if _, temPrazo := ctx.Deadline(); !temPrazo {
		var cancel context.CancelFunc
		esperaCtx, cancel = context.WithTimeout(ctx, publisherConfirmTimeout)
		defer cancel()
	}

	ack, err := confirmacao.WaitContext(esperaCtx)
	if err != nil {
		return fmt.Errorf("%w: aguardando confirmacao: %w", ErrPublicacaoNaoConfirmada, err)
	}
	if !ack {
		return ErrPublicacaoNaoConfirmada
	}

	// O RabbitMQ envia o basic.return ANTES do basic.ack da mesma mensagem, e
	// a biblioteca entrega os dois na ordem em que chegam; com o ack em maos,
	// um eventual retorno ja esta no canal.
	for {
		select {
		case retorno, ok := <-retornos:
			if !ok {
				return nil
			}
			if retorno.MessageId == msg.MessageId {
				return fmt.Errorf("%w (exchange=%q routing_key=%q: %d %s)", ErrMensagemSemRota, retorno.Exchange, retorno.RoutingKey, retorno.ReplyCode, retorno.ReplyText)
			}
		default:
			return nil
		}
	}
}

func drenarRetornos(retornos chan amqp.Return) {
	for {
		select {
		case _, ok := <-retornos:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

func (p *RabbitMQPublisher) Close() error {
	if p == nil {
		return nil
	}
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (c *RabbitMQConsumer) SetFailureCallbacks(callbacks DocumentIndexFailureCallbacks) {
	c.failureCallbacks = callbacks
}

// ErrDeliveriesChannelClosed sinaliza que o RabbitMQ fechou o canal de
// entregas por conta propria -- queda do broker, restart, corte de rede ou
// channel.Close() remoto.
//
// Antes desta distincao o consumidor devolvia `nil` nesse caso, exatamente
// como no encerramento limpo por cancelamento de contexto. O efeito era grave
// e silencioso: `cmd/indexer/main.go` so chama log.Fatalf quando o erro nao e
// nil, entao o processo saia com codigo 0. Para o supervisor (Docker,
// Kubernetes, systemd) isso e uma parada bem-sucedida, e a politica
// `restart: unless-stopped` NAO reinicia o container. O worker sumia, a fila
// parava de ser consumida e nenhum documento era indexado -- sem alarme, sem
// log de erro e sem sintoma ate alguem reparar que os uploads ficam eternamente
// em QUEUED.
//
// Devolver um erro distinto faz o processo morrer com codigo diferente de zero
// e o supervisor reinicia, que e o comportamento correto para uma dependencia
// que caiu.
var ErrDeliveriesChannelClosed = errors.New("rabbitmq fechou o canal de entregas")

func (c *RabbitMQConsumer) ConsumeDocumentIndexRequested(ctx context.Context, handler func(context.Context, domain.DocumentIndexRequestedEvent) error) error {
	deliveries, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				// O canal so fecha por iniciativa do broker: encerramento
				// limpo chega por ctx.Done() no case acima.
				return ErrDeliveriesChannelClosed
			}

			var event domain.DocumentIndexRequestedEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				if moveErr := c.publishToDLQ(ctx, delivery); moveErr != nil {
					return moveErr
				}
				if err := delivery.Ack(false); err != nil {
					return fmt.Errorf("ack invalid delivery: %w", err)
				}
				continue
			}

			if err := c.handleDelivery(ctx, handler, event); err != nil {
				// Distinguir "o processo esta encerrando" de "esta mensagem
				// estourou o prazo" e essencial: no primeiro caso devolvemos o
				// controle sem ack, e o RabbitMQ reentrega para outro worker;
				// no segundo a mensagem precisa seguir o fluxo normal de
				// retry/DLQ. Consultar ctx.Err() do contexto PAI (nao o do
				// handler, que ja expirou) faz essa separacao.
				if ctx.Err() != nil {
					return ctx.Err()
				}

				disposition := classifyDocumentIndexFailure(err, extractRetryCount(delivery.Headers), c.maxRetries, c.retryDelay)

				switch disposition.Action {
				case documentIndexFailureRetry:
					if retryErr := c.publishRetry(ctx, delivery, disposition.NextRetryAttempt); retryErr != nil {
						return retryErr
					}
					if c.failureCallbacks.OnRetryScheduled != nil {
						c.failureCallbacks.OnRetryScheduled(ctx, event, disposition)
					}
				case documentIndexFailureDLQ:
					if moveErr := c.publishToDLQ(ctx, delivery); moveErr != nil {
						return moveErr
					}
					if c.failureCallbacks.OnMovedToDLQ != nil {
						c.failureCallbacks.OnMovedToDLQ(ctx, event, disposition)
					}
				case documentIndexFailurePermanent:
					if c.failureCallbacks.OnPermanentError != nil {
						c.failureCallbacks.OnPermanentError(ctx, event, disposition)
					}
				}

				if err := delivery.Ack(false); err != nil {
					return fmt.Errorf("ack failed delivery: %w", err)
				}
				continue
			}

			if err := delivery.Ack(false); err != nil {
				return fmt.Errorf("ack delivery: %w", err)
			}
		}
	}
}

// handleDelivery roda o handler com prazo maximo por mensagem.
//
// O consumidor processa uma mensagem por vez. Sem deadline, um PDF gigante ou
// um provider de LLM que aceita a conexao e nunca responde prende a goroutine
// para sempre -- e como o ctx do handler era o do processo, nada alem de um
// restart resolvia. A fila inteira parava atras de um unico documento.
//
// O erro de timeout e envolvido em ErrDocumentIndexing de proposito: e uma
// falha transiente (o proximo retry pode pegar um provider saudavel), entao
// precisa ser classificada como retry e nao como falha permanente. Ver
// classifyDocumentIndexFailure.
func (c *RabbitMQConsumer) handleDelivery(
	ctx context.Context,
	handler func(context.Context, domain.DocumentIndexRequestedEvent) error,
	event domain.DocumentIndexRequestedEvent,
) error {
	if c.handlerTimeout <= 0 {
		return handler(ctx, event)
	}

	handlerCtx, cancel := context.WithTimeout(ctx, c.handlerTimeout)
	defer cancel()

	err := handler(handlerCtx, event)

	// A checagem do contexto PAI vem primeiro: se o processo esta encerrando,
	// o handlerCtx tambem expirou por heranca, e reportar "timeout da mensagem"
	// mandaria para a DLQ um documento que so foi interrompido pelo shutdown.
	if ctx.Err() != nil {
		return err
	}

	if handlerCtx.Err() != nil && errors.Is(handlerCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w: processamento excedeu o limite de %s por mensagem", domain.ErrDocumentIndexing, c.handlerTimeout)
	}

	return err
}

func (c *RabbitMQConsumer) publishRetry(ctx context.Context, delivery amqp.Delivery, nextRetryAttempt int) error {
	headers := cloneHeaders(delivery.Headers)
	headers[retryHeader] = int32(nextRetryAttempt)

	return c.channel.PublishWithContext(ctx, c.exchange, c.retryQueue, false, false, amqp.Publishing{
		ContentType:  delivery.ContentType,
		DeliveryMode: amqp.Persistent,
		Body:         delivery.Body,
		Type:         delivery.Type,
		Headers:      headers,
	})
}

func (c *RabbitMQConsumer) publishToDLQ(ctx context.Context, delivery amqp.Delivery) error {
	headers := cloneHeaders(delivery.Headers)
	return c.channel.PublishWithContext(ctx, c.exchange, c.dlq, false, false, amqp.Publishing{
		ContentType:  delivery.ContentType,
		DeliveryMode: amqp.Persistent,
		Body:         delivery.Body,
		Type:         delivery.Type,
		Headers:      headers,
	})
}

func cloneHeaders(headers amqp.Table) amqp.Table {
	cloned := amqp.Table{}
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func extractRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	value, ok := headers[retryHeader]
	if !ok {
		return 0
	}

	switch typed := value.(type) {
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return parsed
		}
	}

	return 0
}

func classifyDocumentIndexFailure(err error, retryCount, maxRetries int, retryDelay time.Duration) DocumentIndexFailureDisposition {
	disposition := DocumentIndexFailureDisposition{
		RetryCount: retryCount,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
		Err:        err,
	}

	if isPermanentDocumentProcessingError(err) {
		disposition.Action = documentIndexFailurePermanent
		return disposition
	}

	if retryCount >= maxRetries {
		disposition.Action = documentIndexFailureDLQ
		return disposition
	}

	disposition.Action = documentIndexFailureRetry
	disposition.NextRetryAttempt = retryCount + 1
	return disposition
}

func isPermanentDocumentProcessingError(err error) bool {
	return errors.Is(err, domain.ErrDocumentSourceMissing) || errors.Is(err, domain.ErrDocumentParsingFailed)
}

func (c *RabbitMQConsumer) Close() error {
	if c == nil {
		return nil
	}
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// reconectar reabre conexao e canal do publicador.
//
// Fecha o que sobrou antes de abrir o novo: sem isso, cada queda do broker
// deixaria um socket orfao para tras, e um dia de instabilidade esgotaria os
// descritores do processo.
func (p *RabbitMQPublisher) reconectar() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}

	// A topologia e redeclarada porque o broker pode ter voltado sem estado --
	// container recriado, volume novo. Declarar de novo e idempotente. O modo
	// de confirmacao tambem e religado: ele e por canal, e um canal novo sem
	// Confirm voltaria a aceitar publicacao sem ack.
	conn, channel, retornos, err := abrirConexaoPublicador(p.cfg)
	if err != nil {
		return err
	}

	p.conn = conn
	p.channel = channel
	p.retornos = retornos
	return nil
}
