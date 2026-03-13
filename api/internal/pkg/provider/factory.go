package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/storage"
)

type openAICompatibleLLMClient struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	model      string
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float64                 `json:"temperature,omitempty"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type StorageProviderOptions struct {
	EndpointValidator *netsec.StorageEndpointValidator
}

func NewLLMProvider(providerType string, credentials map[string]string) (domain.LLMService, error) {
	apiKey := strings.TrimSpace(credentials["api_key"])
	if apiKey == "" {
		return nil, errors.New("missing api_key in credentials")
	}

	endpoint, model, err := llmEndpointAndModel(providerType, credentials)
	if err != nil {
		return nil, err
	}

	return &openAICompatibleLLMClient{
		httpClient: &http.Client{Timeout: 45 * time.Second},
		endpoint:   endpoint,
		apiKey:     apiKey,
		model:      model,
	}, nil
}

func NewStorageProvider(providerType string, credentials map[string]string, options StorageProviderOptions) (domain.StorageService, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))

	switch normalizedProvider {
	case domain.ProviderMinio:
		return buildS3CompatibleStorage(normalizedProvider, credentials, options)
	case domain.ProviderAWSS3:
		return buildS3CompatibleStorage(normalizedProvider, credentials, options)
	case domain.ProviderCloudflare:
		return buildS3CompatibleStorage(normalizedProvider, credentials, options)
	case domain.ProviderDOSpaces:
		return buildS3CompatibleStorage(normalizedProvider, credentials, options)
	case domain.ProviderWasabi:
		return buildS3CompatibleStorage(normalizedProvider, credentials, options)
	case domain.ProviderB2:
		return buildS3CompatibleStorage(normalizedProvider, credentials, options)
	default:
		return nil, fmt.Errorf("storage provider %s is not implemented", normalizedProvider)
	}
}

func buildS3CompatibleStorage(providerType string, credentials map[string]string, options StorageProviderOptions) (domain.StorageService, error) {
	endpoint, accessKey, secretKey, bucketName, useSSL, err := resolveS3CompatibleConfig(providerType, credentials)
	if err != nil {
		return nil, err
	}

	return storage.NewValidatedMinioStorage(endpoint, accessKey, secretKey, bucketName, useSSL, options.EndpointValidator)
}

func resolveS3CompatibleConfig(providerType string, credentials map[string]string) (string, string, string, string, bool, error) {
	accessKey := strings.TrimSpace(credentials["access_key"])
	secretKey := strings.TrimSpace(credentials["secret_key"])
	bucketName := strings.TrimSpace(credentials["bucket_name"])
	useSSL := !strings.EqualFold(strings.TrimSpace(credentials["use_ssl"]), "false")

	switch providerType {
	case domain.ProviderMinio:
		endpoint := strings.TrimSpace(credentials["endpoint"])
		if endpoint == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return "", "", "", "", false, errors.New("missing required minio credentials")
		}
		return endpoint, accessKey, secretKey, bucketName, useSSL, nil

	case domain.ProviderAWSS3:
		region := strings.TrimSpace(credentials["region"])
		endpoint := strings.TrimSpace(credentials["endpoint"])
		if region == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return "", "", "", "", false, errors.New("missing required aws s3 credentials")
		}
		if endpoint == "" {
			endpoint = fmt.Sprintf("s3.%s.amazonaws.com", region)
		}
		return endpoint, accessKey, secretKey, bucketName, true, nil

	case domain.ProviderCloudflare:
		accountID := strings.TrimSpace(credentials["account_id"])
		if accountID == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return "", "", "", "", false, errors.New("missing required cloudflare r2 credentials")
		}
		endpoint := strings.TrimSpace(credentials["endpoint"])
		if endpoint == "" {
			endpoint = fmt.Sprintf("%s.r2.cloudflarestorage.com", accountID)
		}
		return endpoint, accessKey, secretKey, bucketName, true, nil
	case domain.ProviderDOSpaces:
		region := strings.TrimSpace(credentials["region"])
		endpoint := strings.TrimSpace(credentials["endpoint"])
		if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return "", "", "", "", false, errors.New("missing required digitalocean spaces credentials")
		}
		return endpoint, accessKey, secretKey, bucketName, true, nil
	case domain.ProviderWasabi:
		region := strings.TrimSpace(credentials["region"])
		endpoint := strings.TrimSpace(credentials["endpoint"])
		if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return "", "", "", "", false, errors.New("missing required wasabi credentials")
		}
		return endpoint, accessKey, secretKey, bucketName, true, nil
	case domain.ProviderB2:
		region := strings.TrimSpace(credentials["region"])
		endpoint := strings.TrimSpace(credentials["endpoint"])
		if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucketName == "" {
			return "", "", "", "", false, errors.New("missing required backblaze b2 credentials")
		}
		return endpoint, accessKey, secretKey, bucketName, true, nil
	default:
		return "", "", "", "", false, fmt.Errorf("storage provider %s is not implemented", providerType)
	}
}

func (c *openAICompatibleLLMClient) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	payload := chatCompletionRequest{
		Model: c.model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return "", fmt.Errorf("parse llm response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if completion.Error != nil && strings.TrimSpace(completion.Error.Message) != "" {
			return "", errors.New(completion.Error.Message)
		}
		return "", fmt.Errorf("llm request failed with status %d", resp.StatusCode)
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("llm response returned no choices")
	}

	return strings.TrimSpace(completion.Choices[0].Message.Content), nil
}

func llmEndpointAndModel(providerType string, credentials map[string]string) (string, string, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))
	baseURL := strings.TrimRight(strings.TrimSpace(credentials["base_url"]), "/")
	model := strings.TrimSpace(credentials["model"])

	switch normalizedProvider {
	case domain.ProviderOpenAI:
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		if model == "" {
			model = "gpt-4o-mini"
		}
		return baseURL + "/chat/completions", model, nil
	case domain.ProviderDeepSeek:
		if baseURL == "" {
			baseURL = "https://api.deepseek.com"
		}
		if model == "" {
			model = "deepseek-chat"
		}
		return baseURL + "/chat/completions", model, nil
	case domain.ProviderGrok:
		if baseURL == "" {
			baseURL = "https://api.x.ai/v1"
		}
		if model == "" {
			model = "grok-2-latest"
		}
		return baseURL + "/chat/completions", model, nil
	default:
		if baseURL == "" || model == "" {
			return "", "", fmt.Errorf("provider %s requires base_url and model", normalizedProvider)
		}
		return baseURL + "/chat/completions", model, nil
	}
}
