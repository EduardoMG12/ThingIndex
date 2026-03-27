//go:build integration

package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

var loadQueueIntegrationEnvOnce sync.Once

func TestRabbitMQPublisherAndConsumerDeliverDocumentIndexRequested(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	publisher := newIntegrationPublisher(t, cfg)
	defer publisher.Close()

	consumer := newIntegrationConsumer(t, cfg)
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	receivedCh := make(chan domain.DocumentIndexRequestedEvent, 1)
	errCh := make(chan error, 1)

	go func() {
		errCh <- consumer.ConsumeDocumentIndexRequested(ctx, func(_ context.Context, event domain.DocumentIndexRequestedEvent) error {
			receivedCh <- event
			cancel()
			return nil
		})
	}()

	expected := domain.DocumentIndexRequestedEvent{DocumentID: 77, UserID: 9}
	if err := publisher.PublishDocumentIndexRequested(context.Background(), expected); err != nil {
		t.Fatalf("PublishDocumentIndexRequested returned error: %v", err)
	}

	select {
	case received := <-receivedCh:
		if received != expected {
			t.Fatalf("expected event %#v, got %#v", expected, received)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for consumed event")
	}

	if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("ConsumeDocumentIndexRequested returned error: %v", err)
	}
}

func TestRabbitMQConsumerSchedulesRetryWithIncrementedHeader(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	cfg.DocumentIndexRetryDelay = 30 * time.Second
	cfg.DocumentIndexMaxRetries = 3

	publisher := newIntegrationPublisher(t, cfg)
	defer publisher.Close()

	consumer := newIntegrationConsumer(t, cfg)
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callbackCh := make(chan DocumentIndexFailureDisposition, 1)
	consumer.SetFailureCallbacks(DocumentIndexFailureCallbacks{
		OnRetryScheduled: func(_ context.Context, event domain.DocumentIndexRequestedEvent, disposition DocumentIndexFailureDisposition) {
			callbackCh <- disposition
			cancel()
		},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.ConsumeDocumentIndexRequested(ctx, func(_ context.Context, event domain.DocumentIndexRequestedEvent) error {
			return errors.New("temporary integration failure")
		})
	}()

	expected := domain.DocumentIndexRequestedEvent{DocumentID: 88, UserID: 10}
	if err := publisher.PublishDocumentIndexRequested(context.Background(), expected); err != nil {
		t.Fatalf("PublishDocumentIndexRequested returned error: %v", err)
	}

	select {
	case disposition := <-callbackCh:
		if disposition.Action != documentIndexFailureRetry {
			t.Fatalf("expected retry action, got %#v", disposition)
		}
		if disposition.NextRetryAttempt != 1 {
			t.Fatalf("expected next retry attempt 1, got %#v", disposition)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for retry callback")
	}

	if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("ConsumeDocumentIndexRequested returned error: %v", err)
	}

	channel := newInspectionRabbitChannel(t, cfg)
	defer channel.Close()

	delivery := waitForRabbitMessage(t, channel, cfg.DocumentIndexRetryQueue, 5*time.Second)
	assertRabbitEventEquals(t, delivery.Body, expected)
	if got := extractRetryCount(delivery.Headers); got != 1 {
		t.Fatalf("expected retry header 1, got %d", got)
	}
}

func TestRabbitMQConsumerMovesExhaustedMessageToDLQ(t *testing.T) {
	cfg := newIntegrationRabbitMQConfig(t)
	cfg.DocumentIndexRetryDelay = 30 * time.Second
	cfg.DocumentIndexMaxRetries = 2

	channel := newInspectionRabbitChannel(t, cfg)
	defer channel.Close()

	consumer := newIntegrationConsumer(t, cfg)
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callbackCh := make(chan DocumentIndexFailureDisposition, 1)
	consumer.SetFailureCallbacks(DocumentIndexFailureCallbacks{
		OnMovedToDLQ: func(_ context.Context, event domain.DocumentIndexRequestedEvent, disposition DocumentIndexFailureDisposition) {
			callbackCh <- disposition
			cancel()
		},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.ConsumeDocumentIndexRequested(ctx, func(_ context.Context, event domain.DocumentIndexRequestedEvent) error {
			return errors.New("persistent integration failure")
		})
	}()

	expected := domain.DocumentIndexRequestedEvent{DocumentID: 99, UserID: 11}
	body, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if err := channel.PublishWithContext(context.Background(), cfg.DocumentIndexExchange, cfg.DocumentIndexRoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Type:         domain.OutboxEventDocumentIndexRequested,
		Headers: amqp.Table{
			retryHeader: int32(cfg.DocumentIndexMaxRetries),
		},
	}); err != nil {
		t.Fatalf("PublishWithContext returned error: %v", err)
	}

	select {
	case disposition := <-callbackCh:
		if disposition.Action != documentIndexFailureDLQ {
			t.Fatalf("expected dlq action, got %#v", disposition)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for dlq callback")
	}

	if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("ConsumeDocumentIndexRequested returned error: %v", err)
	}

	delivery := waitForRabbitMessage(t, channel, cfg.DocumentIndexDLQ, 5*time.Second)
	assertRabbitEventEquals(t, delivery.Body, expected)
	if got := extractRetryCount(delivery.Headers); got != cfg.DocumentIndexMaxRetries {
		t.Fatalf("expected retry header %d, got %d", cfg.DocumentIndexMaxRetries, got)
	}
}

func newIntegrationPublisher(t *testing.T, cfg *config.Config) *RabbitMQPublisher {
	t.Helper()

	publisher, err := NewRabbitMQPublisher(cfg)
	if err != nil {
		t.Fatalf("NewRabbitMQPublisher returned error: %v", err)
	}

	return publisher
}

func newIntegrationConsumer(t *testing.T, cfg *config.Config) *RabbitMQConsumer {
	t.Helper()

	consumer, err := NewRabbitMQConsumer(cfg)
	if err != nil {
		t.Fatalf("NewRabbitMQConsumer returned error: %v", err)
	}

	return consumer
}

func newInspectionRabbitChannel(t *testing.T, cfg *config.Config) *amqp.Channel {
	t.Helper()

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		t.Fatalf("Channel returned error: %v", err)
	}

	return channel
}

