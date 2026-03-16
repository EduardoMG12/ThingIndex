package domain

import (
	"context"
	"io"
	"time"

	"thing-index/internal/models"
)

const (
	ProviderOllama = "OLLAMA"
)

type PricingSourceRepository interface {
	SaveSource(ctx context.Context, source *models.PricingSource) error
	GetSource(ctx context.Context, userID, sourceID uint) (*models.PricingSource, error)
	ListSources(ctx context.Context, userID uint) ([]models.PricingSource, error)
	ListEnabledSources(ctx context.Context, userID uint) ([]models.PricingSource, error)
	ListAllEnabledSources(ctx context.Context) ([]models.PricingSource, error)
	ReplacePrices(ctx context.Context, source *models.PricingSource, prices []models.ModelPrice, syncedAt time.Time) error
	MarkSourceSyncSucceeded(ctx context.Context, source *models.PricingSource, finishedAt time.Time, durationMs int64, priceCount int) error
	MarkSourceSyncFailed(ctx context.Context, source *models.PricingSource, finishedAt time.Time, durationMs int64, failure string) error
	ListPrices(ctx context.Context, userID uint) ([]models.ModelPrice, error)
	ListPriceHistory(ctx context.Context, userID uint, limit int) ([]models.ModelPriceHistory, error)
}

type PricingUseCase interface {
	SaveSource(ctx context.Context, userID uint, input PricingSourceInput) (*models.PricingSource, error)
	ListSources(ctx context.Context, userID uint) ([]models.PricingSource, error)
	ListPrices(ctx context.Context, userID uint) ([]models.ModelPrice, error)
	ListPriceHistory(ctx context.Context, userID uint, limit int) ([]models.ModelPriceHistory, error)
}

type PricingSourceInput struct {
	ProviderType string
	Name         string
	SourceURL    string
	IsEnabled    bool
}

type PricingSyncResult struct {
	SourcesSynced int
	PricesSynced  int
	SyncedAt      time.Time
}

type PricingSyncService interface {
	SyncAllEnabledSources(ctx context.Context) (*PricingSyncResult, error)
}

type PricingFetcher interface {
	Fetch(ctx context.Context, sourceURL string) (io.ReadCloser, error)
}

type PricingParser interface {
	ProviderType() string
	Parse(ctx context.Context, sourceURL string, body io.Reader) ([]ParsedModelPrice, error)
}

type ParsedModelPrice struct {
	ModelKey                     string
	ModelName                    string
	Unit                         string
	Currency                     string
	FlatPriceUSD                 float64
	InputCostPerMillionUSD       float64
	OutputCostPerMillionUSD      float64
	CachedInputCostPerMillionUSD float64
	ContextCostPerMillionUSD     float64
	PriceNotes                   string
}
