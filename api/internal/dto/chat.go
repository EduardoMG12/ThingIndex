package dto

import (
	"time"

	"thing-index/internal/models"
)

type ChatMessageRequest struct {
	Role    string `json:"role" binding:"required,oneof=user assistant"`
	Content string `json:"content" binding:"required"`
}

type CreateChatConversationRequest struct {
	CollectionID uint   `json:"collection_id" binding:"required,gte=1"`
	Title        string `json:"title" binding:"omitempty,max=255"`
}

type UpdateChatConversationRequest struct {
	Title string `json:"title" binding:"required,max=255"`
}

type ChatStreamRequest struct {
	ConversationID uint                 `json:"conversation_id" binding:"required,gte=1"`
	CollectionID   uint                 `json:"collection_id" binding:"required,gte=1"`
	Prompt         string               `json:"prompt" binding:"required"`
	History        []ChatMessageRequest `json:"history"`
}

type ChatSourceResponse struct {
	ID         string `json:"id"`
	DocName    string `json:"doc_name"`
	Page       int    `json:"page"`
	Snippet    string `json:"snippet"`
	DocumentID uint   `json:"document_id,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
}

type ChatConversationResponse struct {
	ID            uint       `json:"id"`
	CollectionID  uint       `json:"collection_id"`
	Title         string     `json:"title"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ChatUsageResponse struct {
	PromptTokens     int      `json:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens"`
	UsageReported    bool     `json:"usage_reported"`
	ProviderType     string   `json:"provider_type,omitempty"`
	Model            string   `json:"model,omitempty"`
	EstimatedCostUSD *float64 `json:"estimated_cost_usd,omitempty"`
}

type ChatConversationMessageResponse struct {
	ID        uint                 `json:"id"`
	Role      string               `json:"role"`
	Content   string               `json:"content"`
	Sources   []ChatSourceResponse `json:"sources,omitempty"`
	Usage     *ChatUsageResponse   `json:"usage,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

type ChatConversationDetailResponse struct {
	Conversation *ChatConversationResponse         `json:"conversation"`
	Messages     []ChatConversationMessageResponse `json:"messages"`
}

func ToChatConversationResponse(conversation *models.ChatConversation) *ChatConversationResponse {
	if conversation == nil {
		return nil
	}

	return &ChatConversationResponse{
		ID:            conversation.ID,
		CollectionID:  conversation.CollectionID,
		Title:         conversation.Title,
		LastMessageAt: conversation.LastMessageAt,
		CreatedAt:     conversation.CreatedAt,
		UpdatedAt:     conversation.UpdatedAt,
	}
}

func ToChatConversationResponses(conversations []models.ChatConversation) []ChatConversationResponse {
	if len(conversations) == 0 {
		return []ChatConversationResponse{}
	}

	result := make([]ChatConversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		response := ToChatConversationResponse(&conversation)
		if response != nil {
			result = append(result, *response)
		}
	}

	return result
}

func ToChatConversationMessageResponses(messages []models.ChatConversationEntry) []ChatConversationMessageResponse {
	if len(messages) == 0 {
		return []ChatConversationMessageResponse{}
	}

	result := make([]ChatConversationMessageResponse, 0, len(messages))
	for _, message := range messages {
		sources := make([]ChatSourceResponse, 0, len(message.Sources))
		for _, source := range message.Sources {
			sources = append(sources, ChatSourceResponse{
				ID:         source.ID,
				DocName:    source.DocName,
				Page:       source.Page,
				Snippet:    source.Snippet,
				DocumentID: source.DocumentID,
				NodeID:     source.NodeID,
			})
		}

		var usage *ChatUsageResponse
		// Mensagens gravadas antes do consumo existir ficam sem o bloco.
		if message.Role == "assistant" && (message.PromptTokens > 0 || message.CompletionTokens > 0 || message.Model != "") {
			usage = &ChatUsageResponse{
				PromptTokens:     message.PromptTokens,
				CompletionTokens: message.CompletionTokens,
				UsageReported:    message.UsageReported,
				ProviderType:     message.ProviderType,
				Model:            message.Model,
				EstimatedCostUSD: message.EstimatedCostUSD,
			}
		}
		result = append(result, ChatConversationMessageResponse{
			ID:        message.ID,
			Role:      message.Role,
			Usage:     usage,
			Content:   message.Content,
			Sources:   sources,
			CreatedAt: message.CreatedAt,
			UpdatedAt: message.UpdatedAt,
		})
	}

	return result
}
