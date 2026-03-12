package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"thing-index/internal/domain"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/storage"
)

type genericLLMClient struct {
	provider string
	apiKey   string
	model    string
	baseURL  string
}

func (c *genericLLMClient) Summarize(ctx context.Context, text string) (string, error) {
	return fmt.Sprintf("[%s - %s] Placeholder summary for: %s", c.provider, c.model, text), nil
}

type StorageProviderOptions struct {
	EndpointValidator *netsec.StorageEndpointValidator
}

func NewLLMProvider(providerType string, credentials map[string]string) (domain.LLMService, error) {
	apiKey, ok := credentials["api_key"]
	if !ok {
		return nil, errors.New("missing api_key in credentials")
	}

	model := credentials["model"]
	if model == "" {
		switch providerType {
		case domain.ProviderOpenAI:
			model = "gpt-4o"
		case domain.ProviderGemini:
			model = "gemini-1.5-pro"
		case domain.ProviderDeepSeek:
			model = "deepseek-chat"
		}
	}

	return &genericLLMClient{
		provider: providerType,
		apiKey:   apiKey,
		model:    model,
		baseURL:  credentials["base_url"],
	}, nil
}

func NewStorageProvider(providerType string, credentials map[string]string, options StorageProviderOptions) (domain.StorageService, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))

	switch normalizedProvider {
	case domain.ProviderMinio:
		endpoint := strings.TrimSpace(credentials["endpoint"])
		accessKey := strings.TrimSpace(credentials["access_key"])
		secretKey := strings.TrimSpace(credentials["secret_key"])
		bucketName := strings.TrimSpace(credentials["bucket_name"])
		useSSL := strings.EqualFold(strings.TrimSpace(credentials["use_ssl"]), "true")

		if endpoint == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return nil, errors.New("missing required minio credentials")
		}

		return storage.NewValidatedMinioStorage(endpoint, accessKey, secretKey, bucketName, useSSL, options.EndpointValidator)
	default:
		return nil, fmt.Errorf("storage provider %s is not implemented", normalizedProvider)
	}
}
