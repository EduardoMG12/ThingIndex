package usecase

import (
	"context"
	"net/url"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type pricingUseCase struct {
	repo domain.PricingSourceRepository
}

func NewPricingUseCase(repo domain.PricingSourceRepository) domain.PricingUseCase {
	return &pricingUseCase{
		repo: repo,
	}
}

func (u *pricingUseCase) SaveSource(ctx context.Context, userID uint, input domain.PricingSourceInput) (*models.PricingSource, error) {
	providerType, err := normalizePricingProvider(input.ProviderType)
	if err != nil {
		return nil, err
	}
	sourceURL := strings.TrimSpace(input.SourceURL)
	if err := validatePricingSourceURL(sourceURL); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = providerType + " Pricing"
	}

	source := &models.PricingSource{
		UserID:       userID,
		ProviderType: providerType,
		Name:         name,
		SourceURL:    sourceURL,
		IsEnabled:    input.IsEnabled,
	}
	if err := u.repo.SaveSource(ctx, source); err != nil {
		return nil, err
	}
	return source, nil
}

func (u *pricingUseCase) ListSources(ctx context.Context, userID uint) ([]models.PricingSource, error) {
	return u.repo.ListSources(ctx, userID)
}

func (u *pricingUseCase) ListPrices(ctx context.Context, userID uint) ([]models.ModelPrice, error) {
	return u.repo.ListPrices(ctx, userID)
}

func validatePricingSourceURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return domain.ErrPricingSourceInvalid
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return domain.ErrPricingSourceInvalid
	}
	return nil
}

func normalizePricingProvider(providerType string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(providerType))
	switch normalized {
	case domain.ProviderOpenAI, domain.ProviderGemini, domain.ProviderDeepSeek, domain.ProviderOllama:
		return normalized, nil
	default:
		return "", domain.ErrProviderTypeInvalid
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
