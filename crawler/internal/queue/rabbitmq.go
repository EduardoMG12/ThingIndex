package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/user/legisgraph-crawler/internal/domain"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
	url     string
	qName   string
	mu      sync.Mutex // protects channel publishing
}

func NewRabbitMQClient(url, qName string) (*RabbitMQClient, error) {
	return &RabbitMQClient{
		url:   url,
		qName: qName,
	}, nil
}

func (r *RabbitMQClient) Connect() error {
	var err error
	
	// Retry logic for connection
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

	// Declare queue to ensure it exists with priority argument
	args := amqp.Table{"x-max-priority": int32(10)}
	r.queue, err = r.channel.QueueDeclare(
		r.qName, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		args,    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare main queue: %w", err)
	}

	// Also proactively declare the queue we publish to, so publishers never block waiting
	_, err = r.channel.QueueDeclare(
		"raw_documents",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare raw_documents queue: %w", err)
	}
	
	// Set QoS to ensure fair dispatch
	err = r.channel.Qos(
		3,   // prefetch count
		0,    // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

func (r *RabbitMQClient) Publish(ctx context.Context, task domain.CrawlTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = r.channel.PublishWithContext(ctx,
		"",      // exchange
		r.qName, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Priority:     0,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}
	return nil
}

func (r *RabbitMQClient) Consume() (<-chan domain.TaskDelivery, error) {
	msgs, err := r.channel.Consume(
		r.qName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	// buffered channel to avoid blocking the main RabbitMQ consumer loop
	deliveryChan := make(chan domain.TaskDelivery, 10)

	go func() {
		defer close(deliveryChan)
		for d := range msgs {
			var task domain.CrawlTask
			if err := json.Unmarshal(d.Body, &task); err != nil {
				log.Printf("Error unmarshalling task: %v", err)
				d.Nack(false, false)
				continue
			}

			// Capture the current delivery instance
			currentDelivery := d
			
			deliveryChan <- domain.TaskDelivery{
				Task: task,
				Ack: func() {
					r.mu.Lock()
					defer r.mu.Unlock()
					currentDelivery.Ack(false)
				},
				Nack: func(requeue bool) {
					r.mu.Lock()
					defer r.mu.Unlock()
					currentDelivery.Nack(false, requeue)
				},
			}
		}
	}()

	return deliveryChan, nil
}

// PublishToQueue sends a raw byte payload to a specific queue
func (r *RabbitMQClient) PublishToQueue(ctx context.Context, queueName string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.channel.PublishWithContext(ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
}

// GetMessageCount returns the number of messages sitting in the main queue
func (r *RabbitMQClient) GetMessageCount() (int, error) {
	return r.GetQueueMessages(r.qName)
}

// GetQueueMessages returns the number of messages sitting in any specified queue
func (r *RabbitMQClient) GetQueueMessages(queueName string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, err := r.channel.QueueInspect(queueName)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

func (r *RabbitMQClient) Purge() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.channel.QueuePurge(r.qName, false)
	if err != nil {
		return fmt.Errorf("failed to purge queue: %w", err)
	}
	return nil
}

func (r *RabbitMQClient) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
