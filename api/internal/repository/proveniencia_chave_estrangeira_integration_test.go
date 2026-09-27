//go:build integration

package repository

import (
	"errors"
	"sort"
	"testing"

	"gorm.io/gorm"

	"thing-index/internal/models"
)

// Bug de 2026-09-26: as seis tabelas de estrutura extraida tinham document_id
// NOT NULL mas nenhuma chave estrangeira, porque os modelos nao declaravam a
// associacao com Document e o GORM so cria constraint a partir dela. O artigo
// afirma que a proveniencia e garantida por chave estrangeira; este teste
// trava a afirmacao (constituicao, principio III).
func TestTabelasEstruturaisTemChaveEstrangeiraParaDocumento(t *testing.T) {
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
		  AND kcu.column_name = 'document_id'
		  AND ccu.table_name = 'documents'`).Scan(&tabelas).Error
	if err != nil {
		t.Fatalf("consultar chaves estrangeiras: %v", err)
	}
	tem := map[string]bool{}
	for _, nome := range tabelas {
		tem[nome] = true
	}

	var faltando []string
	for _, nome := range []string{"paragraph_nodes", "document_pages", "table_sections", "metadata_pairs", "document_chunks", "document_outline_nodes"} {
		if !tem[nome] {
			faltando = append(faltando, nome)
		}
	}
	sort.Strings(faltando)
	if len(faltando) > 0 {
		t.Fatalf("tabelas estruturais sem chave estrangeira para documents: %v", faltando)
	}
}

func TestParagrafoComDocumentoInexistenteERecusadoPeloBanco(t *testing.T) {
	db := openIntegrationDB(t)

	err := db.Create(&models.ParagraphNode{DocumentID: 999999, SequenceOrder: 0, Content: "sem origem"}).Error
	if err == nil {
		t.Fatal("o banco aceitou paragrafo apontando para documento inexistente")
	}
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("esperava violacao de chave estrangeira, veio: %v", err)
	}
}
