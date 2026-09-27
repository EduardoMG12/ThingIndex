package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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

	// Consumo da ultima chamada. Protegido por mutex porque o mesmo cliente e
	// compartilhado entre as goroutines do enriquecimento -- sem isso o
	// go test -race acusaria, e a metrica de custo sairia corrompida sob carga.
	usageMu       sync.Mutex
	lastUsage     tokenUsage
	usageReported bool
}

// LastTokenUsage implementa domain.TokenUsageReporter.
func (c *openAICompatibleLLMClient) LastTokenUsage() (int, int, int, bool) {
	c.usageMu.Lock()
	defer c.usageMu.Unlock()

	if !c.usageReported {
		return 0, 0, 0, false
	}
	return c.lastUsage.PromptTokens, c.lastUsage.CompletionTokens, c.lastUsage.Total(), true
}

// registrarConsumo guarda o que o provedor reportou.
//
// Resposta sem `usage` NAO apaga o registro anterior nem marca como reportado:
// distinguir "nao informou" de "informou zero" e o que impede a metrica de
// custo mentir por omissao do provedor.
func (c *openAICompatibleLLMClient) registrarConsumo(usage *tokenUsage) {
	if usage == nil {
		return
	}

	c.usageMu.Lock()
	defer c.usageMu.Unlock()
	c.lastUsage = *usage
	c.usageReported = true
}

type ollamaNativeLLMClient struct {
	httpClient *http.Client
	provider   string
	endpoint   string
	apiKey     string
	model      string
	pricing    domain.LLMPricing
	hasPricing bool

	usageMu       sync.Mutex
	lastUsage     tokenUsage
	usageReported bool
}

// LastTokenUsage implementa domain.TokenUsageReporter. O /api/chat do Ollama
// devolve prompt_eval_count e eval_count no pedaco final; sem le-los, o
// consumo virava estimativa por runas/4 e um prompt truncado pelo contexto do
// modelo ficava invisivel.
func (c *ollamaNativeLLMClient) LastTokenUsage() (int, int, int, bool) {
	c.usageMu.Lock()
	defer c.usageMu.Unlock()
	if !c.usageReported {
		return 0, 0, 0, false
	}
	return c.lastUsage.PromptTokens, c.lastUsage.CompletionTokens, c.lastUsage.Total(), true
}

func (c *ollamaNativeLLMClient) registrarConsumo(chunk *ollamaChatResponse) {
	if chunk == nil || (chunk.PromptEvalCount == 0 && chunk.EvalCount == 0) {
		return
	}
	c.usageMu.Lock()
	defer c.usageMu.Unlock()
	c.lastUsage = tokenUsage{PromptTokens: chunk.PromptEvalCount, CompletionTokens: chunk.EvalCount}
	c.usageReported = true
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

// tokenUsage e o consumo reportado pelo provedor numa chamada.
//
// Existe porque a Metodologia do artigo promete avaliar custo operacional e o
// parser descartava o campo `usage` da resposta em silencio -- o dado chegava
// pela rede e sumia no unmarshal. Sem ele, qualquer afirmacao de custo seria
// estimativa por contagem local de caracteres, o tipo de numero que este
// trabalho critica.
//
// Ponteiro, e nao valor: provedor que nao reporta consumo (Ollama local, por
// exemplo) precisa ser distinguivel de provedor que reportou zero.
type tokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Total devolve o consumo somado, tolerando as duas omissoes comuns.
//
// Alguns provedores compativeis mandam as partes sem o total; somar no cliente
// evita metrica agregada zerada por omissao alheia. Quando o provedor informa o
// total, ele manda -- pode incluir tokens de sistema que as partes nao mostram.
//
// Receptor nulo devolve zero para o chamador nao precisar checar nil antes de
// somar.
func (u *tokenUsage) Total() int {
	if u == nil {
		return 0
	}
	if u.TotalTokens > 0 {
		return u.TotalTokens
	}
	return u.PromptTokens + u.CompletionTokens
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
	Usage *tokenUsage `json:"usage,omitempty"`
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
	// SEM omitempty: o padrao do /api/chat do Ollama e stream=true. Com
	// omitempty, `false` sumia do JSON, o servidor respondia em NDJSON e o
	// decoder lia so o primeiro pedaco -- cada resumo do enriquecimento virava
	// um token, e o pipeline caia em fallback sem nenhum erro registrado.
	Stream  bool               `json:"stream"`
	Options *ollamaChatOptions `json:"options,omitempty"`
}

type ollamaChatOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
}

