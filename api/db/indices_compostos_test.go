package db

import (
	"reflect"
	"sync"
	"testing"

	"thing-index/internal/models"

	"gorm.io/gorm/schema"
)

// Indices compostos dos filtros quentes. A ordem das colunas importa para o ORDER BY.
func TestIndicesCompostosDosFiltrosQuentes(t *testing.T) {
	casos := []struct {
		modelo   any
		indice   string
		colunas  []string
		consulta string
	}{
		{&models.Document{}, "idx_documents_user_collection_created", []string{"user_id", "collection_id", "created_at"},
			"documents WHERE user_id AND collection_id ORDER BY created_at"},
		{&models.DocumentAuditEvent{}, "idx_document_audit_events_user_created", []string{"user_id", "created_at", "id"},
			"document_audit_events WHERE user_id ORDER BY created_at DESC, id DESC"},
		{&models.OutboxEvent{}, "idx_outbox_events_claim", []string{"status", "available_at", "id"},
			"outbox_events WHERE status IN ? AND available_at <= ? ORDER BY available_at, id"},
		{&models.ChatConversation{}, "idx_chat_conversations_user_collection", []string{"user_id", "collection_id"},
			"chat_conversations WHERE user_id AND collection_id"},
	}

	cache := &sync.Map{}
	for _, caso := range casos {
		s, err := schema.Parse(caso.modelo, cache, schema.NamingStrategy{})
		if err != nil {
			t.Fatalf("parse %T: %v", caso.modelo, err)
		}
		var encontrado *schema.Index
		for _, idx := range s.ParseIndexes() {
			if idx.Name == caso.indice {
				encontrado = idx
			}
		}
		if encontrado == nil {
			t.Errorf("%s: indice %s ausente (%s)", s.Table, caso.indice, caso.consulta)
			continue
		}
		var colunas []string
		for _, campo := range encontrado.Fields {
			colunas = append(colunas, campo.DBName)
		}
		if !reflect.DeepEqual(colunas, caso.colunas) {
			t.Errorf("%s: %s tem colunas %v, esperava %v", s.Table, caso.indice, colunas, caso.colunas)
		}
	}
}
