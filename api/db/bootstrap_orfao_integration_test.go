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

// A primeira subida com as chaves estrangeiras novas encontra bases antigas
// com linhas estruturais orfas; sem a limpeza previa o AutoMigrate falha ao
// criar a constraint e a API nao sobe. Este teste monta essa base antiga (as
// tabelas sem FK, com um paragrafo orfao) e prova que o bootstrap limpa o
// orfao, cria a FK e preserva a linha que tem documento.
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
