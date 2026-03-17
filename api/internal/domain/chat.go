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
	ID      string
	DocName string
	Page    int
	Snippet string
}

type ChatStreamEvent struct {
	Type    string
	Stage   string
	Message string
	Delta   string
	Sources []ChatSource
}

type ChatConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *models.ChatConversation) error
	GetConversationByID(ctx context.Context, userID uint, conversationID uint) (*models.ChatConversation, error)
	ListConversationsByCollection(ctx context.Context, userID uint, collectionID uint) ([]models.ChatConversation, error)
	ListMessagesByConversation(ctx context.Context, userID uint, conversationID uint) ([]models.ChatConversationEntry, error)
	DeleteConversation(ctx context.Context, userID uint, conversationID uint) error
	AddMessage(ctx context.Context, userID uint, conversationID uint, message *models.ChatConversationEntry) error
}

type ChatUseCase interface {
	StreamCollectionChat(ctx context.Context, userID uint, request ChatRequest, onEvent func(ChatStreamEvent) error) error
	CreateConversation(ctx context.Context, userID uint, collectionID uint, title string) (*models.ChatConversation, error)
	ListConversations(ctx context.Context, userID uint, collectionID uint) ([]models.ChatConversation, error)
	GetConversationMessages(ctx context.Context, userID uint, conversationID uint) (*models.ChatConversation, []models.ChatConversationEntry, error)
	DeleteConversation(ctx context.Context, userID uint, conversationID uint) error
}