type ollamaChatResponse struct {
	Message         *chatCompletionMessage `json:"message,omitempty"`
	Done            bool                   `json:"done"`
	Error           string                 `json:"error,omitempty"`
	PromptEvalCount int                    `json:"prompt_eval_count,omitempty"`
	EvalCount       int                    `json:"eval_count,omitempty"`
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

const defaultLLMGenerationTemperature = 0.0
const defaultLLMStreamingTemperature = 0.2
const llmValidationTimeout = 45 * time.Second
const maxLLMInfraAttempts = 2
const llmInfraRetryDelay = 250 * time.Millisecond

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

// LLMClientOptions carrega a politica de rede do cliente HTTP do LLM.
//
// EndpointValidator, quando presente, e aplicado na conexao a todo base_url
// informado pelo usuario: o IP e resolvido de novo no dial e IP privado e
// recusado (a menos que o validador tenha sido criado com a flag
// ALLOW_PRIVATE_STORAGE_ENDPOINTS, caso do Ollama local em desenvolvimento).
// Sem isso a validacao so acontecia no save, e um DNS rebinding entre a
// validacao e a conexao levava a API a requisitar a rede interna.
type LLMClientOptions struct {
	EndpointValidator *netsec.StorageEndpointValidator

	// responseHeaderTimeout substitui llmResponseHeaderTimeout; so os testes
	// do pacote usam, para exercitar o prazo com relogio curto.
	responseHeaderTimeout time.Duration
}

// llmResponseHeaderTimeout limita a espera pelo cabecalho da resposta.
//
// MOTIVO: o cliente tinha http.Client{Timeout: 45s}, que inclui a leitura do
// corpo; todo stream de chat com mais de 45 s era cortado com "Client.Timeout
// exceeded while reading body". O limite total agora vem do ctx do chamador
// (chat: chatStreamTimeout; indexacao: prazo por mensagem do consumidor;
// validacao e listagem de modelos: WithTimeout proprio).
const llmResponseHeaderTimeout = 45 * time.Second

// errLLMRedirectRecusado: redirect de um provedor de LLM e recusado. Seguir um
// 302 para http://10.0.0.x/... contornaria a validacao do base_url.
var errLLMRedirectRecusado = errors.New("llm provider redirect refused")

// newLLMHTTPClient e o gancho de teste: quando nao nil, substitui o cliente
// de producao. nil significa "cliente de producao".
var newLLMHTTPClient func() *http.Client

// NewLLMHTTPClientForTests instala factory como cliente HTTP do LLM e devolve
// o anterior; passar o anterior de volta restaura o estado (inclusive nil).
func NewLLMHTTPClientForTests(factory func() *http.Client) func() *http.Client {
	previous := newLLMHTTPClient
	newLLMHTTPClient = factory
	return previous
}

func resolveLLMClientOptions(opts []LLMClientOptions) LLMClientOptions {
	if len(opts) == 0 {
		return LLMClientOptions{}
	}
	return opts[0]
}

// buildLLMHTTPClient monta o cliente HTTP para falar com rawURL.
//
// userSuppliedBaseURL diz se o destino veio do base_url das credenciais. O
// endpoint padrao do provedor (inclusive o Ollama em localhost quando o
// usuario nao informa base_url) e configuracao do servidor e nao passa pelo
// validador -- manter isso preserva o Ollama local sem a flag de endpoints
// privados, exatamente como a validacao no save ja fazia.
func buildLLMHTTPClient(rawURL string, userSuppliedBaseURL bool, options LLMClientOptions) (*http.Client, error) {
	if newLLMHTTPClient != nil {
		return newLLMHTTPClient(), nil
	}

	headerTimeout := llmResponseHeaderTimeout
	if options.responseHeaderTimeout > 0 {
		headerTimeout = options.responseHeaderTimeout
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: headerTimeout,
		ExpectContinueTimeout: time.Second,
	}

	if options.EndpointValidator != nil && userSuppliedBaseURL {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil || parsed.Host == "" {
			return nil, fmt.Errorf("invalid llm endpoint: %q", rawURL)
		}
		defaultPort := "443"
		if strings.EqualFold(parsed.Scheme, "http") {
			defaultPort = "80"
		}
		dialContext, err := options.EndpointValidator.NewDialContext(parsed.Host, defaultPort, dialer.DialContext)
		if err != nil {
			return nil, fmt.Errorf("prepare llm dialer: %w", err)
		}
		transport.DialContext = dialContext
		// Com proxy o dial iria para o proxy, nao para o host validado.
		transport.Proxy = nil
	}

	return &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errLLMRedirectRecusado
		},
	}, nil
}

