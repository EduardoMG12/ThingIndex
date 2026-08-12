package queue

import (
	"context"
	"log"
	"strings"
	"time"

	"thing-index/internal/config"
)

const defaultBootstrapRetryDelay = 2 * time.Second

type BootstrapOptions struct {
	RetryDelay   time.Duration
	Sleep        func(context.Context, time.Duration) error
	NewPublisher func(*config.Config) (*RabbitMQPublisher, error)
	NewConsumer  func(*config.Config) (*RabbitMQConsumer, error)
}

func BootstrapRabbitMQRuntime(
	ctx context.Context,
	cfg *config.Config,
	opts BootstrapOptions,
) (*RabbitMQPublisher, *RabbitMQConsumer, error) {
	opts = withBootstrapDefaults(opts)

	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		publisher, err := opts.NewPublisher(cfg)
		if err != nil {
			if !isRetryableBootstrapError(err) {
				return nil, nil, err
			}
			if waitErr := waitForBootstrapRetry(ctx, opts.Sleep, opts.RetryDelay, "publisher", err); waitErr != nil {
				return nil, nil, waitErr
			}
			continue
		}

		consumer, err := opts.NewConsumer(cfg)
		if err == nil {
			return publisher, consumer, nil
		}

		_ = publisher.Close()
		if !isRetryableBootstrapError(err) {
			return nil, nil, err
		}
		if waitErr := waitForBootstrapRetry(ctx, opts.Sleep, opts.RetryDelay, "consumer", err); waitErr != nil {
			return nil, nil, waitErr
		}
	}
}

func withBootstrapDefaults(opts BootstrapOptions) BootstrapOptions {
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = defaultBootstrapRetryDelay
	}
	if opts.Sleep == nil {
		opts.Sleep = sleepWithContext
	}
	if opts.NewPublisher == nil {
		opts.NewPublisher = NewRabbitMQPublisher
	}
	if opts.NewConsumer == nil {
		opts.NewConsumer = NewRabbitMQConsumer
	}
	return opts
}

func waitForBootstrapRetry(
	ctx context.Context,
	wait func(context.Context, time.Duration) error,
	delay time.Duration,
	component string,
	err error,
) error {
	log.Printf("rabbitmq %s initialization failed, retrying in %s: %v", component, delay, err)
	return wait(ctx, delay)
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableBootstrapError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	permanentMarkers := []string{
		"access refused",
		"access_refused",
		"authentication",
		"authorization",
		"login refused",
		"sasl",
		"no access to this vhost",
		"vhost",
		"invalid uri",
		"uri scheme",
		"missing protocol scheme",
		"malformed",
		"invalid port",
		"non-positive port",
	}
	for _, marker := range permanentMarkers {
		if strings.Contains(message, marker) {
			return false
		}
	}

	return true
}
