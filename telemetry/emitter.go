package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Event represents a single telemetry event emitted by a worker.
type Event struct {
	Worker    string            `json:"worker"`
	Stage     string            `json:"stage"`
	EventType string            `json:"event_type"` // "info" | "warning" | "error" | "success"
	Message   string            `json:"message"`
	Timestamp string            `json:"timestamp"`
	Metadata  map[string]int    `json:"metadata,omitempty"`
}

// Emitter publishes telemetry events to a RabbitMQ topic exchange.
// It is safe for concurrent use.
type Emitter struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	workerName   string
	mu           sync.Mutex
}

// NewEmitter creates and returns a connected Emitter.
// If connection fails after retries, it returns a no-op emitter that silently
// discards events — workers must never be blocked by telemetry failures.
func NewEmitter(rabbitmqURL, exchangeName, workerName string) *Emitter {
	e := &Emitter{
		exchangeName: exchangeName,
		workerName:   workerName,
	}

	if err := e.connect(rabbitmqURL); err != nil {
		log.Printf("[Telemetry] WARNING: could not connect to RabbitMQ for telemetry: %v — events will be discarded", err)
		return e
	}

	log.Printf("[Telemetry] Emitter initialized for worker '%s' on exchange '%s'", workerName, exchangeName)
	return e
}

func (e *Emitter) connect(url string) error {
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		e.conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		delay := time.Duration(1<<uint(i)) * time.Second
		if delay > 10*time.Second {
			delay = 10 * time.Second
		}
		log.Printf("[Telemetry] Failed to connect, retrying in %v... (%d/%d)", delay, i+1, maxRetries)
		time.Sleep(delay)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	e.channel, err = e.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the topic exchange (idempotent)
	err = e.channel.ExchangeDeclare(
		e.exchangeName, // name
		"topic",        // type
		false,          // durable — telemetry is ephemeral
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	return nil
}

// Emit publishes a telemetry event. It is fire-and-forget:
// errors are logged but never propagated to the caller.
func (e *Emitter) Emit(stage, eventType, message string, metadata map[string]int) {
	if e.channel == nil {
		return // no-op if not connected
	}

	event := Event{
		Worker:    e.workerName,
		Stage:     stage,
		EventType: eventType,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
		Metadata:  metadata,
	}

	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Telemetry] Failed to marshal event: %v", err)
		return
	}

	routingKey := fmt.Sprintf("worker.%s.%s", e.workerName, stage)

	e.mu.Lock()
	defer e.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = e.channel.PublishWithContext(ctx,
		e.exchangeName, // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("[Telemetry] Failed to publish event: %v", err)
	}
}

// Close gracefully shuts down the emitter connection.
func (e *Emitter) Close() {
	if e.channel != nil {
		e.channel.Close()
	}
	if e.conn != nil {
		e.conn.Close()
	}
}
