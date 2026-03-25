//go:build integration

package provider

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode"

	"thing-index/internal/domain"
	"thing-index/internal/testutil"
)

func TestRealLLMValidateConnectionAndGenerateText(t *testing.T) {
	cfg := testutil.RequireRealLLM(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := ValidateLLMConnection(ctx, cfg.ProviderType, cfg.Credentials); err != nil {
		skipIfTransientRealLLMInfraError(t, "ValidateLLMConnection", err)
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
		skipIfTransientRealLLMInfraError(t, "GenerateText", err)
		t.Fatalf("GenerateText returned error: %v", err)
	}

	for attempt := 0; attempt < 2 && !hasAcceptableSmokeResponse(response); attempt++ {
		time.Sleep(250 * time.Millisecond)
		response, err = llm.GenerateText(
			ctx,
			"You are validating a production-like integration test. Reply with one short sentence and include the token THINGINDEX_REAL_LLM_OK exactly once.",
			"Confirm that the provider is reachable and include THINGINDEX_REAL_LLM_OK exactly once.",
		)
		if err != nil {
			skipIfTransientRealLLMInfraError(t, "GenerateText retry", err)
			t.Fatalf("GenerateText retry returned error: %v", err)
		}
	}

	if !hasAcceptableSmokeResponse(response) {
		t.Fatalf("expected a non-empty text response from the provider, got %q", response)
	}
}

func TestRealLLMModelDiscovery(t *testing.T) {
	cfg := testutil.RequireRealLLMModelDiscovery(t)

	if !SupportsLLMModelDiscovery(cfg.ProviderType) {
		t.Skipf("real model discovery is not implemented for provider %s", cfg.ProviderType)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	models, err := ListLLMModels(ctx, cfg.ProviderType, cfg.Credentials)
	if err != nil {
		skipIfTransientRealLLMInfraError(t, "ListLLMModels", err)
		t.Fatalf("ListLLMModels returned error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected at least one discovered model")
	}
}

func skipIfTransientRealLLMInfraError(t *testing.T, operation string, err error) {
	t.Helper()

	if isRetriableLLMInfraError(err) {
		t.Skipf("%s skipped due to transient provider infrastructure error: %v", operation, err)
	}
}

func hasAcceptableSmokeResponse(response string) bool {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return false
	}

	if strings.Contains(strings.ToUpper(trimmed), "THINGINDEX_REAL_LLM_OK") {
		return true
	}

	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}

	return false
}
