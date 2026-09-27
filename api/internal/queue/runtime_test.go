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

	publisher, err := BootstrapRabbitMQRuntime(
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
	if attempts != 3 {
		t.Fatalf("expected 3 publisher attempts, got %d", attempts)
	}
	if waitCalls != 2 {
		t.Fatalf("expected 2 waits between failed publisher attempts, got %d", waitCalls)
	}
}

func TestBootstrapRabbitMQRuntimeStopsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := BootstrapRabbitMQRuntime(
		ctx,
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				return nil, errors.New("dial rabbitmq: connection refused")
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

	_, err := BootstrapRabbitMQRuntime(
		context.Background(),
		cfg,
		BootstrapOptions{
			NewPublisher: func(*config.Config) (*RabbitMQPublisher, error) {
				return nil, errors.New("dial rabbitmq: invalid URI scheme")
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

func TestRabbitMQConsumerStillImplementsMessageConsumer(t *testing.T) {
	t.Parallel()

	var _ domain.MessageConsumer = (*RabbitMQConsumer)(nil)
}
