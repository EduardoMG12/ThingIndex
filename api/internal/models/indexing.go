package models

import "time"

// UserIndexingPreferences: o que a LLM gera na indexacao dos documentos do
// usuario. Sem tag default nos booleanos de proposito: com `default:true` o
// GORM gravaria true quando o valor e false.
type UserIndexingPreferences struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"not null;uniqueIndex" json:"user_id"`
	DocumentSynopsis bool      `gorm:"not null" json:"document_synopsis"`
	PageSummaries    bool      `gorm:"not null" json:"page_summaries"`
	SectionSummaries bool      `gorm:"not null" json:"section_summaries"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EnrichmentCacheEntry guarda resumo e topicos por (tipo, modelo, hash do
// texto). Global, nao por usuario: o texto de uma pagina e o mesmo para todos.
type EnrichmentCacheEntry struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Kind        string    `gorm:"size:16;not null;uniqueIndex:idx_enrichment_cache,priority:1" json:"kind"`
	Model       string    `gorm:"size:128;not null;uniqueIndex:idx_enrichment_cache,priority:2" json:"model"`
	ContentHash string    `gorm:"size:64;not null;uniqueIndex:idx_enrichment_cache,priority:3" json:"content_hash"`
	Summary     string    `gorm:"type:text;not null" json:"summary"`
	Topics      string    `gorm:"type:text" json:"topics"`
	CreatedAt   time.Time `json:"created_at"`
}
