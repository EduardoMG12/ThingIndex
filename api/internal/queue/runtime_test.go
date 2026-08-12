package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"
)

func TestBootstrapRabbitMQRuntimeRetriesPublisherUntilItSucceeds(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	attempts := 0
	waitCalls := 0
	consumer := &RabbitMQConsumer{}

	publisher, resolvedConsumer, err := BootstrapRabbitMQRuntime(
		context.Background(),
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				attempts++
				if attempts < 3 {
					return nil, errors.New("dial rabbitmq: connection refused")
				}
				return &RabbitMQPublisher{}, nil
			},
			NewConsumer: func(*config.Config) (*RabbitMQConsumer, error) {
				return consumer, nil
			},
			Sleep: func(context.Context, time.Duration) error {
				waitCalls++
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("expected retry bootstrap to succeed, got %v", err)
	}
	if publisher == nil {
		t.Fatal("expected publisher to be returned")
	}
	if resolvedConsumer != consumer {
		t.Fatalf("expected consumer instance to be preserved, got %#v", resolvedConsumer)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 publisher attempts, got %d", attempts)
	}
	if waitCalls != 2 {
		t.Fatalf("expected 2 waits between failed publisher attempts, got %d", waitCalls)
	}
}

func TestBootstrapRabbitMQRuntimeRetriesConsumerInitialization(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	publisherCalls := 0
	consumerCalls := 0
	waitCalls := 0
	firstPublisher := &RabbitMQPublisher{}
	secondPublisher := &RabbitMQPublisher{}
	resolvedConsumer := &RabbitMQConsumer{}

	publisher, consumer, err := BootstrapRabbitMQRuntime(
		context.Background(),
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				publisherCalls++
				if publisherCalls == 1 {
					return firstPublisher, nil
				}
				return secondPublisher, nil
			},
			NewConsumer: func(*config.Config) (*RabbitMQConsumer, error) {
				consumerCalls++
				if consumerCalls == 1 {
					return nil, errors.New("create rabbitmq channel: temporary failure")
				}
				return resolvedConsumer, nil
			},
			Sleep: func(context.Context, time.Duration) error {
				waitCalls++
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("expected consumer retry bootstrap to succeed, got %v", err)
	}
	if publisher != secondPublisher {
		t.Fatalf("expected second publisher to remain active, got %#v", publisher)
	}
	if consumer != resolvedConsumer {
		t.Fatalf("expected final consumer to be returned, got %#v", consumer)
	}
	if waitCalls != 1 {
		t.Fatalf("expected 1 wait after failed consumer initialization, got %d", waitCalls)
	}
}

func TestBootstrapRabbitMQRuntimeStopsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := BootstrapRabbitMQRuntime(
		ctx,
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				return nil, errors.New("dial rabbitmq: connection refused")
			},
			NewConsumer: func(*config.Config) (*RabbitMQConsumer, error) {
				return &RabbitMQConsumer{}, nil
			},
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation to stop bootstrap, got %v", err)
	}
}

func TestBootstrapRabbitMQRuntimeFailsFastForPermanentPublisherError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	waitCalls := 0

	_, _, err := BootstrapRabbitMQRuntime(
		context.Background(),
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				return nil, errors.New("dial rabbitmq: invalid URI scheme")
			},
			NewConsumer: func(*config.Config) (*RabbitMQConsumer, error) {
				return &RabbitMQConsumer{}, nil
			},
			Sleep: func(context.Context, time.Duration) error {
				waitCalls++
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("expected permanent bootstrap error to be returned")
	}
	if waitCalls != 0 {
		t.Fatalf("expected no waits for permanent error, got %d", waitCalls)
	}
}

func TestBootstrapRabbitMQRuntimeFailsFastForPermanentConsumerError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	waitCalls := 0

	_, _, err := BootstrapRabbitMQRuntime(
		context.Background(),
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				return &RabbitMQPublisher{}, nil
			},
			NewConsumer: func(*config.Config) (*RabbitMQConsumer, error) {
				return nil, errors.New("create rabbitmq channel: Exception (403) ACCESS_REFUSED - Login refused")
			},
			Sleep: func(context.Context, time.Duration) error {
				waitCalls++
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("expected permanent consumer bootstrap error to be returned")
	}
	if waitCalls != 0 {
		t.Fatalf("expected no waits for permanent consumer error, got %d", waitCalls)
	}
}

func TestRabbitMQConsumerStillImplementsMessageConsumer(t *testing.T) {
	t.Parallel()

	var _ domain.MessageConsumer = (*RabbitMQConsumer)(nil)
}
