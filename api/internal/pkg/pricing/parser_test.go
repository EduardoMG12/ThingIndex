package pricing

import (
	"context"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

func TestDefaultParsersParseKnownProviders(t *testing.T) {
	t.Parallel()

	registry := DefaultParsers()
	cases := []struct {
		name         string
		providerType string
		html         string
		modelName    string
		inputCost    float64
		outputCost   float64
	}{
		{
			name:         "openai",
			providerType: domain.ProviderOpenAI,
			html: `<html><body>
				<div>gpt-4o-mini-2024-07-18$3.00$0.15$0.075$0.60</div>
			</body></html>`,
			modelName:  "gpt-4o-mini-2024-07-18",
			inputCost:  0.15,
			outputCost: 0.60,
		},
		{
			name:         "deepseek",
			providerType: domain.ProviderDeepSeek,
			html: `<html><body>
				<div>deepseek-chat</div>
				<div>Input $0.27 / 1M tokens</div>
				<div>Output $1.10 / 1M tokens</div>
			</body></html>`,
			modelName:  "deepseek-chat",
			inputCost:  0.27,
			outputCost: 1.10,
		},
		{
			name:         "gemini",
			providerType: domain.ProviderGemini,
			html: `<html><body>
				<div>Gemini 2.5 Pro</div>
				<div>Input $1.25 / 1M tokens</div>
				<div>Output $10.00 / 1M tokens</div>
			</body></html>`,
			modelName:  "Gemini 2.5 Pro",
			inputCost:  1.25,
			outputCost: 10.00,
		},
		{
			name:         "ollama",
			providerType: domain.ProviderOllama,
			html: `<html><body>
				<h2>Free</h2>
				<p>$0/mo</p>
			</body></html>`,
			modelName: "Free",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser, ok := registry.ParserFor(tc.providerType)
			if !ok {
				t.Fatalf("parser not found for %s", tc.providerType)
			}

			prices, err := parser.Parse(context.Background(), "https://example.com", strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if len(prices) != 1 {
				t.Fatalf("expected 1 price, got %d", len(prices))
			}
			if prices[0].ModelName != tc.modelName {
				t.Fatalf("unexpected model name: %q", prices[0].ModelName)
			}
			if prices[0].InputCostPerMillionUSD != tc.inputCost {
				t.Fatalf("unexpected input cost: %f", prices[0].InputCostPerMillionUSD)
			}
			if prices[0].OutputCostPerMillionUSD != tc.outputCost {
				t.Fatalf("unexpected output cost: %f", prices[0].OutputCostPerMillionUSD)
			}
		})
	}
}

func TestDefaultParsersRejectUnsupportedMarkup(t *testing.T) {
	t.Parallel()

	parser, ok := DefaultParsers().ParserFor(domain.ProviderOpenAI)
	if !ok {
		t.Fatal("expected parser")
	}

	_, err := parser.Parse(context.Background(), "https://example.com", strings.NewReader(`<html><body><p>business plan $25</p></body></html>`))
	if err == nil {
		t.Fatal("expected parse error")
	}
}
