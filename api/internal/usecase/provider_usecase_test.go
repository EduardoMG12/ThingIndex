package usecase

import (
	"context"
	"testing"

	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/pkg/netsec"
)

func TestValidateCredentialsRejectsPrivateLLMBaseURL(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "secret",
		"base_url": "https://127.0.0.1:11434/v1",
	})
	if err == nil {
		t.Fatal("expected private llm base url to be rejected")
	}
}

func TestValidateCredentialsAcceptsPublicHTTPSLLMBaseURL(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "secret",
		"base_url": "https://8.8.8.8/v1",
	})
	if err != nil {
		t.Fatalf("expected public https llm base url to be accepted, got %v", err)
	}
}

func TestValidateCredentialsRejectsHTTPBaseURLOutsideDevelopment(t *testing.T) {
	t.Parallel()

	useCase := &providerUseCase{
		cfg:               &config.Config{Environment: "production"},
		endpointValidator: netsec.NewStorageEndpointValidator(false),
	}

	err := useCase.validateCredentials(context.Background(), domain.FamilyLLM, domain.ProviderOpenAI, map[string]string{
		"api_key":  "secret",
		"base_url": "http://8.8.8.8/v1",
	})
	if err == nil {
		t.Fatal("expected http llm base url to be rejected outside development")
	}
}
