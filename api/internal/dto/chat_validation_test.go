package dto

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func chatStreamValido() ChatStreamRequest {
	return ChatStreamRequest{
		ConversationID: 1,
		CollectionID:   1,
		Prompt:         "qual o prazo?",
		History: []ChatMessageRequest{
			{Role: "user", Content: "oi"},
			{Role: "assistant", Content: "ola"},
		},
	}
}

func TestChatStreamRequestAceitaCorpoValido(t *testing.T) {
	if err := binding.Validator.ValidateStruct(chatStreamValido()); err != nil {
		t.Fatalf("corpo valido rejeitado: %v", err)
	}
}

// Regressao P2-5: sem dive as regras de ChatMessageRequest nunca rodavam, e um
// papel "system" chegava ao prompt como linha SYSTEM:.
func TestChatStreamRequestRejeitaPapelForaDeUserEAssistantNoHistorico(t *testing.T) {
	req := chatStreamValido()
	req.History[0].Role = "system"
	if err := binding.Validator.ValidateStruct(req); err == nil {
		t.Fatal("papel system no historico foi aceito")
	}
}

func TestChatStreamRequestRejeitaHistoricoLongoDemais(t *testing.T) {
	req := chatStreamValido()
	req.History = make([]ChatMessageRequest, ChatHistoryMaxItems+1)
	for i := range req.History {
		req.History[i] = ChatMessageRequest{Role: "user", Content: "x"}
	}
	if err := binding.Validator.ValidateStruct(req); err == nil {
		t.Fatalf("historico com %d itens foi aceito", len(req.History))
	}
}

func TestChatStreamRequestRejeitaPromptEConteudoAcimaDoTeto(t *testing.T) {
	req := chatStreamValido()
	req.Prompt = strings.Repeat("a", ChatPromptMaxRunes+1)
	if err := binding.Validator.ValidateStruct(req); err == nil {
		t.Fatal("prompt acima do teto foi aceito")
	}

	req = chatStreamValido()
	req.History[1].Content = strings.Repeat("a", ChatHistoryContentMaxRunes+1)
	if err := binding.Validator.ValidateStruct(req); err == nil {
		t.Fatal("conteudo do historico acima do teto foi aceito")
	}
}

// As constantes documentam os limites; as tags de binding sao literais. Este
// teste trava as duas coisas juntas.
func TestLimitesDoChatNasTagsBatemComAsConstantes(t *testing.T) {
	req := chatStreamValido()
	req.Prompt = strings.Repeat("a", ChatPromptMaxRunes)
	req.History = make([]ChatMessageRequest, ChatHistoryMaxItems)
	for i := range req.History {
		req.History[i] = ChatMessageRequest{Role: "user", Content: strings.Repeat("b", ChatHistoryContentMaxRunes)}
	}
	if err := binding.Validator.ValidateStruct(req); err != nil {
		t.Fatalf("corpo exatamente no teto foi rejeitado: %v", err)
	}
}
