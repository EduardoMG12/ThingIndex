package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"thing-index/db"
	"thing-index/internal/config"
	"thing-index/internal/indexing"
	"thing-index/internal/pkg/storage"
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

	minioStorage, err := storage.NewMinioStorage(
		config.AppConfig.MinioEndpoint,
		config.AppConfig.MinioAccessKey,
		config.AppConfig.MinioSecretKey,
		config.AppConfig.MinioBucketName,
		config.AppConfig.MinioUseSSL == "true",
	)
	if err != nil {
		log.Fatalf("failed to initialize minio storage: %v", err)
	}

	providerRepo := repository.NewGormProviderRepository(database)
	providerService := usecase.NewProviderUseCase(providerRepo, config.AppConfig)

	outboxRepo := repository.NewGormOutboxRepository(database)
	processingRepo := repository.NewGormDocumentProcessingRepository(database)
	indexer := indexing.NewDeterministicIndexer()
	enricher := indexing.NewLLMEnricher()
	processor := processing.NewDocumentProcessor(processingRepo, minioStorage, providerService, indexer, providerService, enricher)

	publisher, err := queue.NewRabbitMQPublisher(config.AppConfig)
	if err != nil {
		log.Fatalf("failed to initialize rabbitmq publisher: %v", err)
	}
	defer publisher.Close()

	consumer, err := queue.NewRabbitMQConsumer(config.AppConfig)
	if err != nil {
		log.Fatalf("failed to initialize rabbitmq consumer: %v", err)
	}
	defer consumer.Close()

	dispatcher := processing.NewOutboxDispatcher(
		outboxRepo,
		publisher,
		config.AppConfig.OutboxPollInterval,
		config.AppConfig.OutboxBatchSize,
	)

	go func() {
		if err := dispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("outbox dispatcher stopped with error: %v", err)
			stop()
		}
	}()

	if err := consumer.ConsumeDocumentIndexRequested(ctx, processor.Process); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("document consumer stopped with error: %v", err)
	}
}
