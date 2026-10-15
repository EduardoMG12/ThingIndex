package models

import "time"

// Sem `default:true` nos booleanos: o GORM gravaria true quando o valor e false.
type UserIndexingPreferences struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"not null;uniqueIndex" json:"user_id"`
	User             *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	DocumentSynopsis bool      `gorm:"not null" json:"document_synopsis"`
	PageSummaries    bool      `gorm:"not null" json:"page_summaries"`
	SectionSummaries bool      `gorm:"not null" json:"section_summaries"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EnrichmentCacheEntry guarda resumo e topicos por (usuario, provedor, modelo,
// tipo, hash do texto). Linhas antigas, sem user_id, nunca sao lidas.
type EnrichmentCacheEntry struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserID      uint      `gorm:"not null;default:0;uniqueIndex:idx_enrichment_cache_scope,priority:1" json:"user_id"`
	User        *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Provider    string    `gorm:"size:255;not null;default:'';uniqueIndex:idx_enrichment_cache_scope,priority:2" json:"provider"`
	Model       string    `gorm:"size:128;not null;uniqueIndex:idx_enrichment_cache_scope,priority:3" json:"model"`
	Kind        string    `gorm:"size:16;not null;uniqueIndex:idx_enrichment_cache_scope,priority:4" json:"kind"`
	ContentHash string    `gorm:"size:64;not null;uniqueIndex:idx_enrichment_cache_scope,priority:5" json:"content_hash"`
	Summary     string    `gorm:"type:text;not null" json:"summary"`
	Topics      string    `gorm:"type:text" json:"topics"`
	CreatedAt   time.Time `json:"created_at"`
}
