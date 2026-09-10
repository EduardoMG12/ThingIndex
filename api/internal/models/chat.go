package models

import (
	"time"

	"gorm.io/gorm"
)

// SourceBox e a caixa citada no espaco da pagina do PDF (origem no canto
// superior esquerdo, pontos), a mesma de metadata.box dos paragrafos. Vai
// gravada na fonte para o grifo sobreviver a uma reindexacao: paragrafo e
// secao ganham ids novos, a caixa na pagina nao muda.
type SourceBox struct {
	XMin       float64 `json:"x_min"`
	YMin       float64 `json:"y_min"`
	XMax       float64 `json:"x_max"`
	YMax       float64 `json:"y_max"`
	PageWidth  float64 `json:"page_width"`
	PageHeight float64 `json:"page_height"`
}

type ChatSourceReference struct {
	ID           string   `json:"id"`
	DocName      string   `json:"doc_name"`
	Page         int      `json:"page"`
	Snippet      string   `json:"snippet"`
	DocumentID   uint     `json:"document_id,omitempty"`
	NodeID       string   `json:"node_id,omitempty"`
	ParagraphID  uint     `json:"paragraph_id,omitempty"`
	Lines        []int    `json:"lines,omitempty"`
	MatchedTerms []string `json:"matched_terms,omitempty"`
	// Pagina seguinte quando o texto da fonte continua nela (quebra de paragrafo na virada).
	ContinuesOnPage int `json:"continues_on_page,omitempty"`
	// Geometria e versao do indice no momento da resposta.
	Box       *SourceBox   `json:"box,omitempty"`
	LineBoxes [][4]float64 `json:"line_boxes,omitempty"`
	IndexedAt *time.Time   `json:"indexed_at,omitempty"`
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