func deleteIntegrationRabbitTopology(t *testing.T, channel *amqp.Channel, cfg *config.Config) {
	t.Helper()

	for _, queueName := range []string{cfg.DocumentIndexQueue, cfg.DocumentIndexRetryQueue, cfg.DocumentIndexDLQ} {
		if _, err := channel.QueueDelete(queueName, false, false, false); err != nil {
			t.Fatalf("QueueDelete(%s) returned error: %v", queueName, err)
		}
	}
	if err := channel.ExchangeDelete(cfg.DocumentIndexExchange, false, false); err != nil {
		t.Fatalf("ExchangeDelete(%s) returned error: %v", cfg.DocumentIndexExchange, err)
	}
}

func waitForRabbitMessage(t *testing.T, channel *amqp.Channel, queueName string, timeout time.Duration) amqp.Delivery {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		delivery, ok, err := channel.Get(queueName, true)
		if err != nil {
			t.Fatalf("Get(%s) returned error: %v", queueName, err)
		}
		if ok {
			return delivery
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for message in queue %s", queueName)
	return amqp.Delivery{}
}

func assertRabbitEventEquals(t *testing.T, body []byte, expected domain.DocumentIndexRequestedEvent) {
	t.Helper()

	var got domain.DocumentIndexRequestedEvent
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected event %#v, got %#v", expected, got)
	}
}

func newIntegrationRabbitMQConfig(t *testing.T) *config.Config {
	t.Helper()

	loadQueueIntegrationEnvOnce.Do(func() {
		_ = godotenv.Load("../../../.env")
	})

	baseURL := normalizeRabbitMQURL(firstQueueNonEmpty(os.Getenv("THINGINDEX_TEST_RABBITMQ_URL"), os.Getenv("RABBITMQ_URL")))
	if baseURL == "" {
		baseURL = "amqp://guest:guest@127.0.0.1:5672/"
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	cfg := &config.Config{
		RabbitMQURL:             baseURL,
		DocumentIndexExchange:   "thingindex.integration.exchange." + suffix,
		DocumentIndexQueue:      "thingindex.integration.queue." + suffix,
		DocumentIndexRoutingKey: "thingindex.integration.route." + suffix,
		DocumentIndexRetryQueue: "thingindex.integration.retry." + suffix,
		DocumentIndexDLQ:        "thingindex.integration.dlq." + suffix,
		DocumentIndexRetryDelay: 30 * time.Second,
		DocumentIndexMaxRetries: 2,
		ConsumerPrefetchCount:   1,
	}

	t.Cleanup(func() {
		conn, err := amqp.Dial(cfg.RabbitMQURL)
		if err != nil {
			t.Fatalf("cleanup Dial returned error: %v", err)
		}
		defer conn.Close()

		channel, err := conn.Channel()
		if err != nil {
			t.Fatalf("cleanup Channel returned error: %v", err)
		}
		defer channel.Close()

		deleteIntegrationRabbitTopology(t, channel, cfg)
	})

	return cfg
}

func normalizeRabbitMQURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	host := parsed.Hostname()
	if host == "rabbitmq" {
		port := parsed.Port()
		if port == "" {
			port = "5672"
		}
		parsed.Host = "127.0.0.1:" + port
	}

	return parsed.String()
}

func firstQueueNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
