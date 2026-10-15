package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"thing-index/internal/domain"
)

func TestRespostaDeChatCarregaOConsumoDeTokens(t *testing.T) {
	// Formato da API compativel com OpenAI.
	bruto := `{
		"choices": [{"message": {"role": "assistant", "content": "resposta"}}],
		"usage": {"prompt_tokens": 412, "completion_tokens": 87, "total_tokens": 499}
	}`

	var resposta chatCompletionResponse
	if err := json.Unmarshal([]byte(bruto), &resposta); err != nil {
		t.Fatalf("falha ao decodificar: %v", err)
	}

	if resposta.Usage == nil {
		t.Fatal("campo usage descartado: o consumo de tokens nao chega ao chamador")
	}
	if resposta.Usage.PromptTokens != 412 {
		t.Fatalf("prompt_tokens: esperava 412, veio %d", resposta.Usage.PromptTokens)
	}
	if resposta.Usage.CompletionTokens != 87 {
		t.Fatalf("completion_tokens: esperava 87, veio %d", resposta.Usage.CompletionTokens)
	}
	if resposta.Usage.TotalTokens != 499 {
		t.Fatalf("total_tokens: esperava 499, veio %d", resposta.Usage.TotalTokens)
	}
}

// Provedor que nao reporta consumo (Ollama local) nao pode virar erro de indexacao.
func TestRespostaSemUsageNaoQuebra(t *testing.T) {
	bruto := `{"choices": [{"message": {"role": "assistant", "content": "resposta"}}]}`

	var resposta chatCompletionResponse
	if err := json.Unmarshal([]byte(bruto), &resposta); err != nil {
		t.Fatalf("falha ao decodificar resposta sem usage: %v", err)
	}

	if resposta.Usage != nil {
		t.Fatalf("usage deveria ser nil quando ausente, veio %+v", resposta.Usage)
	}
	if len(resposta.Choices) != 1 {
		t.Fatal("a resposta em si deixou de ser lida")
	}
}

// Alguns provedores devolvem prompt e completion sem o total.
func TestTotalDeTokensSeApoiaNasPartesQuandoAusente(t *testing.T) {
	bruto := `{
		"choices": [{"message": {"role": "assistant", "content": "ok"}}],
		"usage": {"prompt_tokens": 100, "completion_tokens": 25}
	}`

	var resposta chatCompletionResponse
	if err := json.Unmarshal([]byte(bruto), &resposta); err != nil {
		t.Fatalf("falha ao decodificar: %v", err)
	}

	if total := resposta.Usage.Total(); total != 125 {
		t.Fatalf("total deveria cair para a soma das partes (125), veio %d", total)
	}
}

// Se o provedor informa o total, ele vale (pode incluir tokens de sistema).
func TestTotalDeTokensPrefereOValorDoProvedor(t *testing.T) {
	usage := &tokenUsage{PromptTokens: 100, CompletionTokens: 25, TotalTokens: 140}

	if total := usage.Total(); total != 140 {
		t.Fatalf("com total informado deveria devolver 140, veio %d", total)
	}
}

// TestUsageNuloEhSeguro: o chamador nao deve precisar checar nil antes de somar.
func TestUsageNuloEhSeguro(t *testing.T) {
	var usage *tokenUsage
	if total := usage.Total(); total != 0 {
		t.Fatalf("usage nil deveria somar 0, veio %d", total)
	}
}

// Daqui para baixo: cliente contra um servidor HTTP real.

func TestClienteExpoeOConsumoDaUltimaChamada(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"role": "assistant", "content": "resumo"}}],
			"usage": {"prompt_tokens": 300, "completion_tokens": 50, "total_tokens": 350}
		}`))
	}))
	defer servidor.Close()

	cliente := &openAICompatibleLLMClient{
		httpClient: servidor.Client(),
		provider:   "TESTE",
		endpoint:   servidor.URL,
		model:      "modelo-de-teste",
	}

	// Antes de qualquer chamada nao ha consumo a reportar.
	if _, _, _, reportado := cliente.LastTokenUsage(); reportado {
		t.Fatal("cliente reportou consumo antes da primeira chamada")
	}

	if _, err := cliente.GenerateText(context.Background(), "sistema", "usuario"); err != nil {
		t.Fatalf("geracao falhou: %v", err)
	}

	prompt, completion, total, reportado := cliente.LastTokenUsage()
	if !reportado {
		t.Fatal("consumo nao chegou ao chamador: a metrica de custo continua sem fonte")
	}
	if prompt != 300 || completion != 50 || total != 350 {
		t.Fatalf("consumo errado: prompt=%d completion=%d total=%d", prompt, completion, total)
	}
}

func TestProvedorSemUsageNaoInventaNumero(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices": [{"message": {"role": "assistant", "content": "ok"}}]}`))
	}))
	defer servidor.Close()

	cliente := &openAICompatibleLLMClient{
		httpClient: servidor.Client(),
		provider:   "TESTE",
		endpoint:   servidor.URL,
		model:      "modelo-de-teste",
	}

	if _, err := cliente.GenerateText(context.Background(), "sistema", "usuario"); err != nil {
		t.Fatalf("geracao falhou: %v", err)
	}

	// Provedor que nao reporta tem de ser distinguivel de provedor que reportou zero.
	if _, _, _, reportado := cliente.LastTokenUsage(); reportado {
		t.Fatal("cliente marcou consumo como reportado sem o provedor informar")
	}
}

// O chamador usa type assertion: mudar a assinatura pararia de medir em silencio.
func TestClienteImplementaTokenUsageReporter(t *testing.T) {
	var _ domain.TokenUsageReporter = (*openAICompatibleLLMClient)(nil)
}
