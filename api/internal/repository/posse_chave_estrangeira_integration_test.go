//go:build integration

package repository

import (
	"errors"
	"sort"
	"testing"

	"gorm.io/gorm"

	"thing-index/internal/models"
)

// O GORM so cria a FK para users se o modelo declara a associacao com User;
// sem ela o banco aceitava user_id inexistente. Confere no catalogo do Postgres.
func TestTabelasComDonoTemChaveEstrangeiraParaUsuario(t *testing.T) {
	db := openIntegrationDB(t)

	var tabelas []string
	err := db.Raw(`
		SELECT tc.table_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_name = tc.constraint_name AND kcu.table_schema = tc.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = current_schema()
		  AND kcu.column_name = 'user_id'
		  AND ccu.table_name = 'users'`).Scan(&tabelas).Error
	if err != nil {
		t.Fatalf("consultar chaves estrangeiras: %v", err)
	}
	tem := map[string]bool{}
	for _, nome := range tabelas {
		tem[nome] = true
	}

	var faltando []string
	for _, nome := range []string{
		"document_categories", "collections", "workspace_view_states", "documents",
		"document_processing_attempts", "paragraph_highlights", "document_audit_events",
		"user_provider_configs", "provider_model_catalogs", "provider_audit_events",
		"pricing_sources", "model_prices", "model_price_histories", "chat_conversations",
		"user_indexing_preferences", "enrichment_cache_entries",
	} {
		if !tem[nome] {
			faltando = append(faltando, nome)
		}
	}
	sort.Strings(faltando)
	if len(faltando) > 0 {
		t.Fatalf("tabelas com user_id sem chave estrangeira para users: %v", faltando)
	}
}

func TestColecaoComUsuarioInexistenteERecusadaPeloBanco(t *testing.T) {
	db := openIntegrationDB(t)

	err := db.Create(&models.Collection{UserID: 999999, Name: "sem dono", NormalizedName: "sem dono"}).Error
	if err == nil {
		t.Fatal("o banco aceitou colecao apontando para usuario inexistente")
	}
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("esperava violacao de chave estrangeira, veio: %v", err)
	}
}

func TestConversaComUsuarioInexistenteERecusadaPeloBanco(t *testing.T) {
	db := openIntegrationDB(t)

	err := db.Create(&models.ChatConversation{UserID: 999999, CollectionID: 1, Title: "sem dono"}).Error
	if err == nil {
		t.Fatal("o banco aceitou conversa apontando para usuario inexistente")
	}
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("esperava violacao de chave estrangeira, veio: %v", err)
	}
}
