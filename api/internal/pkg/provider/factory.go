package provider

import (
	"context"
	"errors"
	"thing-index/internal/domain"
)

type openaiClient struct {
	apiKey string
}

func (c *openaiClient) Summarize(ctx context.Context, text string) (string, error) {
	// Implementation would use c.apiKey to call OpenAI API
	return "[OpenAI] Placeholder summary of: " + text, nil
}

type geminiClient struct {
	apiKey string
}

func (c *geminiClient) Summarize(ctx context.Context, text string) (string, error) {
	// Implementation would use c.apiKey to call Google Gemini API
	return "[Gemini] Placeholder summary of: " + text, nil
}

// NewLLMProvider instantiates the correct LLM client based on the provider type
func NewLLMProvider(providerType string, credentials map[string]string) (domain.LLMService, error) {
	apiKey, ok := credentials["api_key"]
	if !ok {
		return nil, errors.New("missing api_key in credentials")
	}

	switch providerType {
	case "OPENAI":
		return &openaiClient{apiKey: apiKey}, nil
	case "GEMINI":
		return &geminiClient{apiKey: apiKey}, nil
	default:
		return nil, errors.New("unsupported provider type")
	}
}
