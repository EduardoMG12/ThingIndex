package models

import (
	"time"

	"gorm.io/gorm"
)

// WorkspaceViewState e o estado de visualizacao do workspace de uma colecao
// para um usuario: zoom, posicao do canvas e dos quadros, modo e pagina de
// cada PDF, marcadores visiveis. Preferencia de tela, nao dado da tese.
type WorkspaceViewState struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	UserID       uint           `gorm:"not null;index;uniqueIndex:idx_workspace_view_states_user_collection" json:"user_id"`
	User         *User          `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	CollectionID uint           `gorm:"not null;uniqueIndex:idx_workspace_view_states_user_collection" json:"collection_id"`
	State        map[string]any `gorm:"type:jsonb;serializer:json" json:"state"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ParagraphHighlight e uma marca colorida que o usuario poe num paragrafo no
// workspace. E anotacao, nao recuperacao: nada aqui e lido pelo caminho de
// contexto ou de conexoes (principio II da constituicao). Uma marca por
// paragrafo por usuario; a cor e uma das cinco chaves de domain.HighlightColors.
type ParagraphHighlight struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	UserID          uint           `gorm:"not null;index;uniqueIndex:idx_paragraph_highlights_user_node" json:"user_id"`
	User            *User          `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	CollectionID    uint           `gorm:"not null;index" json:"collection_id"`
	DocumentID      uint           `gorm:"not null;index" json:"document_id"`
	ParagraphNodeID uint           `gorm:"not null;uniqueIndex:idx_paragraph_highlights_user_node" json:"paragraph_node_id"`
	ParagraphNode   *ParagraphNode `gorm:"foreignKey:ParagraphNodeID;constraint:OnDelete:CASCADE" json:"-"`
	Color           string         `gorm:"size:16;not null" json:"color"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
