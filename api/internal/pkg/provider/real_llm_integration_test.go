//go:build integration

package provider

import (
	"context"
	"strings"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/testutil"
)

func TestRealLLMValidateConnectionAndGenerateText(t *testing.T) {
	cfg := testutil.RequireRealLLM(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := ValidateLLMConnection(ctx, cfg.ProviderType, cfg.Credentials); err != nil {
		t.Fatalf("ValidateLLMConnection returned error: %v", err)
	}

	llm, err := NewLLMProvider(cfg.ProviderType, cfg.Credentials)
	if err != nil {
		t.Fatalf("NewLLMProvider returned error: %v", err)
	}

	metadataProvider, ok := llm.(domain.LLMMetadataProvider)
	if !ok {
		t.Fatal("expected real llm to expose metadata")
	}
	if metadataProvider.ProviderType() != cfg.ProviderType {
		t.Fatalf("expected provider type %q, got %q", cfg.ProviderType, metadataProvider.ProviderType())
	}
	if strings.TrimSpace(metadataProvider.ModelName()) == "" {
		t.Fatal("expected resolved model name to be non-empty")
	}

	response, err := llm.GenerateText(
		ctx,
		"You are validating a production-like integration test. Reply with one short sentence and include the token THINGINDEX_REAL_LLM_OK exactly once.",
		"Confirm that the provider is reachable and include THINGINDEX_REAL_LLM_OK exactly once.",
	)
	if err != nil {
		t.Fatalf("GenerateText returned error: %v", err)
	}

	if !strings.Contains(strings.ToUpper(response), "THINGINDEX_REAL_LLM_OK") {
		t.Fatalf("expected response to contain THINGINDEX_REAL_LLM_OK, got %q", response)
	}
}

func TestRealLLMModelDiscovery(t *testing.T) {
	cfg := testutil.RequireRealLLMModelDiscovery(t)

	if cfg.ProviderType != domain.ProviderOllama {
		t.Skipf("real model discovery test is currently implemented only for %s, got %s", domain.ProviderOllama, cfg.ProviderType)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	models, err := ListLLMModels(ctx, cfg.ProviderType, cfg.Credentials)
	if err != nil {
		t.Fatalf("ListLLMModels returned error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected at least one discovered model")
	}
}
