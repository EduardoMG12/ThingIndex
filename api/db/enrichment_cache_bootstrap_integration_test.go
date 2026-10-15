//go:build integration

package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	"thing-index/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func abrirBancoDeMigracao(t *testing.T) *gorm.DB {
	t.Helper()
	host := os.Getenv("THINGINDEX_TEST_DB_HOST")
	if host == "" {
		t.Skip("THINGINDEX_TEST_DB_* nao configurado")
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, os.Getenv("THINGINDEX_TEST_DB_PORT"), os.Getenv("THINGINDEX_TEST_DB_USER"),
		os.Getenv("THINGINDEX_TEST_DB_PASSWORD"), os.Getenv("THINGINDEX_TEST_DB_NAME"),
		os.Getenv("THINGINDEX_TEST_DB_SSLMODE"))
	cfg := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}
	admin, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		t.Fatalf("abrir banco: %v", err)
	}
	schema := fmt.Sprintf("migracao_cache_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("criar schema: %v", err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), cfg)
	if err != nil {
		t.Fatalf("abrir schema: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// Banco com o cache global antigo: a migracao tem de remover o indice unico antigo
// e apagar as linhas sem dono, senao dois usuarios com o mesmo texto colidem.
func TestMigracaoDoCacheDeEnriquecimentoRemoveIndiceGlobalELinhasSemDono(t *testing.T) {
	db := abrirBancoDeMigracao(t)

	legado := []string{
		`CREATE TABLE enrichment_cache_entries (
			id bigserial PRIMARY KEY,
			kind varchar(16) NOT NULL,
			model varchar(128) NOT NULL,
			content_hash varchar(64) NOT NULL,
			summary text NOT NULL,
			topics text,
			created_at timestamptz)`,
		`CREATE UNIQUE INDEX idx_enrichment_cache ON enrichment_cache_entries (kind, model, content_hash)`,
		`INSERT INTO enrichment_cache_entries (kind, model, content_hash, summary) VALUES ('page', 'gpt-4o-mini', 'h1', 'envenenado')`,
	}
	for _, stmt := range legado {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("montar esquema legado: %v", err)
		}
	}

	// Com a FK para users, os donos precisam existir, e as linhas legadas
	// (sem user_id) saem antes do AutoMigrate, como no bootstrap.
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("criar users: %v", err)
	}
	for _, usuario := range []uint{1, 2} {
		if err := db.Create(&models.User{ID: usuario, Name: "u", Email: fmt.Sprintf("u%d@x", usuario), Password: "x"}).Error; err != nil {
			t.Fatalf("criar usuario %d: %v", usuario, err)
		}
	}
	if err := removeOrphanUserRows(db); err != nil {
		t.Fatalf("removeOrphanUserRows: %v", err)
	}
	if err := db.AutoMigrate(&models.EnrichmentCacheEntry{}); err != nil {
		t.Fatalf("AutoMigrate sobre o esquema legado: %v", err)
	}
	if err := ensureEnrichmentCacheScope(db); err != nil {
		t.Fatalf("ensureEnrichmentCacheScope: %v", err)
	}

	if existe, err := indexExists(db, legacyEnrichmentCacheIndex); err != nil || existe {
		t.Fatalf("indice global antigo continua de pe (existe=%v err=%v)", existe, err)
	}
	if existe, err := indexExists(db, "idx_enrichment_cache_scope"); err != nil || !existe {
		t.Fatalf("indice com escopo nao foi criado (existe=%v err=%v)", existe, err)
	}
	var sobras int64
	if err := db.Table("enrichment_cache_entries").Where("user_id = 0").Count(&sobras).Error; err != nil {
		t.Fatal(err)
	}
	if sobras != 0 {
		t.Fatalf("linhas do cache global sobreviveram: %d", sobras)
	}

	// Mesmo (kind, model, hash) para dois usuarios agora coexiste.
	for _, usuario := range []uint{1, 2} {
		if err := db.Create(&models.EnrichmentCacheEntry{UserID: usuario, Provider: "OPENAI|api.openai.com", Model: "gpt-4o-mini", Kind: "page", ContentHash: "h1", Summary: "s"}).Error; err != nil {
			t.Fatalf("gravar para o usuario %d: %v", usuario, err)
		}
	}

	// Idempotente: rodar de novo no boot seguinte nao falha.
	if err := ensureEnrichmentCacheScope(db); err != nil {
		t.Fatalf("segunda execucao: %v", err)
	}
}
