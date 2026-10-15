package db

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// Teto de parametros por statement do protocolo do Postgres (pgx recusa acima).
const postgresMaxParametros = 65535

// Sem lote, documentos grandes estouravam 65535 parametros. O lote vezes a tabela
// mais larga tem de caber no teto.
func TestConexaoInsereEmLotesQueCabemNoLimiteDeParametrosDoPostgres(t *testing.T) {
	t.Parallel()

	cfg := newGormConfig()
	if cfg.CreateBatchSize != InsertBatchSize || InsertBatchSize <= 0 {
		t.Fatalf("CreateBatchSize=%d, esperava InsertBatchSize=%d", cfg.CreateBatchSize, InsertBatchSize)
	}

	cache := &sync.Map{}
	for _, model := range ManagedModels() {
		s, err := schema.Parse(model, cache, schema.NamingStrategy{})
		if err != nil {
			t.Fatalf("parse %T: %v", model, err)
		}
		colunas := len(s.DBNames)
		if parametros := InsertBatchSize * colunas; parametros > postgresMaxParametros {
			t.Fatalf("%s: lote de %d x %d colunas = %d parametros, acima de %d",
				s.Table, InsertBatchSize, colunas, parametros, postgresMaxParametros)
		}
	}
}
