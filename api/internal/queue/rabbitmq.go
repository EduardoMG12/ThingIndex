package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"thing-index/internal/config"
	"thing-index/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string
}

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
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
		conn:    conn,
		channel: channel,
		queue:   cfg.DocumentIndexQueue,
	}, nil
}

func declareTopology(channel *amqp.Channel, cfg *config.Config) error {
	if err := channel.ExchangeDeclare(
		cfg.DocumentIndexExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	if _, err := channel.QueueDeclare(
		cfg.DocumentIndexQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	if err := channel.QueueBind(
		cfg.DocumentIndexQueue,
		cfg.DocumentIndexRoutingKey,
		cfg.DocumentIndexExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind queue: %w", err)
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
				_ = delivery.Nack(false, false)
				continue
			}

			if err := handler(ctx, event); err != nil {
				_ = delivery.Nack(false, true)
				continue
			}

			if err := delivery.Ack(false); err != nil {
				return fmt.Errorf("ack delivery: %w", err)
			}
		}
	}
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
