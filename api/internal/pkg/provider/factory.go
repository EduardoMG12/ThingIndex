package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"thing-index/internal/domain"
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

type userStorageClient struct {
	provider   string
	accessKey  string
	secretKey  string
	endpoint   string
	bucketName string
	region     string
}

func (c *userStorageClient) UploadFile(ctx context.Context, file io.Reader, objectSize int64, contentType string, objectName string) (string, error) {
	return fmt.Sprintf("https://%s/%s/%s", c.endpoint, c.bucketName, objectName), nil
}

func NewStorageProvider(providerType string, credentials map[string]string) (domain.StorageService, error) {
	required := []string{"access_key", "secret_key", "bucket_name"}
	for _, field := range required {
		if _, ok := credentials[field]; !ok {
			return nil, fmt.Errorf("missing %s in storage credentials", field)
		}
	}

	return &userStorageClient{
		provider:   providerType,
		accessKey:  credentials["access_key"],
		secretKey:  credentials["secret_key"],
		endpoint:   credentials["endpoint"],
		bucketName: credentials["bucket_name"],
		region:     credentials["region"],
	}, nil
}