func NewLLMProvider(providerType string, credentials map[string]string, opts ...LLMClientOptions) (domain.LLMService, error) {
	options := resolveLLMClientOptions(opts)
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

	httpClient, err := buildLLMHTTPClient(endpoint, hasUserBaseURL(credentials), options)
	if err != nil {
		return nil, err
	}

	if normalizedProvider == domain.ProviderOllama && isOllamaNativeEndpoint(endpoint) {
		return &ollamaNativeLLMClient{
			httpClient: httpClient,
			provider:   normalizedProvider,
			endpoint:   endpoint,
			apiKey:     apiKey,
			model:      model,
			pricing:    pricing,
			hasPricing: hasPricing,
		}, nil
	}

	return &openAICompatibleLLMClient{
		httpClient: httpClient,
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
		Temperature: defaultLLMGenerationTemperature,
	}

	completion, err := c.createChatCompletion(ctx, payload)
	if err != nil {
		return "", err
	}

	c.registrarConsumo(completion.Usage)

	if len(completion.Choices) == 0 {
		return "", errors.New("llm response returned no choices")
	}

	return strings.TrimSpace(completion.Choices[0].Message.Content), nil
}

func hasUserBaseURL(credentials map[string]string) bool {
	return strings.TrimSpace(credentials["base_url"]) != ""
}

func ValidateLLMConnection(ctx context.Context, providerType string, credentials map[string]string, opts ...LLMClientOptions) error {
	service, err := NewLLMProvider(providerType, credentials, opts...)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 1; attempt <= maxLLMInfraAttempts; attempt++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, llmValidationTimeout)
		err = validateLLMConnectionOnce(timeoutCtx, service)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		if ctx.Err() != nil || !isRetriableLLMInfraError(err) || attempt == maxLLMInfraAttempts || !waitForLLMInfraRetry(ctx) {
			break
		}
	}

	return lastErr
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

func ListLLMModels(ctx context.Context, providerType string, credentials map[string]string, opts ...LLMClientOptions) ([]string, error) {
	options := resolveLLMClientOptions(opts)
	normalizedProvider := strings.ToUpper(strings.TrimSpace(providerType))

	if !SupportsLLMModelDiscovery(normalizedProvider) {
		return nil, fmt.Errorf("model discovery is not implemented for provider %s", normalizedProvider)
	}

	var (
		models  []string
		lastErr error
	)

	for attempt := 1; attempt <= maxLLMInfraAttempts; attempt++ {
		switch normalizedProvider {
		case domain.ProviderOllama:
			models, lastErr = listOllamaModels(ctx, credentials, options)
		default:
			models, lastErr = listOpenAICompatibleModels(ctx, normalizedProvider, credentials, options)
		}

		if lastErr == nil {
			return models, nil
		}

		if ctx.Err() != nil || !isRetriableLLMInfraError(lastErr) || attempt == maxLLMInfraAttempts || !waitForLLMInfraRetry(ctx) {
			break
		}
	}

	return nil, lastErr
}

func listOllamaModels(ctx context.Context, credentials map[string]string, options LLMClientOptions) ([]string, error) {
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

	httpClient, err := buildLLMHTTPClient(req.URL.String(), hasUserBaseURL(credentials), options)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
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

func listOpenAICompatibleModels(ctx context.Context, providerType string, credentials map[string]string, options LLMClientOptions) ([]string, error) {
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

	httpClient, err := buildLLMHTTPClient(req.URL.String(), hasUserBaseURL(credentials), options)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
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
		Temperature: defaultLLMStreamingTemperature,
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
			Temperature: floatPtr(defaultLLMGenerationTemperature),
		},
	})
	if err != nil {
		return "", err
	}
	c.registrarConsumo(response)
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
			Temperature: floatPtr(defaultLLMStreamingTemperature),
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
		c.registrarConsumo(&chunk)
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

func validateLLMConnectionOnce(ctx context.Context, service domain.LLMService) error {
	switch client := service.(type) {
	case *openAICompatibleLLMClient:
		_, err := client.createChatCompletion(ctx, chatCompletionRequest{
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
		_, err := client.createChat(ctx, ollamaChatRequest{
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

func isRetriableLLMInfraError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var networkErr net.Error
	if errors.As(err, &networkErr) && (networkErr.Timeout() || isTemporaryNetError(networkErr)) {
		return true
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	patterns := []string{
		"deadline exceeded",
		"timeout",
		"i/o timeout",
		"temporary failure in name resolution",
		"tls handshake timeout",
	}
	for _, pattern := range patterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}

	return false
}

func isTemporaryNetError(err net.Error) bool {
	type temporaryError interface {
		Temporary() bool
	}

	temporaryErr, ok := err.(temporaryError)
	return ok && temporaryErr.Temporary()
}

func waitForLLMInfraRetry(ctx context.Context) bool {
	timer := time.NewTimer(llmInfraRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
