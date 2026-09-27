package usecase

import (
	"context"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type chatLLMPrazoStub struct {
	temPrazo bool
	prazo    time.Time
}

func (s *chatLLMPrazoStub) GenerateText(context.Context, string, string) (string, error) {
	return "", nil
}

func (s *chatLLMPrazoStub) StreamText(ctx context.Context, _ string, _ string, onDelta func(string) error) error {
	s.prazo, s.temPrazo = ctx.Deadline()
	return onDelta("ok")
}

// O cliente HTTP do LLM perdeu o Timeout global (P1-2), entao o chat precisa
// entregar ao provedor um ctx com prazo explicito, mesmo quando o ctx da
// requisicao nao tem nenhum.
func TestStreamCollectionChatEntregaCtxComPrazoAoLLM(t *testing.T) {
	t.Parallel()

	repo := &chatConversationRepoStub{
		conversation: &models.ChatConversation{ID: 13, UserID: 1, CollectionID: 3},
	}
	llm := &chatLLMPrazoStub{}
	useCase := newChatUseCaseForTest(repo, llm)

	inicio := time.Now()
	if err := useCase.StreamCollectionChat(context.Background(), 1, domain.ChatRequest{
		ConversationID: 13,
		CollectionID:   3,
		Prompt:         "pergunta",
	}, func(domain.ChatStreamEvent) error { return nil }); err != nil {
		t.Fatalf("chat falhou: %v", err)
	}

	if !llm.temPrazo {
		t.Fatal("o LLM recebeu ctx sem prazo")
	}
	if restante := llm.prazo.Sub(inicio); restante > chatLLMTimeout+time.Second || restante <= 0 {
		t.Fatalf("prazo do LLM fora do esperado: %s", restante)
	}
}
