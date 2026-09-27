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

// EnrichmentCacheEntry guarda resumo e topicos por (usuario, provedor, modelo,
// tipo, hash do texto).
//
// Era global, chaveado so por (tipo, modelo, hash), com a premissa de que o
// texto de uma pagina e o mesmo para todos. O texto e; o resumo nao: ele vem
// do LLM que cada usuario configura, e nome de modelo e base_url sao texto
// livre. Um usuario podia envenenar o cache de todos os outros (P0-1). O
// indice unico antigo (idx_enrichment_cache) e removido na migracao; as
// linhas antigas ficam sem user_id (0) e nunca mais sao lidas.
type EnrichmentCacheEntry struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserID      uint      `gorm:"not null;default:0;uniqueIndex:idx_enrichment_cache_scope,priority:1" json:"user_id"`
	Provider    string    `gorm:"size:255;not null;default:'';uniqueIndex:idx_enrichment_cache_scope,priority:2" json:"provider"`
	Model       string    `gorm:"size:128;not null;uniqueIndex:idx_enrichment_cache_scope,priority:3" json:"model"`
	Kind        string    `gorm:"size:16;not null;uniqueIndex:idx_enrichment_cache_scope,priority:4" json:"kind"`
	ContentHash string    `gorm:"size:64;not null;uniqueIndex:idx_enrichment_cache_scope,priority:5" json:"content_hash"`
	Summary     string    `gorm:"type:text;not null" json:"summary"`
	Topics      string    `gorm:"type:text" json:"topics"`
	CreatedAt   time.Time `json:"created_at"`
}
