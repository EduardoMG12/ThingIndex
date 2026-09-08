package domain

import (
	"context"

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
	// DocumentID e NodeID deixam o frontend ir ao documento (e a secao, quando
	// o hit e de um no do sumario) sem parsear o ID. Ambos opcionais para as
	// mensagens gravadas antes de existirem.
	DocumentID uint   `json:"documentId,omitempty"`
	NodeID     string `json:"nodeId,omitempty"`
}

type ChatStreamEvent struct {
	Type    string       `json:"type"`
	Stage   string       `json:"stage,omitempty"`
	Message string       `json:"message,omitempty"`
	Delta   string       `json:"delta,omitempty"`
	Sources []ChatSource `json:"sources,omitempty"`
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
