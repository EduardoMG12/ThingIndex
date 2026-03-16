package processing

import (
	"context"
	"math"
	"strings"

	"thing-index/internal/domain"
)

type llmUsageSnapshot struct {
	ProviderType          string
	Model                 string
	RequestCount          int
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	EstimatedTotalTokens  int
	PricingAvailable      bool
	InputCostUSD          float64
	OutputCostUSD         float64
	TotalCostUSD          float64
}

type observedLLMService struct {
	base     domain.LLMService
	usage    llmUsageSnapshot
	pricing  domain.LLMPricing
	hasPrice bool
}

func newObservedLLMService(base domain.LLMService) *observedLLMService {
	observer := &observedLLMService{base: base}
	if metadataProvider, ok := base.(domain.LLMMetadataProvider); ok {
		observer.usage.ProviderType = metadataProvider.ProviderType()
		observer.usage.Model = metadataProvider.ModelName()
	}
	if pricingProvider, ok := base.(domain.LLMPricingProvider); ok {
		if pricing, ok := pricingProvider.Pricing(); ok {
			observer.pricing = pricing
			observer.hasPrice = true
			observer.usage.PricingAvailable = true
		}
	}
	return observer
}

func (o *observedLLMService) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	output, err := o.base.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	o.usage.RequestCount++
	o.usage.EstimatedInputTokens += estimateTextTokens(systemPrompt) + estimateTextTokens(userPrompt)
	o.usage.EstimatedOutputTokens += estimateTextTokens(output)
	o.usage.EstimatedTotalTokens = o.usage.EstimatedInputTokens + o.usage.EstimatedOutputTokens

	if o.hasPrice {
		o.usage.InputCostUSD = estimateCostUSD(o.usage.EstimatedInputTokens, o.pricing.InputCostPerMillionUSD)
		o.usage.OutputCostUSD = estimateCostUSD(o.usage.EstimatedOutputTokens, o.pricing.OutputCostPerMillionUSD)
		o.usage.TotalCostUSD = o.usage.InputCostUSD + o.usage.OutputCostUSD
	}

	return output, nil
}

func (o *observedLLMService) Snapshot() llmUsageSnapshot {
	return o.usage
}

func estimateTextTokens(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}

	runes := len([]rune(trimmed))
	return int(math.Ceil(float64(runes) / 4.0))
}

func estimateCostUSD(tokens int, costPerMillion float64) float64 {
	if tokens <= 0 || costPerMillion <= 0 {
		return 0
	}
	return (float64(tokens) / 1_000_000.0) * costPerMillion
}
