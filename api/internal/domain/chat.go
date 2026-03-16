package domain

import "context"

type ChatMessage struct {
	Role    string
	Content string
}

type ChatRequest struct {
	CollectionID uint
	Prompt       string
	Language     string
	History      []ChatMessage
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

type ChatUseCase interface {
	StreamCollectionChat(ctx context.Context, userID uint, request ChatRequest, onEvent func(ChatStreamEvent) error) error
}
