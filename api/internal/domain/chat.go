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
