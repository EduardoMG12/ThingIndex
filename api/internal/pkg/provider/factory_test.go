package provider

import (
	"testing"

	"thing-index/internal/domain"
)

func TestLLMEndpointAndModelUsesOpenRouterDefaults(t *testing.T) {
	t.Parallel()

	endpoint, model, err := llmEndpointAndModel(domain.ProviderOpenRouter, map[string]string{})
	if err != nil {
		t.Fatalf("expected openrouter defaults, got error: %v", err)
	}

	if endpoint != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("unexpected openrouter endpoint: %s", endpoint)
	}
	if model != "openai/gpt-4o-mini" {
		t.Fatalf("unexpected openrouter model: %s", model)
	}
}

func TestLLMEndpointAndModelUsesOllamaDefaults(t *testing.T) {
	t.Parallel()

	endpoint, model, err := llmEndpointAndModel(domain.ProviderOllama, map[string]string{})
	if err != nil {
		t.Fatalf("expected ollama defaults, got error: %v", err)
	}

	if endpoint != "http://localhost:11434/v1/chat/completions" {
		t.Fatalf("unexpected ollama endpoint: %s", endpoint)
	}
	if model != "llama3.1:8b" {
		t.Fatalf("unexpected ollama model: %s", model)
	}
}

func TestLLMEndpointAndModelRejectsGeminiWithoutBridgeConfiguration(t *testing.T) {
	t.Parallel()

	if _, _, err := llmEndpointAndModel(domain.ProviderGemini, map[string]string{}); err == nil {
		t.Fatal("expected gemini without base_url and model to be rejected")
	}
}
