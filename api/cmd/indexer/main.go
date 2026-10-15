package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"thing-index/db"
	internalapi "thing-index/internal/api"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/indexing"
	providerpkg "thing-index/internal/pkg/provider"
	"thing-index/internal/processing"
	"thing-index/internal/queue"
	"thing-index/internal/repository"
	"thing-index/internal/usecase"
)

func main() {
	config.Load()
	if err := config.AppConfig.Validate(); err != nil {
		log.Fatalf("invalid application configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect()
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	sqlDB, err := database.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	defaultStorage := internalapi.ResolveDefaultStorageForIndexer(func() (domain.StorageService, error) {
		return providerpkg.NewDefaultStorageProvider(config.AppConfig)
	})

	providerRepo := repository.NewGormProviderRepository(database)
	providerService := usecase.NewProviderUseCase(providerRepo, config.AppConfig)

	outboxRepo := repository.NewGormOutboxRepository(database)
	processingRepo := repository.NewGormDocumentProcessingRepository(database)
	indexer := indexing.NewDeterministicIndexer()
	indexingRepo := repository.NewGormIndexingRepository(database)
	enricher := indexing.NewLLMEnricher().WithCache(indexingRepo)
	processor := processing.NewDocumentProcessor(processingRepo, defaultStorage, providerService, indexer, providerService, enricher).WithIndexingPreferences(indexingRepo)

	publisher, err := queue.BootstrapRabbitMQRuntime(
		ctx,
		config.AppConfig,
		queue.BootstrapOptions{},
	)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		log.Fatalf("failed to initialize rabbitmq runtime: %v", err)
	}
	defer publisher.Close()

	dispatcher := processing.NewOutboxDispatcher(
		outboxRepo,
		publisher,
		config.AppConfig.OutboxPollInterval,
		config.AppConfig.OutboxBatchSize,
		config.AppConfig.OutboxVisibilityTimeout,
	)

	go func() {
		if err := dispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("outbox dispatcher stopped with error: %v", err)
			stop()
		}
	}()

	// O timeout por mensagem cobre worker lento; o reaper cobre worker que
	// morreu sem marcar FAILED (documento preso em PROCESSING).
	reaper := processing.NewStuckDocumentReaper(
		processingRepo,
		config.AppConfig.StuckDocumentThreshold,
		config.AppConfig.QueuedDocumentRequeueThreshold,
		config.AppConfig.StuckDocumentPollInterval,
	)

	go func() {
		if err := reaper.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("stuck document reaper stopped with error: %v", err)
		}
	}()

	// Consumidor supervisionado em vez de derrubar o processo. Recria o
	// consumidor a cada tentativa: conexao e canal morrem com o broker e
	// rechamar Consume no mesmo objeto da "Exception (504) channel/connection
	// is not open". Reiniciar so o consumidor preserva o dispatcher de outbox e
	// o reaper. As callbacks de falha vao em cada consumidor recriado.
	supervisor := queue.NewConsumerSupervisor(
		queue.ConsumoComCallbacks(
			func() (queue.ConsumidorDeIndexacao, error) {
				return queue.NewRabbitMQConsumer(config.AppConfig)
			},
			queue.DocumentIndexFailureCallbacks{
				OnRetryScheduled: processor.MarkRetryScheduled,
				OnMovedToDLQ:     processor.MarkMovedToDLQ,
				OnPermanentError: processor.MarkPermanentFailure,
			},
			processor.Process,
		),
		config.AppConfig.ConsumerReconnectInterval,
	)

	if err := supervisor.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("document consumer stopped with error: %v", err)
	}
}
