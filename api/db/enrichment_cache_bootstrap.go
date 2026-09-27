package db

import "gorm.io/gorm"

// legacyEnrichmentCacheIndex e o indice unico global (kind, model,
// content_hash) do cache de enriquecimento.
const legacyEnrichmentCacheIndex = "idx_enrichment_cache"

// ensureEnrichmentCacheScope conclui a migracao do cache de enriquecimento
// para o escopo por usuario e provedor (P0-1).
//
// O AutoMigrate cria as colunas user_id/provider e o indice novo, mas nao
// remove o indice antigo: com ele de pe, dois usuarios com o mesmo modelo e o
// mesmo texto colidiriam, e o ON CONFLICT DO NOTHING descartaria em silencio
// a entrada do segundo. As linhas antigas (user_id = 0) vieram de um cache
// que qualquer usuario podia envenenar; sao apagadas em vez de herdadas.
func ensureEnrichmentCacheScope(db *gorm.DB) error {
	if err := dropLegacyIndexIfExists(db, legacyEnrichmentCacheIndex); err != nil {
		return err
	}
	return db.Exec("DELETE FROM enrichment_cache_entries WHERE user_id = 0").Error
}
