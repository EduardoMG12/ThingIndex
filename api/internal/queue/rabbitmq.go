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

	// Usada para reabrir a conexao quando o broker cai e volta.
	cfg *config.Config
	// Reconexao nao pode acontecer no meio de outra publicacao.
	mu sync.Mutex

	// Mensagens devolvidas por falta de rota (mandatory=true).
	retornos chan amqp.Return
	// Uma publicacao em voo por vez: o basic.return chega antes do basic.ack,
	// e so assim da para saber de qual mensagem ele e.
	publicando sync.Mutex
	sequencia  uint64
}

// ErrMensagemSemRota: o broker aceitou, mas nenhuma fila esta vinculada a routing key.
var ErrMensagemSemRota = errors.New("rabbitmq devolveu a mensagem: nenhuma fila vinculada a routing key")

// ErrPublicacaoNaoConfirmada: basic.nack ou canal fechado antes da confirmacao.
var ErrPublicacaoNaoConfirmada = errors.New("rabbitmq nao confirmou a publicacao")

// Limite de espera pelo ack quando o ctx do chamador nao tem prazo.
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
	// Prazo por mensagem; o consumidor e sequencial e uma mensagem travada para a fila toda.
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

// abrirConexaoPublicador abre conexao e canal, declara a topologia e liga
// publisher confirms e NotifyReturn. Usado tambem na reconexao.
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

	// Com buffer: a lib entrega o retorno bloqueando a goroutine de leitura.
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

// publicarConfirmado publica com mandatory=true e so devolve nil apos o
// basic.ack, sem basic.return para a mensagem.
func publicarConfirmado(
	ctx context.Context,
	canal *amqp.Channel,
	retornos chan amqp.Return,
	exchange, routingKey string,
	msg amqp.Publishing,
) error {
	// Descarta retornos de uma publicacao anterior que desistiu por timeout.
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

	// O basic.return chega antes do basic.ack; com o ack em maos, um retorno ja esta no canal.
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

// ErrDeliveriesChannelClosed: o broker fechou o canal de entregas (queda, restart,
// rede). Precisa ser erro para o processo sair com codigo != 0 e o supervisor
// reiniciar o container; com nil, restart: unless-stopped nao reinicia.
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
				// Encerramento limpo chega por ctx.Done(); aqui foi o broker.
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
				// ctx do pai cancelado = shutdown: sai sem ack e o broker reentrega.
				// Senao a mensagem estourou o prazo e segue o fluxo de retry/DLQ.
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

// handleDelivery roda o handler com prazo por mensagem. O timeout vira
// ErrDocumentIndexing para ser tratado como falha transiente (retry).
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

	// Checa o ctx pai antes: no shutdown o handlerCtx tambem expira, e nao e timeout da mensagem.
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

// reconectar fecha o que sobrou antes de abrir de novo, para nao vazar sockets.
func (p *RabbitMQPublisher) reconectar() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}

	// Redeclara a topologia (o broker pode ter voltado sem estado) e religa o
	// modo confirm, que e por canal.
	conn, channel, retornos, err := abrirConexaoPublicador(p.cfg)
	if err != nil {
		return err
	}

	p.conn = conn
	p.channel = channel
	p.retornos = retornos
	return nil
}
