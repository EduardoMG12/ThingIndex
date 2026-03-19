package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
}

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

	failureCallbacks DocumentIndexFailureCallbacks
}

func NewRabbitMQPublisher(cfg *config.Config) (*RabbitMQPublisher, error) {
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

	return &RabbitMQPublisher{
		conn:       conn,
		channel:    channel,
		exchange:   cfg.DocumentIndexExchange,
		routingKey: cfg.DocumentIndexRoutingKey,
	}, nil
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
		conn:       conn,
		channel:    channel,
		exchange:   cfg.DocumentIndexExchange,
		routingKey: cfg.DocumentIndexRoutingKey,
		queue:      cfg.DocumentIndexQueue,
		retryQueue: cfg.DocumentIndexRetryQueue,
		dlq:        cfg.DocumentIndexDLQ,
		retryDelay: cfg.DocumentIndexRetryDelay,
		maxRetries: cfg.DocumentIndexMaxRetries,
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

	return p.channel.PublishWithContext(ctx, p.exchange, p.routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Type:         domain.OutboxEventDocumentIndexRequested,
		Headers: amqp.Table{
			retryHeader: int32(0),
		},
	})
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
				return nil
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

			if err := handler(ctx, event); err != nil {
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
