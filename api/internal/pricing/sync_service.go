package pricing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	pkgpricing "thing-index/internal/pkg/pricing"
)

type SyncService struct {
	repo    domain.PricingSourceRepository
	fetcher domain.PricingFetcher
	parsers *pkgpricing.Registry
}

func NewSyncService(repo domain.PricingSourceRepository, fetcher domain.PricingFetcher, parsers *pkgpricing.Registry) domain.PricingSyncService {
	return &SyncService{
		repo:    repo,
		fetcher: fetcher,
		parsers: parsers,
	}
}

func (s *SyncService) SyncAllEnabledSources(ctx context.Context) (*domain.PricingSyncResult, error) {
	sources, err := s.repo.ListAllEnabledSources(ctx)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return &domain.PricingSyncResult{SyncedAt: time.Now().UTC()}, nil
	}

	totalPrices := 0
	totalSources := 0
	for i := range sources {
		startedAt := time.Now().UTC()
		count, syncErr := s.syncSingleSource(ctx, &sources[i])
		if syncErr != nil {
			_ = s.repo.MarkSourceSyncFailed(ctx, &sources[i], time.Now().UTC(), time.Since(startedAt).Milliseconds(), syncErr.Error())
			log.Printf("pricing sync skipped for user=%d source=%d provider=%s: %v", sources[i].UserID, sources[i].ID, sources[i].ProviderType, syncErr)
			continue
		}
		_ = s.repo.MarkSourceSyncSucceeded(ctx, &sources[i], time.Now().UTC(), time.Since(startedAt).Milliseconds(), count)
		totalSources++
		totalPrices += count
	}

	return &domain.PricingSyncResult{
		SourcesSynced: totalSources,
		PricesSynced:  totalPrices,
		SyncedAt:      time.Now().UTC(),
	}, nil
}

func (s *SyncService) syncSingleSource(ctx context.Context, source *models.PricingSource) (int, error) {
	parser, ok := s.parsers.ParserFor(source.ProviderType)
	if !ok {
		return 0, domain.ErrPricingParserUnavailable
	}

	body, err := s.fetcher.Fetch(ctx, source.SourceURL)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", domain.ErrPricingFetchFailed, err)
	}
	defer body.Close()

	parsed, err := parser.Parse(ctx, source.SourceURL, body)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", domain.ErrPricingParseFailed, err)
	}

	now := time.Now().UTC()
	prices := make([]models.ModelPrice, 0, len(parsed))
	for _, item := range parsed {
		prices = append(prices, models.ModelPrice{
			ModelKey:                     item.ModelKey,
			ModelName:                    item.ModelName,
			Unit:                         normalized(item.Unit, "1M tokens"),
			Currency:                     normalized(item.Currency, "USD"),
			FlatPriceUSD:                 item.FlatPriceUSD,
			InputCostPerMillionUSD:       item.InputCostPerMillionUSD,
			OutputCostPerMillionUSD:      item.OutputCostPerMillionUSD,
			CachedInputCostPerMillionUSD: item.CachedInputCostPerMillionUSD,
			ContextCostPerMillionUSD:     item.ContextCostPerMillionUSD,
			PriceNotes:                   item.PriceNotes,
			LastSyncedAt:                 now,
		})
	}

	if err := s.repo.ReplacePrices(ctx, source, prices, now); err != nil {
		return 0, err
	}
	return len(prices), nil
}

func normalized(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
