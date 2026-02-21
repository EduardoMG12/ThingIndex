package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

	// Declare queue to ensure it exists
	r.queue, err = r.channel.QueueDeclare(
		r.qName, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}
	
	// Set QoS to ensure fair dispatch
	err = r.channel.Qos(
		10,   // prefetch count
		0,    // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

func (r *RabbitMQClient) Publish(ctx context.Context, task domain.CrawlTask) error {
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
			ContentType: "application/json",
			Body:        body,
			DeliveryMode: amqp.Persistent, 
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

	deliveryChan := make(chan domain.TaskDelivery)

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
					currentDelivery.Ack(false)
				},
				Nack: func(requeue bool) {
					currentDelivery.Nack(false, requeue)
				},
			}
		}
	}()

	return deliveryChan, nil
}

func (r *RabbitMQClient) Purge() error {
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
