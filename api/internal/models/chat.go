package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatSourceReference struct {
	ID         string `json:"id"`
	DocName    string `json:"doc_name"`
	Page       int    `json:"page"`
	Snippet    string `json:"snippet"`
	DocumentID uint   `json:"document_id,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
}

type ChatConversation struct {
	ID            uint                    `gorm:"primarykey" json:"id"`
	UserID        uint                    `gorm:"not null;index" json:"user_id"`
	CollectionID  uint                    `gorm:"not null;index" json:"collection_id"`
	Title         string                  `gorm:"size:255;not null" json:"title"`
	LastMessageAt *time.Time              `gorm:"index" json:"last_message_at,omitempty"`
	Messages      []ChatConversationEntry `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	DeletedAt     gorm.DeletedAt          `gorm:"index" json:"-"`
}

type ChatConversationEntry struct {
	ID             uint                  `gorm:"primarykey" json:"id"`
	ConversationID uint                  `gorm:"not null;index" json:"conversation_id"`
	Role           string                `gorm:"size:16;not null" json:"role"`
	Content        string                `gorm:"type:text;not null" json:"content"`
	Sources        []ChatSourceReference `gorm:"type:jsonb;serializer:json" json:"sources,omitempty"`
	// Consumo da resposta, em colunas proprias (nada polimorfico no jsonb).
	PromptTokens     int            `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens int            `gorm:"not null;default:0" json:"completion_tokens"`
	UsageReported    bool           `gorm:"not null;default:false" json:"usage_reported"`
	ProviderType     string         `gorm:"size:32" json:"provider_type,omitempty"`
	Model            string         `gorm:"size:128" json:"model,omitempty"`
	EstimatedCostUSD *float64       `json:"estimated_cost_usd,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
