package db

import "thing-index/internal/models"

// ManagedModels e a fonte unica da verdade sobre quais modelos compoem o
// esquema do ThingIndex.
//
// Antes desta funcao a lista existia copiada em quatro lugares: o AutoMigrate
// de producao, o helper de integracao do repository, o testutil e o setup do
// E2E de rotas. As copias divergiram na pratica -- o comentario em
// integration_test_helpers_test.go registra o caso em que
// DocumentProcessingAttempt faltava e DeleteDocumentsByCollection falhava com
// `relation "document_processing_attempts" does not exist`, um teste vermelho
// que ninguem via porque a suite de integracao nao roda sozinha.
//
// Manter uma lista so troca esse modo de falha silencioso por um erro de
// compilacao. Tambem e a entrada do gerador de DBML do Apendice A
// (cmd/schema-dbml), entao o esquema documentado no artigo nao consegue
// dessincronizar do esquema real.
//
// A ordem importa: o AutoMigrate cria as chaves estrangeiras na ordem em que
// recebe os modelos, entao a tabela referenciada precisa vir antes de quem a
// referencia.
func ManagedModels() []any {
	return []any{
		&models.User{},
		&models.DocumentCategory{},
		&models.Collection{},
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
	}
}
