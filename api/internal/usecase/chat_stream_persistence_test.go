package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type chatConversationRepoStub struct {
	conversation *models.ChatConversation
	added        []models.ChatConversationEntry
	addErr       error
}

func (r *chatConversationRepoStub) CreateConversation(context.Context, *models.ChatConversation) error {
	return nil
}

func (r *chatConversationRepoStub) GetConversationByID(context.Context, uint, uint) (*models.ChatConversation, error) {
	return r.conversation, nil
}

func (r *chatConversationRepoStub) ListConversationsByCollection(context.Context, uint, uint) ([]models.ChatConversation, error) {
	return nil, nil
}

func (r *chatConversationRepoStub) ListMessagesByConversation(context.Context, uint, uint) ([]models.ChatConversationEntry, error) {
	return nil, nil
}

func (r *chatConversationRepoStub) UpdateConversationTitle(context.Context, uint, uint, string) (*models.ChatConversation, error) {
	return r.conversation, nil
}

func (r *chatConversationRepoStub) DeleteConversation(context.Context, uint, uint) error {
	return nil
}

func (r *chatConversationRepoStub) AddMessage(_ context.Context, _ uint, _ uint, message *models.ChatConversationEntry) error {
	if r.addErr != nil {
		return r.addErr
	}
	if message != nil {
		r.added = append(r.added, *message)
	}
	return nil
}

func (r *chatConversationRepoStub) assistantMessages() []models.ChatConversationEntry {
	out := make([]models.ChatConversationEntry, 0, len(r.added))
	for _, message := range r.added {
		if message.Role == "assistant" {
			out = append(out, message)
		}
	}
	return out
}

type chatCollectionContextStub struct{}

func (chatCollectionContextStub) GetCollectionContext(context.Context, uint, uint) (*domain.CollectionContext, error) {
	// Collection e ponteiro e buildCollectionChatPrompt o desreferencia sem
	// checar: um contexto "vazio" demais faz panic em vez de exercitar o
	// caminho que interessa.
	return &domain.CollectionContext{
		Collection: &models.Collection{ID: 3, Name: "Contratos"},
	}, nil
}

// chatStreamingLLMStub emite os deltas informados e entao devolve streamErr.
// Reproduz o cenario real: o provider entrega parte da resposta e a conexao
// morre no meio.
type chatStreamingLLMStub struct {
	deltas    []string
	streamErr error
}

func (s *chatStreamingLLMStub) GenerateText(context.Context, string, string) (string, error) {
	return strings.Join(s.deltas, ""), nil
}

func (s *chatStreamingLLMStub) StreamText(_ context.Context, _ string, _ string, onDelta func(string) error) error {
	for _, delta := range s.deltas {
		if err := onDelta(delta); err != nil {
			return err
		}
	}
	return s.streamErr
}

type chatLLMResolverStub struct {
	llm domain.LLMService
}

func (r *chatLLMResolverStub) ResolveLLMForIndexing(context.Context, uint) (domain.LLMService, error) {
	return r.llm, nil
}

func (r *chatLLMResolverStub) ResolveLLMForChat(context.Context, uint) (domain.LLMService, error) {
	return r.llm, nil
}

func newChatUseCaseForTest(repo *chatConversationRepoStub, llm domain.LLMService) domain.ChatUseCase {
	return NewChatUseCase(chatCollectionContextStub{}, repo, &chatLLMResolverStub{llm: llm})
}

// O caso que motivou a correcao: o LLM entrega parte da resposta e a conexao
// cai. Antes, o `return err` pulava a persistencia inteira -- o usuario via o
// texto na tela, recarregava a conversa e a resposta tinha sumido, sem nenhum
// registro de que a chamada foi feita e paga.
func TestStreamCollectionChatPersisteRespostaParcialQuandoOStreamFalha(t *testing.T) {
	t.Parallel()

	repo := &chatConversationRepoStub{
		conversation: &models.ChatConversation{ID: 9, UserID: 1, CollectionID: 3},
	}
	falhaDoProvider := errors.New("conexao encerrada pelo provider")
	llm := &chatStreamingLLMStub{
		deltas:    []string{"O prazo de pagamento ", "e de 15 dias"},
		streamErr: falhaDoProvider,
	}

	useCase := newChatUseCaseForTest(repo, llm)

	err := useCase.StreamCollectionChat(context.Background(), 1, domain.ChatRequest{
		ConversationID: 9,
		CollectionID:   3,
		Prompt:         "qual o prazo?",
	}, func(domain.ChatStreamEvent) error { return nil })

	// O erro original precisa continuar chegando ao handler, que o usa para
	// escolher a mensagem certa ao usuario.
	if !errors.Is(err, falhaDoProvider) {
		t.Fatalf("esperava o erro original do provider, veio %v", err)
	}

	assistentes := repo.assistantMessages()
	if len(assistentes) != 1 {
		t.Fatalf("esperava 1 resposta parcial persistida, veio %d", len(assistentes))
	}
	if assistentes[0].Content != "O prazo de pagamento e de 15 dias" {
		t.Fatalf("o texto parcial persistido esta errado: %q", assistentes[0].Content)
	}
}

