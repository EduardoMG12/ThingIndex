package db

import "gorm.io/gorm"

// legacyEnrichmentCacheIndex e o indice unico global (kind, model,
// content_hash) do cache de enriquecimento.
const legacyEnrichmentCacheIndex = "idx_enrichment_cache"

// ensureEnrichmentCacheScope remove o indice global antigo, que o AutoMigrate nao
// remove (com ele, o ON CONFLICT DO NOTHING descartaria a entrada do segundo
// usuario), e apaga as linhas antigas (user_id = 0).
func ensureEnrichmentCacheScope(db *gorm.DB) error {
	if err := dropLegacyIndexIfExists(db, legacyEnrichmentCacheIndex); err != nil {
		return err
	}
	return db.Exec("DELETE FROM enrichment_cache_entries WHERE user_id = 0").Error
}
