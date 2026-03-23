package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

type factoryRoundTripFunc func(*http.Request) (*http.Response, error)

func (f factoryRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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

func TestLLMEndpointAndModelUsesOllamaCloudDefaultWhenAPIKeyIsPresent(t *testing.T) {
	t.Parallel()

	endpoint, model, err := llmEndpointAndModel(domain.ProviderOllama, map[string]string{
		"api_key": "secret",
		"model":   "deepseek-v3.1:671b",
	})
	if err != nil {
		t.Fatalf("expected ollama cloud defaults, got error: %v", err)
	}

	if endpoint != "https://ollama.com/api/chat" {
		t.Fatalf("unexpected ollama cloud endpoint: %s", endpoint)
	}
	if model != "deepseek-v3.1:671b" {
		t.Fatalf("unexpected ollama cloud model: %s", model)
	}
}

func TestLLMEndpointAndModelNormalizesExplicitOllamaCloudV1BaseURL(t *testing.T) {
	t.Parallel()

	endpoint, model, err := llmEndpointAndModel(domain.ProviderOllama, map[string]string{
		"api_key":  "secret",
		"base_url": "https://ollama.com/v1",
		"model":    "deepseek-v3.1:671b",
	})
	if err != nil {
		t.Fatalf("expected explicit ollama cloud v1 base url to be normalized, got error: %v", err)
	}

	if endpoint != "https://ollama.com/api/chat" {
		t.Fatalf("unexpected normalized ollama cloud endpoint: %s", endpoint)
	}
	if model != "deepseek-v3.1:671b" {
		t.Fatalf("unexpected ollama cloud model: %s", model)
	}
}

func TestLLMEndpointAndModelRejectsGeminiWithoutBridgeConfiguration(t *testing.T) {
	t.Parallel()

	if _, _, err := llmEndpointAndModel(domain.ProviderGemini, map[string]string{}); err == nil {
		t.Fatal("expected gemini without base_url and model to be rejected")
	}
}

func TestSupportsLLMModelDiscoveryRecognizesOpenAICompatibleProviders(t *testing.T) {
	t.Parallel()

	for _, providerType := range []string{
		domain.ProviderOpenAI,
		domain.ProviderDeepSeek,
		domain.ProviderGrok,
		domain.ProviderGemini,
		domain.ProviderClaude,
		domain.ProviderOpenRouter,
		domain.ProviderOllama,
	} {
		if !SupportsLLMModelDiscovery(providerType) {
			t.Fatalf("expected provider %s to support discovery", providerType)
		}
	}
}

func TestListLLMModelsUsesOpenAICompatibleModelsEndpoint(t *testing.T) {
	originalFactory := NewLLMHTTPClientForTests(nil)
	defer NewLLMHTTPClientForTests(originalFactory)

	NewLLMHTTPClientForTests(func() *http.Client {
		return &http.Client{
			Transport: factoryRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected method %s", req.Method)
				}
				if req.URL.String() != "https://api.openai.com/v1/models" {
					t.Fatalf("unexpected models endpoint %q", req.URL.String())
				}
				if got := req.Header.Get("Authorization"); got != "Bearer secret" {
					t.Fatalf("unexpected auth header %q", got)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data":[
							{"id":"gpt-4o-mini"},
							{"id":" gpt-4.1-mini "},
							{"id":"gpt-4o-mini"},
							{"id":""}
						]
					}`)),
					Header: make(http.Header),
				}, nil
			}),
		}
	})

	models, err := ListLLMModels(context.Background(), domain.ProviderOpenAI, map[string]string{
		"api_key": "secret",
	})
	if err != nil {
		t.Fatalf("expected models discovery to succeed, got %v", err)
	}

	if len(models) != 2 || models[0] != "gpt-4o-mini" || models[1] != "gpt-4.1-mini" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestOpenAIModelsEndpointNormalizesChatCompletionsURL(t *testing.T) {
	t.Parallel()

	endpoint := openAIModelsEndpoint("https://gateway.example.com/v1/chat/completions")
	if endpoint != "https://gateway.example.com/v1/models" {
		t.Fatalf("unexpected endpoint %q", endpoint)
	}
}

func TestResolveGCSConfigAcceptsEmulatorHostWithoutCredentialsJSON(t *testing.T) {
	t.Parallel()

	projectID, bucketName, location, credentialsJSON, emulatorHost, err := resolveGCSConfig(map[string]string{
		"project_id":    "thingindex-dev",
		"bucket_name":   "thingindex-docs",
		"location":      "us-central1",
		"emulator_host": "127.0.0.1:4443",
	})
	if err != nil {
		t.Fatalf("expected gcs config to resolve, got %v", err)
	}
	if projectID != "thingindex-dev" || bucketName != "thingindex-docs" {
		t.Fatalf("unexpected gcs config: project=%q bucket=%q", projectID, bucketName)
	}
	if location != "us-central1" {
		t.Fatalf("expected location to be preserved, got %q", location)
	}
	if credentialsJSON != "" {
		t.Fatalf("expected empty credentials json, got %q", credentialsJSON)
	}
	if emulatorHost != "127.0.0.1:4443" {
		t.Fatalf("unexpected emulator host %q", emulatorHost)
	}
}

func TestResolveGCSConfigRejectsMissingCredentialsAndEmulatorHost(t *testing.T) {
	t.Parallel()

	_, _, _, _, _, err := resolveGCSConfig(map[string]string{
		"project_id":  "thingindex-dev",
		"bucket_name": "thingindex-docs",
	})
	if err == nil {
		t.Fatal("expected missing gcs credentials to be rejected")
	}
}