// Nada de resposta => nada a persistir. Gravar uma mensagem de assistente vazia
// sujaria a conversa com uma bolha em branco.
func TestStreamCollectionChatNaoPersisteRespostaVazia(t *testing.T) {
	t.Parallel()

	repo := &chatConversationRepoStub{
		conversation: &models.ChatConversation{ID: 10, UserID: 1, CollectionID: 3},
	}
	llm := &chatStreamingLLMStub{streamErr: errors.New("provider fora do ar")}

	useCase := newChatUseCaseForTest(repo, llm)

	_ = useCase.StreamCollectionChat(context.Background(), 1, domain.ChatRequest{
		ConversationID: 10,
		CollectionID:   3,
		Prompt:         "qual o prazo?",
	}, func(domain.ChatStreamEvent) error { return nil })

	if got := len(repo.assistantMessages()); got != 0 {
		t.Fatalf("nao deveria persistir resposta vazia, persistiu %d", got)
	}
}

func TestStreamCollectionChatPersisteRespostaCompletaNoCaminhoFeliz(t *testing.T) {
	t.Parallel()

	repo := &chatConversationRepoStub{
		conversation: &models.ChatConversation{ID: 11, UserID: 1, CollectionID: 3},
	}
	llm := &chatStreamingLLMStub{deltas: []string{"resposta ", "completa"}}

	useCase := newChatUseCaseForTest(repo, llm)

	var done bool
	err := useCase.StreamCollectionChat(context.Background(), 1, domain.ChatRequest{
		ConversationID: 11,
		CollectionID:   3,
		Prompt:         "pergunta",
	}, func(event domain.ChatStreamEvent) error {
		if event.Type == chatStreamEventDone {
			done = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminho feliz nao deveria falhar: %v", err)
	}
	if !done {
		t.Fatal("o evento 'done' nao foi emitido")
	}

	assistentes := repo.assistantMessages()
	if len(assistentes) != 1 || assistentes[0].Content != "resposta completa" {
		t.Fatalf("resposta completa nao foi persistida corretamente: %+v", assistentes)
	}
}

// Se a escrita no cliente falhar (conexao morta), o texto ja gerado tambem
// precisa ser salvo: a chamada de LLM foi paga de qualquer forma.
func TestStreamCollectionChatPersisteQuandoAEscritaNoClienteFalha(t *testing.T) {
	t.Parallel()

	repo := &chatConversationRepoStub{
		conversation: &models.ChatConversation{ID: 12, UserID: 1, CollectionID: 3},
	}
	llm := &chatStreamingLLMStub{deltas: []string{"parte um ", "parte dois"}}

	useCase := newChatUseCaseForTest(repo, llm)

	falhaDeEscrita := errors.New("broken pipe")
	var deltasVistos int
	err := useCase.StreamCollectionChat(context.Background(), 1, domain.ChatRequest{
		ConversationID: 12,
		CollectionID:   3,
		Prompt:         "pergunta",
	}, func(event domain.ChatStreamEvent) error {
		if event.Type != chatStreamEventDelta {
			return nil
		}
		deltasVistos++
		if deltasVistos == 2 {
			return falhaDeEscrita
		}
		return nil
	})

	if !errors.Is(err, falhaDeEscrita) {
		t.Fatalf("esperava o erro de escrita, veio %v", err)
	}

	assistentes := repo.assistantMessages()
	if len(assistentes) != 1 {
		t.Fatalf("esperava a resposta parcial persistida, veio %d", len(assistentes))
	}
	if assistentes[0].Content != "parte um parte dois" {
		t.Fatalf("texto persistido inesperado: %q", assistentes[0].Content)
	}
}
