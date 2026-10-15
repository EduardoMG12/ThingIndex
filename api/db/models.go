package db

import "thing-index/internal/models"

// ManagedModels e a lista unica dos modelos do esquema (AutoMigrate, testes de
// integracao e cmd/schema-dbml). A ordem importa: o AutoMigrate cria as FKs na
// ordem recebida, entao a tabela referenciada vem antes.
func ManagedModels() []any {
	return []any{
		&models.User{},
		&models.DocumentCategory{},
		&models.Collection{},
		&models.WorkspaceViewState{},
		&models.Document{},
		&models.DocumentProcessingAttempt{},
		&models.DocumentPage{},
		&models.ParagraphNode{},
		&models.ParagraphHighlight{},
		&models.TableSection{},
		&models.MetadataPair{},
		&models.DocumentChunk{},
		&models.DocumentOutlineNode{},
		&models.OutboxEvent{},
		&models.DocumentAuditEvent{},
		&models.UserProviderConfig{},
		&models.ProviderModelCatalog{},
		&models.ProviderAuditEvent{},
		&models.PricingSource{},
		&models.ModelPrice{},
		&models.ModelPriceHistory{},
		&models.ChatConversation{},
		&models.ChatConversationEntry{},
		&models.UserIndexingPreferences{},
		&models.EnrichmentCacheEntry{},
	}
}
