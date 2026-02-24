package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ParserTask struct {
	Status    string `json:"status"`
	Directory string `json:"directory"`
}

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
	url     string
	qName   string
}

func NewRabbitMQClient(url, qName string) *RabbitMQClient {
	return &RabbitMQClient{
		url:   url,
		qName: qName,
	}
}

func (r *RabbitMQClient) Connect() error {
	var err error

	for i := 0; i < 5; i++ {
		r.conn, err = amqp.Dial(r.url)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ, retrying in 2s... (%d/5)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	r.queue, err = r.channel.QueueDeclare(
		r.qName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	err = r.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Also proactively declare the parsed_documents queue
	_, err = r.channel.QueueDeclare(
		"parsed_documents",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare parsed_documents queue: %w", err)
	}

	return nil
}

func (r *RabbitMQClient) Consume() (<-chan amqp.Delivery, error) {
	msgs, err := r.channel.Consume(r.qName, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}
	return msgs, nil
}

// PublishToQueue sends a message to a specific queue
func (r *RabbitMQClient) PublishToQueue(queueName string, body []byte) error {
	return r.channel.PublishWithContext(context.Background(),
		"", queueName, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
}

func (r *RabbitMQClient) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
