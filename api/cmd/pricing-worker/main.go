package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	"thing-index/db"
	"thing-index/internal/config"
	pkgpricing "thing-index/internal/pkg/pricing"
	syncpricing "thing-index/internal/pricing"
	"thing-index/internal/repository"
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

	repo := repository.NewGormPricingRepository(database)
	syncService := syncpricing.NewSyncService(repo, pkgpricing.NewHTTPFetcher(), pkgpricing.DefaultParsers())
	interval := config.AppConfig.PricingSyncInterval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	runSync := func() {
		result, syncErr := syncService.SyncAllEnabledSources(ctx)
		if syncErr != nil && !errors.Is(syncErr, context.Canceled) {
			log.Printf("pricing sync failed: %v", syncErr)
			return
		}
		log.Printf("pricing sync completed: sources=%d prices=%d at=%s", result.SourcesSynced, result.PricesSynced, result.SyncedAt.Format(time.RFC3339))
	}

	runSync()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runSync()
		}
	}
}
