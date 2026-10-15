//go:build integration

package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Base antiga sem FK e com um paragrafo orfao: o bootstrap tem de limpar o orfao,
// criar a FK e manter a linha que tem documento.
func TestBootstrapRemoveOrfaoEstruturalAntesDeCriarAChaveEstrangeira(t *testing.T) {
	host := os.Getenv("THINGINDEX_TEST_DB_HOST")
	if host == "" {
		t.Skip("THINGINDEX_TEST_DB_* nao configurado")
	}
	base := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, os.Getenv("THINGINDEX_TEST_DB_PORT"), os.Getenv("THINGINDEX_TEST_DB_USER"),
		os.Getenv("THINGINDEX_TEST_DB_PASSWORD"), os.Getenv("THINGINDEX_TEST_DB_NAME"))
	cfg := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}
	admin, err := gorm.Open(postgres.Open(base), cfg)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("bootstrap_orfao_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE") })

	db, err := gorm.Open(postgres.Open(base+" search_path="+schema), cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Base "antiga": o esquema minimo das duas tabelas, sem constraint.
	for _, sql := range []string{
		`CREATE TABLE documents (id bigserial PRIMARY KEY)`,
		`CREATE TABLE paragraph_nodes (id bigserial PRIMARY KEY, document_id bigint NOT NULL, sequence_order bigint NOT NULL, content text NOT NULL)`,
		`INSERT INTO documents (id) VALUES (1)`,
		`INSERT INTO paragraph_nodes (document_id, sequence_order, content) VALUES (1, 0, 'com origem'), (42, 0, 'orfao')`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	if err := removeOrphanStructuralRows(db); err != nil {
		t.Fatalf("limpeza de orfaos: %v", err)
	}
	var restantes []string
	db.Raw(`SELECT content FROM paragraph_nodes ORDER BY id`).Scan(&restantes)
	if len(restantes) != 1 || restantes[0] != "com origem" {
		t.Fatalf("esperava so a linha com documento, restou %v", restantes)
	}
	// Idempotente: a segunda passada nao encontra nada.
	if err := removeOrphanStructuralRows(db); err != nil {
		t.Fatalf("segunda passada: %v", err)
	}
	if err := db.Exec(`ALTER TABLE paragraph_nodes ADD CONSTRAINT fk_teste FOREIGN KEY (document_id) REFERENCES documents(id)`).Error; err != nil {
		t.Fatalf("a FK deveria ser criavel depois da limpeza: %v", err)
	}
}

// Mesmo cenario para users: documento de usuario inexistente (com paragrafo
// pendurado), colecao sem dono, catalogo global (user_id NULL) e cache sem dono.
func TestBootstrapRemoveLinhaSemDonoAntesDeCriarAChaveEstrangeiraParaUsuario(t *testing.T) {
	host := os.Getenv("THINGINDEX_TEST_DB_HOST")
	if host == "" {
		t.Skip("THINGINDEX_TEST_DB_* nao configurado")
	}
	base := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, os.Getenv("THINGINDEX_TEST_DB_PORT"), os.Getenv("THINGINDEX_TEST_DB_USER"),
		os.Getenv("THINGINDEX_TEST_DB_PASSWORD"), os.Getenv("THINGINDEX_TEST_DB_NAME"))
	cfg := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}
	admin, err := gorm.Open(postgres.Open(base), cfg)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("bootstrap_sem_dono_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE") })

	db, err := gorm.Open(postgres.Open(base+" search_path="+schema), cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Base "antiga": FK de paragrafo para documento ja existe, para users nao.
	for _, sql := range []string{
		`CREATE TABLE users (id bigserial PRIMARY KEY)`,
		`CREATE TABLE collections (id bigserial PRIMARY KEY, user_id bigint NOT NULL, name text NOT NULL)`,
		`CREATE TABLE documents (id bigserial PRIMARY KEY, user_id bigint NOT NULL, title text NOT NULL)`,
		`CREATE TABLE paragraph_nodes (id bigserial PRIMARY KEY, document_id bigint NOT NULL REFERENCES documents(id) ON DELETE RESTRICT, content text NOT NULL)`,
		`CREATE TABLE provider_model_catalogs (id bigserial PRIMARY KEY, user_id bigint, model_id text NOT NULL)`,
		`CREATE TABLE enrichment_cache_entries (id bigserial PRIMARY KEY, user_id bigint NOT NULL DEFAULT 0, summary text NOT NULL)`,
		`INSERT INTO users (id) VALUES (1)`,
		`INSERT INTO collections (user_id, name) VALUES (1, 'com dono'), (42, 'sem dono')`,
		`INSERT INTO documents (id, user_id, title) VALUES (1, 1, 'com dono'), (2, 42, 'sem dono')`,
		`INSERT INTO paragraph_nodes (document_id, content) VALUES (1, 'com dono'), (2, 'sem dono')`,
		`INSERT INTO provider_model_catalogs (user_id, model_id) VALUES (NULL, 'global'), (1, 'com dono'), (42, 'sem dono')`,
		`INSERT INTO enrichment_cache_entries (user_id, summary) VALUES (1, 'com dono'), (0, 'antigo')`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	if err := removeOrphanUserRows(db); err != nil {
		t.Fatalf("limpeza de linhas sem dono: %v", err)
	}
	for tabela, esperado := range map[string]string{
		"collections":              "com dono",
		"documents":                "com dono",
		"paragraph_nodes":          "com dono",
		"enrichment_cache_entries": "com dono",
	} {
		coluna := "name"
		switch tabela {
		case "documents":
			coluna = "title"
		case "paragraph_nodes":
			coluna = "content"
		case "enrichment_cache_entries":
			coluna = "summary"
		}
		var restantes []string
		db.Raw(`SELECT ` + coluna + ` FROM ` + tabela + ` ORDER BY id`).Scan(&restantes)
		if len(restantes) != 1 || restantes[0] != esperado {
			t.Fatalf("%s: esperava so a linha com dono, restou %v", tabela, restantes)
		}
	}
	var catalogo []string
	db.Raw(`SELECT model_id FROM provider_model_catalogs ORDER BY id`).Scan(&catalogo)
	if len(catalogo) != 2 || catalogo[0] != "global" || catalogo[1] != "com dono" {
		t.Fatalf("catalogo: esperava a linha global e a com dono, restou %v", catalogo)
	}
	// Idempotente: a segunda passada nao encontra nada.
	if err := removeOrphanUserRows(db); err != nil {
		t.Fatalf("segunda passada: %v", err)
	}
	for _, tabela := range []string{"collections", "documents", "provider_model_catalogs", "enrichment_cache_entries"} {
		if err := db.Exec(`ALTER TABLE ` + tabela + ` ADD CONSTRAINT fk_teste_` + tabela + ` FOREIGN KEY (user_id) REFERENCES users(id)`).Error; err != nil {
			t.Fatalf("%s: a FK deveria ser criavel depois da limpeza: %v", tabela, err)
		}
	}
}
