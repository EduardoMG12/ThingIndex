package domain

import (
	"context"
	"time"

	"thing-index/internal/models"
)

type ChatMessage struct {
	ID      uint
	Role    string
	Content string
	Sources []ChatSource
}

type ChatRequest struct {
	ConversationID uint
	CollectionID   uint
	Prompt         string
	Language       string
	History        []ChatMessage
}

type ChatSource struct {
	ID      string `json:"id"`
	DocName string `json:"docName"`
	Page    int    `json:"page"`
	Snippet string `json:"snippet"`
	// Opcionais: mensagens antigas nao tem.
	DocumentID uint   `json:"documentId,omitempty"`
	NodeID     string `json:"nodeId,omitempty"`
	// ParagraphID e o paragrafo da pagina com mais termos da pergunta: e o
	// que o canvas grifa quando o usuario clica na fonte.
	ParagraphID uint `json:"paragraphId,omitempty"`
	// Lines sao os indices (em metadata.lines do paragrafo) das linhas que
	// casam a pergunta: o grifo fino, em vez do paragrafo inteiro.
	Lines []int `json:"lines,omitempty"`
	// MatchedTerms sao os termos da pergunta presentes na fonte.
	MatchedTerms []string `json:"matchedTerms,omitempty"`
	// Box, LineBoxes e IndexedAt deixam o grifo reproduzivel depois de reindexar.
	// ContinuesOnPage e a pagina seguinte quando o paragrafo quebra na virada; o
	// prompt leva o comeco dela.
	ContinuesOnPage int               `json:"continuesOnPage,omitempty"`
	Box             *models.SourceBox `json:"box,omitempty"`
	LineBoxes       [][4]float64      `json:"lineBoxes,omitempty"`
	IndexedAt       *time.Time        `json:"indexedAt,omitempty"`
	// Body e o texto que vai ao prompt quando a fonte e menor que uma pagina
	// (linha de tabela). Interno: nao sai na API nem e persistido.
	Body string `json:"-"`
}

// ChatUsage e o consumo de uma resposta. UsageReported=false: numeros estimados
// (runas/4). EstimatedCostUSD e nulo sem preco cadastrado.
type ChatUsage struct {
	PromptTokens     int      `json:"promptTokens"`
	CompletionTokens int      `json:"completionTokens"`
	UsageReported    bool     `json:"usageReported"`
	ProviderType     string   `json:"providerType,omitempty"`
	Model            string   `json:"model,omitempty"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
}

type ChatStreamEvent struct {
	Type    string       `json:"type"`
	Stage   string       `json:"stage,omitempty"`
	Message string       `json:"message,omitempty"`
	Delta   string       `json:"delta,omitempty"`
	Sources []ChatSource `json:"sources,omitempty"`
	Usage   *ChatUsage   `json:"usage,omitempty"`
}

type ChatConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *models.ChatConversation) error
	GetConversationByID(ctx context.Context, userID uint, conversationID uint) (*models.ChatConversation, error)
	ListConversationsByCollection(ctx context.Context, userID uint, collectionID uint) ([]models.ChatConversation, error)
	ListMessagesByConversation(ctx context.Context, userID uint, conversationID uint) ([]models.ChatConversationEntry, error)
	UpdateConversationTitle(ctx context.Context, userID uint, conversationID uint, title string) (*models.ChatConversation, error)
	DeleteConversation(ctx context.Context, userID uint, conversationID uint) error
	AddMessage(ctx context.Context, userID uint, conversationID uint, message *models.ChatConversationEntry) error
}

type ChatUseCase interface {
	StreamCollectionChat(ctx context.Context, userID uint, request ChatRequest, onEvent func(ChatStreamEvent) error) error
	CreateConversation(ctx context.Context, userID uint, collectionID uint, title string) (*models.ChatConversation, error)
	ListConversations(ctx context.Context, userID uint, collectionID uint) ([]models.ChatConversation, error)
	GetConversationMessages(ctx context.Context, userID uint, conversationID uint) (*models.ChatConversation, []models.ChatConversationEntry, error)
	UpdateConversationTitle(ctx context.Context, userID uint, conversationID uint, title string) (*models.ChatConversation, error)
	DeleteConversation(ctx context.Context, userID uint, conversationID uint) error
}
