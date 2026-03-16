package dto

type ChatMessageRequest struct {
	Role    string `json:"role" binding:"required,oneof=user assistant"`
	Content string `json:"content" binding:"required"`
}

type ChatStreamRequest struct {
	CollectionID uint                 `json:"collection_id" binding:"required,gte=1"`
	Prompt       string               `json:"prompt" binding:"required"`
	History      []ChatMessageRequest `json:"history"`
}
