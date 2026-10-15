package provider

import (
	"testing"

	"thing-index/internal/domain"
)

// O preco informado pelo usuario tem prioridade sobre a tabela interna.
func TestLLMPricingProfilePrefersUserInformedPriceOverTable(t *testing.T) {
	t.Parallel()

	pricing, ok := llmPricingProfile(domain.ProviderOpenAI, "gpt-4o-mini", map[string]string{
		CredentialInputCostPerMillionUSD:  "1.5",
		CredentialOutputCostPerMillionUSD: "3",
	})
	if !ok {
		t.Fatal("expected pricing to be available")
	}
	if pricing.InputCostPerMillionUSD != 1.5 || pricing.OutputCostPerMillionUSD != 3 {
		t.Fatalf("expected user informed pricing, got %#v", pricing)
	}
}

func TestLLMPricingProfileFallsBackToInternalTable(t *testing.T) {
	t.Parallel()

	pricing, ok := llmPricingProfile(domain.ProviderOpenAI, "gpt-4o-mini", map[string]string{})
	if !ok {
		t.Fatal("expected internal table pricing for gpt-4o-mini")
	}
	if pricing.InputCostPerMillionUSD != 0.15 || pricing.OutputCostPerMillionUSD != 0.60 {
		t.Fatalf("expected internal table pricing, got %#v", pricing)
	}
}

// Ollama e OpenRouter nao tem tabela interna: o preco informado precisa ser
// consultado antes do retorno "sem preco".
func TestLLMPricingProfileUsesUserInformedPriceForOllamaAndOpenRouter(t *testing.T) {
	t.Parallel()

	for _, providerType := range []string{domain.ProviderOllama, domain.ProviderOpenRouter} {
		pricing, ok := llmPricingProfile(providerType, "llama3.1:8b", map[string]string{
			CredentialInputCostPerMillionUSD:  "0,20",
			CredentialOutputCostPerMillionUSD: "0.80",
		})
		if !ok {
			t.Fatalf("%s: expected user informed pricing to be used", providerType)
		}
		if pricing.InputCostPerMillionUSD != 0.20 || pricing.OutputCostPerMillionUSD != 0.80 {
			t.Fatalf("%s: unexpected pricing %#v", providerType, pricing)
		}
	}
}

func TestLLMPricingProfileWithoutPriceForOllama(t *testing.T) {
	t.Parallel()

	if _, ok := llmPricingProfile(domain.ProviderOllama, "llama3.1:8b", map[string]string{"model": "llama3.1:8b"}); ok {
		t.Fatal("expected no pricing for ollama without informed price")
	}
}

func TestLLMPricingProfileAcceptsOnlyOneInformedPrice(t *testing.T) {
	t.Parallel()

	pricing, ok := llmPricingProfile(domain.ProviderOllama, "llama3.1:8b", map[string]string{
		CredentialOutputCostPerMillionUSD: "2",
	})
	if !ok {
		t.Fatal("expected pricing when only output price is informed")
	}
	if pricing.InputCostPerMillionUSD != 0 || pricing.OutputCostPerMillionUSD != 2 {
		t.Fatalf("expected missing input price to count as zero, got %#v", pricing)
	}
}

func TestLLMPricingProfileIgnoresInvalidInformedPrice(t *testing.T) {
	t.Parallel()

	pricing, ok := llmPricingProfile(domain.ProviderOpenAI, "gpt-4o", map[string]string{
		CredentialInputCostPerMillionUSD:  "-1",
		CredentialOutputCostPerMillionUSD: "3",
	})
	if !ok || pricing.InputCostPerMillionUSD != 2.50 {
		t.Fatalf("expected invalid informed price to fall back to the table, got %#v ok=%v", pricing, ok)
	}
}

func TestParsePricingValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw   string
		want  float64
		valid bool
	}{
		{"0", 0, true},
		{"2.5", 2.5, true},
		{" 2,5 ", 2.5, true},
		{"", 0, false},
		{"-0.1", 0, false},
		{"abc", 0, false},
		{"NaN", 0, false},
		{"Inf", 0, false},
	}
	for _, tc := range cases {
		got, ok := ParsePricingValue(tc.raw)
		if ok != tc.valid || (ok && got != tc.want) {
			t.Fatalf("ParsePricingValue(%q) = %v, %v; want %v, %v", tc.raw, got, ok, tc.want, tc.valid)
		}
	}
}

func TestPublicMetadataExposesInformedLLMPricing(t *testing.T) {
	t.Parallel()

	metadata := PublicMetadata(domain.FamilyLLM, map[string]string{
		"api_key":                         "secret",
		"model":                           "llama3.1:8b",
		CredentialInputCostPerMillionUSD:  "0.2",
		CredentialOutputCostPerMillionUSD: "0.8",
	})
	if metadata[CredentialInputCostPerMillionUSD] != "0.2" || metadata[CredentialOutputCostPerMillionUSD] != "0.8" {
		t.Fatalf("expected pricing in public metadata, got %#v", metadata)
	}
	if _, leaked := metadata["api_key"]; leaked {
		t.Fatalf("api_key must never be public metadata: %#v", metadata)
	}
}
