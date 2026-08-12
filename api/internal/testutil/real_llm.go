package testutil

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"thing-index/internal/domain"

	"github.com/joho/godotenv"
)

var loadRootEnvOnce sync.Once

type RealLLMConfig struct {
	ProviderType      string
	Credentials       map[string]string
	RunModelDiscovery bool
	RunProcessing     bool
}

func RequireRealLLM(t *testing.T) RealLLMConfig {
	t.Helper()

	cfg, ok := realLLMConfigFromEnv()
	if !ok {
		t.Skip("real LLM integration is disabled; set THINGINDEX_TEST_LLM_ENABLED=true and configure provider credentials")
	}

	return cfg
}

func RequireRealLLMProcessing(t *testing.T) RealLLMConfig {
	t.Helper()

	cfg := RequireRealLLM(t)
	if !cfg.RunProcessing {
		t.Skip("real LLM document-processing test is disabled; set THINGINDEX_TEST_LLM_RUN_PROCESSING=true to enable it")
	}

	return cfg
}

func RequireRealLLMModelDiscovery(t *testing.T) RealLLMConfig {
	t.Helper()

	cfg := RequireRealLLM(t)
	if !cfg.RunModelDiscovery {
		t.Skip("real LLM model-discovery test is disabled; set THINGINDEX_TEST_LLM_RUN_MODEL_DISCOVERY=true to enable it")
	}

	return cfg
}

func realLLMConfigFromEnv() (RealLLMConfig, bool) {
	loadRootEnvOnce.Do(loadNearestDotEnv)

	enabled, _ := strconv.ParseBool(strings.TrimSpace(firstNonEmpty(os.Getenv("THINGINDEX_TEST_LLM_ENABLED"), "false")))
	if !enabled {
		return RealLLMConfig{}, false
	}

	providerType := strings.ToUpper(strings.TrimSpace(firstNonEmpty(
		os.Getenv("THINGINDEX_TEST_LLM_PROVIDER"),
		inferProviderTypeFromEnv(),
	)))
	if providerType == "" {
		return RealLLMConfig{}, false
	}

	credentials := map[string]string{}
	if apiKey := strings.TrimSpace(firstNonEmpty(os.Getenv("THINGINDEX_TEST_LLM_API_KEY"), providerSpecificEnv(providerType, "API_KEY"))); apiKey != "" {
		credentials["api_key"] = apiKey
	}
	if baseURL := strings.TrimSpace(firstNonEmpty(os.Getenv("THINGINDEX_TEST_LLM_BASE_URL"), providerSpecificEnv(providerType, "BASE_URL"))); baseURL != "" {
		credentials["base_url"] = baseURL
	}
	if model := strings.TrimSpace(firstNonEmpty(os.Getenv("THINGINDEX_TEST_LLM_MODEL"), providerSpecificEnv(providerType, "MODEL"))); model != "" {
		credentials["model"] = model
	}

	runModelDiscovery, _ := strconv.ParseBool(strings.TrimSpace(firstNonEmpty(os.Getenv("THINGINDEX_TEST_LLM_RUN_MODEL_DISCOVERY"), "false")))
	runProcessing, _ := strconv.ParseBool(strings.TrimSpace(firstNonEmpty(os.Getenv("THINGINDEX_TEST_LLM_RUN_PROCESSING"), "false")))

	return RealLLMConfig{
		ProviderType:      providerType,
		Credentials:       credentials,
		RunModelDiscovery: runModelDiscovery,
		RunProcessing:     runProcessing,
	}, true
}

func inferProviderTypeFromEnv() string {
	switch {
	case strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "":
		return domain.ProviderOpenAI
	case strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) != "":
		return domain.ProviderOpenRouter
	case strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) != "":
		return domain.ProviderDeepSeek
	case strings.TrimSpace(os.Getenv("GROK_API_KEY")) != "":
		return domain.ProviderGrok
	case strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) != "":
		return domain.ProviderGemini
	case strings.TrimSpace(os.Getenv("CLAUDE_API_KEY")) != "":
		return domain.ProviderClaude
	case strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")) != "" || strings.TrimSpace(os.Getenv("OLLAMA_MODEL")) != "" || strings.TrimSpace(os.Getenv("OLLAMA_API_KEY")) != "":
		return domain.ProviderOllama
	default:
		return ""
	}
}

func providerSpecificEnv(providerType, suffix string) string {
	switch providerType {
	case domain.ProviderOpenAI:
		return os.Getenv("OPENAI_" + suffix)
	case domain.ProviderOpenRouter:
		return os.Getenv("OPENROUTER_" + suffix)
	case domain.ProviderDeepSeek:
		return os.Getenv("DEEPSEEK_" + suffix)
	case domain.ProviderGrok:
		return os.Getenv("GROK_" + suffix)
	case domain.ProviderGemini:
		return os.Getenv("GEMINI_" + suffix)
	case domain.ProviderClaude:
		return os.Getenv("CLAUDE_" + suffix)
	case domain.ProviderOllama:
		return os.Getenv("OLLAMA_" + suffix)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func loadNearestDotEnv() {
	workingDir, err := os.Getwd()
	if err != nil {
		return
	}

	dir := workingDir
	for {
		candidate := filepath.Join(dir, ".env")
		if _, statErr := os.Stat(candidate); statErr == nil {
			_ = godotenv.Load(candidate)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
