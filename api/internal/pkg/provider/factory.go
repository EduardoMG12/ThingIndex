package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/pkg/netsec"
	"thing-index/internal/pkg/storage"
)

type openAICompatibleLLMClient struct {
	httpClient *http.Client
	provider   string
	endpoint   string
	apiKey     string
	model      string
	pricing    domain.LLMPricing
	hasPricing bool
}

type ollamaNativeLLMClient struct {
	httpClient *http.Client
	provider   string
	endpoint   string
	apiKey     string
	model      string
	pricing    domain.LLMPricing
	hasPricing bool
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float64                 `json:"temperature,omitempty"`
	MaxTokens   int                     `json:"max_tokens,omitempty"`
	Stream      bool                    `json:"stream,omitempty"`
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

type chatCompletionStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type ollamaChatRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
	Stream   bool                    `json:"stream,omitempty"`
	Options  *ollamaChatOptions      `json:"options,omitempty"`
}

type ollamaChatOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
}

type ollamaChatResponse struct {
	Message *chatCompletionMessage `json:"message,omitempty"`
	Done    bool                   `json:"done"`
	Error   string                 `json:"error,omitempty"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type StorageProviderOptions struct {
	EndpointValidator *netsec.StorageEndpointValidator
}

type llmProviderDefinition struct {
	defaultBaseURL   string
	defaultModel     string
	requireBaseURL   bool
	requireModel     bool
	allowEmptyAPIKey bool
}

var llmProviderDefinitions = map[string]llmProviderDefinition{
	domain.ProviderOpenAI: {
		defaultBaseURL: "https://api.openai.com/v1",
		defaultModel:   "gpt-4o-mini",
	},
	domain.ProviderDeepSeek: {
		defaultBaseURL: "https://api.deepseek.com",
		defaultModel:   "deepseek-chat",
	},
	domain.ProviderGrok: {
		defaultBaseURL: "https://api.x.ai/v1",
		defaultModel:   "grok-2-latest",
	},
	domain.ProviderOpenRouter: {
		defaultBaseURL: "https://openrouter.ai/api/v1",
		defaultModel:   "openai/gpt-4o-mini",
	},
	domain.ProviderOllama: {
		defaultModel:     "llama3.1:8b",
		allowEmptyAPIKey: true,
	},
	domain.ProviderGemini: {
		requireBaseURL: true,
		requireModel:   true,
	},
	domain.ProviderClaude: {
		requireBaseURL: true,
		requireModel:   true,
	},
}

var newLLMHTTPClient = func() *http.Client {
	return &http.Client{Timeout: 45 * time.Second}
}

func NewLLMHTTPClientForTests(factory func() *http.Client) func() *http.Client {
	previous := newLLMHTTPClient
	if factory == nil {
		return previous
	}
	newLLMHTTPClient = factory
	return previous
}

func NewLLMProvider(providerType string, credentials map[string]string) (domain.LLMService, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))
	definition, ok := llmProviderDefinitions[normalizedProvider]
	if !ok {
		return nil, fmt.Errorf("provider %s is not implemented", normalizedProvider)
	}

	apiKey := strings.TrimSpace(credentials["api_key"])
	if apiKey == "" && !definition.allowEmptyAPIKey {
		return nil, errors.New("missing api_key in credentials")
	}

	endpoint, model, err := llmEndpointAndModel(normalizedProvider, credentials)
	if err != nil {
		return nil, err
	}

	pricing, hasPricing := llmPricingProfile(normalizedProvider, model, credentials)

	if normalizedProvider == domain.ProviderOllama && isOllamaNativeEndpoint(endpoint) {
		return &ollamaNativeLLMClient{
			httpClient: newLLMHTTPClient(),
			provider:   normalizedProvider,
			endpoint:   endpoint,
			apiKey:     apiKey,
			model:      model,
			pricing:    pricing,
			hasPricing: hasPricing,
		}, nil
	}

	return &openAICompatibleLLMClient{
		httpClient: newLLMHTTPClient(),
		provider:   normalizedProvider,
		endpoint:   endpoint,
		apiKey:     apiKey,
		model:      model,
		pricing:    pricing,
		hasPricing: hasPricing,
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
	case domain.ProviderGCPStorage:
		return buildGCSStorage(credentials)
	default:
		return nil, fmt.Errorf("storage provider %s is not implemented", normalizedProvider)
	}
}

func buildS3CompatibleStorage(providerType string, credentials map[string]string, options StorageProviderOptions) (domain.StorageService, error) {
	endpoint, accessKey, secretKey, bucketName, useSSL, err := resolveS3CompatibleConfig(providerType, credentials)
	if err != nil {
		return nil, err
	}

	if options.EndpointValidator == nil {
		return storage.NewMinioStorage(endpoint, accessKey, secretKey, bucketName, useSSL)
	}

	return storage.NewValidatedMinioStorage(endpoint, accessKey, secretKey, bucketName, useSSL, options.EndpointValidator)
}

func buildGCSStorage(credentials map[string]string) (domain.StorageService, error) {
	projectID, bucketName, location, credentialsJSON, emulatorHost, err := resolveGCSConfig(credentials)
	if err != nil {
		return nil, err
	}

	return storage.NewGCSStorage(projectID, bucketName, location, credentialsJSON, emulatorHost)
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

func resolveGCSConfig(credentials map[string]string) (string, string, string, string, string, error) {
	projectID := strings.TrimSpace(credentials["project_id"])
	bucketName := strings.TrimSpace(credentials["bucket_name"])
	location := strings.TrimSpace(credentials["location"])
	credentialsJSON := strings.TrimSpace(credentials["credentials_json"])
	emulatorHost := strings.TrimSpace(credentials["emulator_host"])

	if projectID == "" || bucketName == "" {
		return "", "", "", "", "", errors.New("missing required gcs credentials")
	}
	if credentialsJSON == "" && emulatorHost == "" {
		return "", "", "", "", "", errors.New("missing required gcs credentials")
	}

	return projectID, bucketName, location, credentialsJSON, emulatorHost, nil
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

	completion, err := c.createChatCompletion(ctx, payload)
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("llm response returned no choices")
	}

	return strings.TrimSpace(completion.Choices[0].Message.Content), nil
}

func ValidateLLMConnection(ctx context.Context, providerType string, credentials map[string]string) error {
	service, err := NewLLMProvider(providerType, credentials)
	if err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	switch client := service.(type) {
	case *openAICompatibleLLMClient:
		_, err = client.createChatCompletion(timeoutCtx, chatCompletionRequest{
			Model: client.model,
			Messages: []chatCompletionMessage{
				{Role: "system", Content: "Reply with OK."},
				{Role: "user", Content: "ping"},
			},
			Temperature: 0,
			MaxTokens:   1,
		})
		return err
	case *ollamaNativeLLMClient:
		_, err = client.createChat(timeoutCtx, ollamaChatRequest{
			Model: client.model,
			Messages: []chatCompletionMessage{
				{Role: "system", Content: "Reply with OK."},
				{Role: "user", Content: "ping"},
			},
			Stream: false,
			Options: &ollamaChatOptions{
				Temperature: floatPtr(0),
				NumPredict:  intPtr(1),
			},
		})
		return err
	default:
		return errors.New("llm validation is not supported for this provider")
	}
}

func ValidateStorageConnection(ctx context.Context, providerType string, credentials map[string]string, options StorageProviderOptions) error {
	service, err := NewStorageProvider(providerType, credentials, options)
	if err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	bucketProbeObject := strings.TrimSpace(credentials["bucket_probe_object"])
	if bucketProbeObject == "" {
		return nil
	}

	_, err = service.FileExists(timeoutCtx, bucketProbeObject)
	return err
}

func SupportsLLMModelDiscovery(providerType string) bool {
	switch strings.ToUpper(strings.TrimSpace(providerType)) {
	case domain.ProviderOpenAI,
		domain.ProviderGemini,
		domain.ProviderDeepSeek,
		domain.ProviderGrok,
		domain.ProviderClaude,
		domain.ProviderOllama,
		domain.ProviderOpenRouter:
		return true
	default:
		return false
	}
}

func ListLLMModels(ctx context.Context, providerType string, credentials map[string]string) ([]string, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))

	if !SupportsLLMModelDiscovery(normalizedProvider) {
		return nil, fmt.Errorf("model discovery is not implemented for provider %s", normalizedProvider)
	}

	switch normalizedProvider {
	case domain.ProviderOllama:
		return listOllamaModels(ctx, credentials)
	default:
		return listOpenAICompatibleModels(ctx, normalizedProvider, credentials)
	}
}

func listOllamaModels(ctx context.Context, credentials map[string]string) ([]string, error) {
	baseURL := strings.TrimSpace(credentials["base_url"])
	if baseURL == "" {
		if strings.TrimSpace(credentials["api_key"]) != "" {
			baseURL = "https://ollama.com/api"
		} else {
			baseURL = "http://localhost:11434/v1"
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaTagsEndpoint(baseURL), nil)
	if err != nil {
		return nil, err
	}
	if apiKey := strings.TrimSpace(credentials["api_key"]); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := newLLMHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, readOllamaErrorBody(resp.StatusCode, respBody)
	}

	var payload ollamaTagsResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("parse ollama models response: %w", err)
	}

	models := make([]string, 0, len(payload.Models))
	seen := make(map[string]struct{}, len(payload.Models))
	for _, item := range payload.Models {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}

	return models, nil
}

func listOpenAICompatibleModels(ctx context.Context, providerType string, credentials map[string]string) ([]string, error) {
	endpoint, err := llmModelsEndpoint(providerType, credentials)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if apiKey := strings.TrimSpace(credentials["api_key"]); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := newLLMHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, readOpenAIErrorBody(resp.StatusCode, respBody)
	}

	var payload openAIModelsResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("parse llm models response: %w", err)
	}

	models := make([]string, 0, len(payload.Data))
	seen := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		models = append(models, modelID)
	}

	return models, nil
}

func (c *openAICompatibleLLMClient) createChatCompletion(ctx context.Context, payload chatCompletionRequest) (*chatCompletionResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if completion.Error != nil && strings.TrimSpace(completion.Error.Message) != "" {
			return nil, errors.New(completion.Error.Message)
		}
		return nil, fmt.Errorf("llm request failed with status %d", resp.StatusCode)
	}

	return &completion, nil
}

func (c *openAICompatibleLLMClient) StreamText(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string) error) error {
	payload := chatCompletionRequest{
		Model: c.model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		Stream:      true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return readErr
		}

		var completion chatCompletionResponse
		if err := json.Unmarshal(respBody, &completion); err == nil && completion.Error != nil && strings.TrimSpace(completion.Error.Message) != "" {
			return errors.New(completion.Error.Message)
		}
		return fmt.Errorf("llm stream request failed with status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			return nil
		}

		var chunk chatCompletionStreamResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && strings.TrimSpace(chunk.Error.Message) != "" {
			return errors.New(chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if err := onDelta(delta); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (c *ollamaNativeLLMClient) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	response, err := c.createChat(ctx, ollamaChatRequest{
		Model: c.model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: false,
		Options: &ollamaChatOptions{
			Temperature: floatPtr(0.2),
		},
	})
	if err != nil {
		return "", err
	}
	if response.Message == nil {
		return "", errors.New("ollama response returned no message")
	}

	return strings.TrimSpace(response.Message.Content), nil
}

func (c *ollamaNativeLLMClient) StreamText(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string) error) error {
	body, err := json.Marshal(ollamaChatRequest{
		Model: c.model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: true,
		Options: &ollamaChatOptions{
			Temperature: floatPtr(0.2),
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readOllamaErrorResponse(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if strings.TrimSpace(chunk.Error) != "" {
			return errors.New(strings.TrimSpace(chunk.Error))
		}
		if chunk.Message == nil {
			if chunk.Done {
				return nil
			}
			continue
		}

		delta := chunk.Message.Content
		if delta == "" {
			if chunk.Done {
				return nil
			}
			continue
		}
		if err := onDelta(delta); err != nil {
			return err
		}
		if chunk.Done {
			return nil
		}
	}

	return scanner.Err()
}

func (c *ollamaNativeLLMClient) createChat(ctx context.Context, payload ollamaChatRequest) (*ollamaChatResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, readOllamaErrorResponse(resp)
	}

	var response ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("parse ollama response: %w", err)
	}
	if strings.TrimSpace(response.Error) != "" {
		return nil, errors.New(strings.TrimSpace(response.Error))
	}

	return &response, nil
}

func readOllamaErrorResponse(resp *http.Response) error {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return readOllamaErrorBody(resp.StatusCode, respBody)
}

func readOllamaErrorBody(statusCode int, respBody []byte) error {
	var apiErr ollamaChatResponse
	if err := json.Unmarshal(respBody, &apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
		return errors.New(strings.TrimSpace(apiErr.Error))
	}

	var generic map[string]any
	if err := json.Unmarshal(respBody, &generic); err == nil {
		if message, ok := generic["message"].(string); ok && strings.TrimSpace(message) != "" {
			return errors.New(strings.TrimSpace(message))
		}
	}

	return fmt.Errorf("ollama request failed with status %d", statusCode)
}

func readOpenAIErrorBody(statusCode int, respBody []byte) error {
	var apiErr openAIModelsResponse
	if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error != nil && strings.TrimSpace(apiErr.Error.Message) != "" {
		return errors.New(strings.TrimSpace(apiErr.Error.Message))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err == nil && completion.Error != nil && strings.TrimSpace(completion.Error.Message) != "" {
		return errors.New(strings.TrimSpace(completion.Error.Message))
	}

	var generic map[string]any
	if err := json.Unmarshal(respBody, &generic); err == nil {
		if message, ok := generic["message"].(string); ok && strings.TrimSpace(message) != "" {
			return errors.New(strings.TrimSpace(message))
		}
	}

	return fmt.Errorf("llm request failed with status %d", statusCode)
}

func (c *openAICompatibleLLMClient) ProviderType() string {
	return c.provider
}

func (c *openAICompatibleLLMClient) ModelName() string {
	return c.model
}

func (c *openAICompatibleLLMClient) Pricing() (domain.LLMPricing, bool) {
	return c.pricing, c.hasPricing
}

func (c *ollamaNativeLLMClient) ProviderType() string {
	return c.provider
}

func (c *ollamaNativeLLMClient) ModelName() string {
	return c.model
}

func (c *ollamaNativeLLMClient) Pricing() (domain.LLMPricing, bool) {
	return c.pricing, c.hasPricing
}

func llmEndpointAndModel(providerType string, credentials map[string]string) (string, string, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))
	definition, ok := llmProviderDefinitions[normalizedProvider]
	if !ok {
		return "", "", fmt.Errorf("provider %s is not implemented", normalizedProvider)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(credentials["base_url"]), "/")
	model := strings.TrimSpace(credentials["model"])

	if baseURL == "" {
		if normalizedProvider == domain.ProviderOllama {
			if strings.TrimSpace(credentials["api_key"]) != "" {
				baseURL = "https://ollama.com/api"
			} else {
				baseURL = "http://localhost:11434/v1"
			}
		} else {
			baseURL = definition.defaultBaseURL
		}
	}
	if model == "" {
		model = definition.defaultModel
	}

	if definition.requireBaseURL && baseURL == "" {
		return "", "", fmt.Errorf("provider %s requires base_url", normalizedProvider)
	}
	if definition.requireModel && model == "" {
		return "", "", fmt.Errorf("provider %s requires model", normalizedProvider)
	}
	if baseURL == "" || model == "" {
		return "", "", fmt.Errorf("provider %s requires base_url and model", normalizedProvider)
	}

	if normalizedProvider == domain.ProviderOllama {
		return ollamaEndpoint(baseURL), model, nil
	}

	return baseURL + "/chat/completions", model, nil
}

func llmModelsEndpoint(providerType string, credentials map[string]string) (string, error) {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))
	definition, ok := llmProviderDefinitions[normalizedProvider]
	if !ok {
		return "", fmt.Errorf("provider %s is not implemented", normalizedProvider)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(credentials["base_url"]), "/")
	if baseURL == "" {
		if normalizedProvider == domain.ProviderOllama {
			if strings.TrimSpace(credentials["api_key"]) != "" {
				baseURL = "https://ollama.com/api"
			} else {
				baseURL = "http://localhost:11434/v1"
			}
		} else {
			baseURL = definition.defaultBaseURL
		}
	}
	if baseURL == "" {
		return "", fmt.Errorf("provider %s requires base_url", normalizedProvider)
	}

	return openAIModelsEndpoint(baseURL), nil
}

func ollamaEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed + "/chat/completions"
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	path := strings.TrimRight(parsed.Path, "/")
	if host == "ollama.com" || host == "www.ollama.com" {
		parsed.Path = "/api/chat"
		return strings.TrimRight(parsed.String(), "/")
	}
	if path == "/api" || path == "/api/chat" {
		parsed.Path = "/api/chat"
		return strings.TrimRight(parsed.String(), "/")
	}

	return trimmed + "/chat/completions"
}

func ollamaTagsEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed + "/api/tags"
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	path := strings.TrimRight(parsed.Path, "/")
	if host == "ollama.com" || host == "www.ollama.com" {
		parsed.Path = "/api/tags"
		return strings.TrimRight(parsed.String(), "/")
	}
	if path == "/v1" || path == "/api" || path == "/api/chat" || path == "/chat/completions" {
		parsed.Path = "/api/tags"
		return strings.TrimRight(parsed.String(), "/")
	}

	return trimmed + "/api/tags"
}

func openAIModelsEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		if strings.HasSuffix(trimmed, "/chat/completions") {
			return strings.TrimSuffix(trimmed, "/chat/completions") + "/models"
		}
		return trimmed + "/models"
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		parsed.Path = strings.TrimSuffix(path, "/chat/completions") + "/models"
	case strings.HasSuffix(path, "/models"):
		parsed.Path = path
	case path == "":
		parsed.Path = "/models"
	default:
		parsed.Path = path + "/models"
	}

	return strings.TrimRight(parsed.String(), "/")
}

func isOllamaNativeEndpoint(endpoint string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return (host == "ollama.com" || host == "www.ollama.com") && strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/api/chat")
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func llmPricingProfile(providerType, model string, credentials map[string]string) (domain.LLMPricing, bool) {
	if pricing, ok := pricingFromCredentials(credentials); ok {
		return pricing, true
	}

	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	switch providerType {
	case domain.ProviderOpenAI:
		switch normalizedModel {
		case "gpt-4o-mini":
			return domain.LLMPricing{
				InputCostPerMillionUSD:  0.15,
				OutputCostPerMillionUSD: 0.60,
			}, true
		case "gpt-4o":
			return domain.LLMPricing{
				InputCostPerMillionUSD:  2.50,
				OutputCostPerMillionUSD: 10.00,
			}, true
		}
	case domain.ProviderDeepSeek:
		switch normalizedModel {
		case "deepseek-chat":
			return domain.LLMPricing{
				InputCostPerMillionUSD:  0.27,
				OutputCostPerMillionUSD: 1.10,
			}, true
		}
	case domain.ProviderOpenRouter, domain.ProviderOllama:
		return domain.LLMPricing{}, false
	}

	return domain.LLMPricing{}, false
}

func pricingFromCredentials(credentials map[string]string) (domain.LLMPricing, bool) {
	input, inputOk := parsePricingValue(credentials["input_cost_per_million_usd"])
	output, outputOk := parsePricingValue(credentials["output_cost_per_million_usd"])
	if !inputOk || !outputOk {
		return domain.LLMPricing{}, false
	}

	return domain.LLMPricing{
		InputCostPerMillionUSD:  input,
		OutputCostPerMillionUSD: output,
	}, true
}

func parsePricingValue(value string) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}

	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}

	return parsed, true
}
