package usecase

import (
	"context"
	"errors"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// Stub que informa consumo, modelo e preco, como o cliente OpenAI-compativel.
type chatMeteredLLMStub struct {
	chatStreamingLLMStub
	prompt, completion int
	hasPrice           bool
}

func (s *chatMeteredLLMStub) LastTokenUsage() (int, int, int, bool) {
	return s.prompt, s.completion, s.prompt + s.completion, true
}
func (s *chatMeteredLLMStub) ProviderType() string { return "OPENAI" }
func (s *chatMeteredLLMStub) ModelName() string    { return "gpt-4o-mini" }
func (s *chatMeteredLLMStub) Pricing() (domain.LLMPricing, bool) {
	if !s.hasPrice {
		return domain.LLMPricing{}, false
	}
	return domain.LLMPricing{InputCostPerMillionUSD: 0.15, OutputCostPerMillionUSD: 0.60}, true
}

func streamAndCollect(t *testing.T, repo *chatConversationRepoStub, llm domain.LLMService) (*domain.ChatUsage, error) {
	t.Helper()
	var usage *domain.ChatUsage
	err := newChatUseCaseForTest(repo, llm).StreamCollectionChat(context.Background(), 1, domain.ChatRequest{
		ConversationID: 11, CollectionID: 3, Prompt: "pergunta",
	}, func(event domain.ChatStreamEvent) error {
		if event.Type == chatStreamEventDone {
			usage = event.Usage
		}
		return nil
	})
	return usage, err
}

// O artigo compara custo por consulta: cada resposta grava tokens, modelo,
// provedor e custo -- os numeros do provedor quando ele informa.
func TestStreamCollectionChatPersisteConsumoReportadoComCusto(t *testing.T) {
	t.Parallel()
	repo := &chatConversationRepoStub{conversation: &models.ChatConversation{ID: 11, UserID: 1, CollectionID: 3}}
	llm := &chatMeteredLLMStub{chatStreamingLLMStub: chatStreamingLLMStub{deltas: []string{"resposta"}}, prompt: 1000, completion: 200, hasPrice: true}

	usage, err := streamAndCollect(t, repo, llm)
	if err != nil {
		t.Fatalf("caminho feliz falhou: %v", err)
	}
	if usage == nil || !usage.UsageReported || usage.PromptTokens != 1000 || usage.CompletionTokens != 200 {
		t.Fatalf("evento done sem o consumo reportado: %+v", usage)
	}
	// 1000 * 0.15/1M + 200 * 0.60/1M = 0.00027
	if usage.EstimatedCostUSD == nil || *usage.EstimatedCostUSD < 0.000269 || *usage.EstimatedCostUSD > 0.000271 {
		t.Fatalf("custo errado: %+v", usage.EstimatedCostUSD)
	}
	gravada := repo.assistantMessages()[0]
	if gravada.PromptTokens != 1000 || gravada.CompletionTokens != 200 || !gravada.UsageReported || gravada.Model != "gpt-4o-mini" || gravada.ProviderType != "OPENAI" || gravada.EstimatedCostUSD == nil {
		t.Fatalf("consumo nao persistido na mensagem: %+v", gravada)
	}
}

// Provedor que nao informa consumo (Ollama local): estimativa runas/4,
// rotulada como estimativa, e custo nulo sem preco.
func TestStreamCollectionChatEstimaConsumoQuandoOProvedorNaoInforma(t *testing.T) {
	t.Parallel()
	repo := &chatConversationRepoStub{conversation: &models.ChatConversation{ID: 11, UserID: 1, CollectionID: 3}}
	llm := &chatStreamingLLMStub{deltas: []string{"uma resposta com dezesseis"}} // 25 runas -> 7 tokens

	usage, err := streamAndCollect(t, repo, llm)
	if err != nil {
		t.Fatalf("caminho feliz falhou: %v", err)
	}
	if usage == nil || usage.UsageReported || usage.PromptTokens == 0 || usage.CompletionTokens != 7 || usage.EstimatedCostUSD != nil {
		t.Fatalf("estimativa errada: %+v", usage)
	}
	if gravada := repo.assistantMessages()[0]; gravada.UsageReported || gravada.CompletionTokens != 7 || gravada.EstimatedCostUSD != nil {
		t.Fatalf("estimativa nao persistida: %+v", gravada)
	}
}

// A resposta parcial tambem foi paga: o consumo vai junto dela.
func TestStreamCollectionChatPersisteConsumoDaRespostaParcial(t *testing.T) {
	t.Parallel()
	repo := &chatConversationRepoStub{conversation: &models.ChatConversation{ID: 11, UserID: 1, CollectionID: 3}}
	llm := &chatMeteredLLMStub{chatStreamingLLMStub: chatStreamingLLMStub{deltas: []string{"parcial"}, streamErr: errors.New("conexao caiu")}, prompt: 50, completion: 5}

	if _, err := streamAndCollect(t, repo, llm); err == nil {
		t.Fatal("esperava o erro do stream")
	}
	if gravada := repo.assistantMessages()[0]; gravada.PromptTokens != 50 || gravada.CompletionTokens != 5 || !gravada.UsageReported {
		t.Fatalf("parcial sem consumo: %+v", gravada)
	}
}
