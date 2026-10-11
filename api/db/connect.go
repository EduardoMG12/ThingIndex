package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"thing-index/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const schemaMigrationLockID int64 = 8452314179902

// InsertBatchSize e o numero maximo de linhas por INSERT multi-linha.
//
// MOTIVO: sem lote, o GORM gera um unico INSERT para o slice inteiro e o pgx
// recusa acima de 65535 parametros. ParagraphNode tem 8 colunas, entao um
// documento com mais de ~8 mil paragrafos (um PDF de ~400 paginas) falhava em
// FAILED_PERSISTING_INDEX e nunca indexava. 500 linhas x a tabela mais larga
// fica muito abaixo do teto; o teste trava essa conta.
const InsertBatchSize = 500

func newGormConfig() *gorm.Config {
	return &gorm.Config{
		TranslateError:  true,
		CreateBatchSize: InsertBatchSize,
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	}
}

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
		config.AppConfig.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), newGormConfig())
	if err != nil {
		return nil, fmt.Errorf("fail to connect to database: %w", err)
	}

	if err := bootstrapSchema(db); err != nil {
		return nil, err
	}

	return db, nil
}

func bootstrapSchema(db *gorm.DB) error {
	return withSchemaMigrationLock(db, func(tx *gorm.DB) error {
		if err := removeOrphanStructuralRows(tx); err != nil {
			return fmt.Errorf("fail to remove orphan structural rows: %w", err)
		}
		if err := removeOrphanUserRows(tx); err != nil {
			return fmt.Errorf("fail to remove rows without owner: %w", err)
		}
		if err := tx.AutoMigrate(ManagedModels()...); err != nil {
			return fmt.Errorf("fail to run automatic migrations: %w", err)
		}

		if err := ensureDocumentCatalog(tx); err != nil {
			return fmt.Errorf("fail to backfill document catalog: %w", err)
		}
		if err := ensureProviderSchema(tx); err != nil {
			return fmt.Errorf("fail to ensure provider schema: %w", err)
		}
		if err := ensureEnrichmentCacheScope(tx); err != nil {
			return fmt.Errorf("fail to scope enrichment cache: %w", err)
		}

		return nil
	})
}

// structuralTables sao as tabelas cuja linha so tem sentido com o documento de
// origem. A ordem nao importa: nenhuma referencia a outra.
var structuralTables = []string{
	"paragraph_nodes",
	"document_pages",
	"table_sections",
	"metadata_pairs",
	"document_chunks",
	"document_outline_nodes",
}

// removeOrphanStructuralRows roda ANTES do AutoMigrate, porque criar a chave
// estrangeira falha se ja existir linha apontando para documento inexistente.
// Apagar e a unica saida coerente: linha estrutural sem documento nao tem
// proveniencia, e inventar um documento para ela seria pior. O numero de
// linhas removidas vai para o log por tabela, para a remocao ficar auditavel.
//
// Idempotente: numa base que ja tem as FKs nao ha orfao e nada e apagado. Numa
// base nova as tabelas ainda nao existem e o passo e pulado.
func removeOrphanStructuralRows(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("documents") {
		return nil
	}
	for _, table := range structuralTables {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		res := tx.Exec(`DELETE FROM ` + table + ` t WHERE NOT EXISTS (SELECT 1 FROM documents d WHERE d.id = t.document_id)`)
		if res.Error != nil {
			return fmt.Errorf("%s: %w", table, res.Error)
		}
		if res.RowsAffected > 0 {
			log.Printf("migracao: %d linha(s) de %s sem documento de origem removidas antes de criar a chave estrangeira", res.RowsAffected, table)
		}
	}
	return nil
}

// userOwnedTables sao as tabelas com coluna user_id, cada uma com chave
// estrangeira para users. A ordem importa: as folhas vem antes de quem elas
// referenciam, para que apagar uma linha sem dono nunca esbarre numa chave
// estrangeira RESTRICT de outra tabela. dependentes lista o que precisa sair
// antes da propria linha (filhos sem coluna user_id).
var userOwnedTables = []struct {
	tabela      string
	dependentes []string
}{
	{tabela: "paragraph_highlights"},
	{tabela: "workspace_view_states"},
	{tabela: "document_processing_attempts"},
	{tabela: "document_audit_events"},
	{tabela: "chat_conversations", dependentes: []string{"chat_conversation_entries.conversation_id"}},
	{tabela: "documents", dependentes: append(structuralColumns("document_id"), "document_processing_attempts.document_id")},
	{tabela: "collections"},
	{tabela: "document_categories"},
	{tabela: "provider_audit_events"},
	{tabela: "provider_model_catalogs"},
	{tabela: "user_provider_configs"},
	{tabela: "model_price_histories"},
	{tabela: "model_prices"},
	{tabela: "pricing_sources"},
	{tabela: "user_indexing_preferences"},
	{tabela: "enrichment_cache_entries"},
}

func structuralColumns(column string) []string {
	out := make([]string, 0, len(structuralTables))
	for _, table := range structuralTables {
		out = append(out, table+"."+column)
	}
	return out
}

// removeOrphanUserRows roda ANTES do AutoMigrate pelo mesmo motivo de
// removeOrphanStructuralRows: criar a chave estrangeira para users falha se ja
// existir linha com user_id de usuario inexistente. O sistema nao exclui conta
// (users so tem exclusao logica, e a linha continua la), entao numa base
// coerente nao ha nada a apagar; a limpeza cobre bases montadas a mao ou
// restauradas pela metade. Linha sem dono nao e de ninguem: nenhuma consulta a
// le, porque todas filtram por user_id. O numero de linhas removidas vai para o
// log por tabela.
//
// user_id NULL (provider_model_catalogs, catalogo global) nao e orfao: a chave
// estrangeira aceita NULL e a linha fica. O cache de enriquecimento anterior
// ao escopo por usuario nao tem a coluna user_id; o AutoMigrate a criaria com 0
// e a chave estrangeira falharia, entao essas linhas saem antes (o bootstrap
// ja as apagava depois, em ensureEnrichmentCacheScope).
//
// Idempotente: numa base que ja tem as FKs nao ha orfao e nada e apagado.
func removeOrphanUserRows(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("users") {
		return nil
	}
	const semDono = `t.user_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users u WHERE u.id = t.user_id)`
	for _, item := range userOwnedTables {
		if !tx.Migrator().HasTable(item.tabela) {
			continue
		}
		if !tx.Migrator().HasColumn(item.tabela, "user_id") {
			if item.tabela != "enrichment_cache_entries" {
				continue
			}
			res := tx.Exec(`DELETE FROM enrichment_cache_entries`)
			if res.Error != nil {
				return fmt.Errorf("%s: %w", item.tabela, res.Error)
			}
			if res.RowsAffected > 0 {
				log.Printf("migracao: %d linha(s) de enrichment_cache_entries sem user_id removidas antes de criar a chave estrangeira", res.RowsAffected)
			}
			continue
		}
		for _, dep := range item.dependentes {
			depTable, depColumn, _ := strings.Cut(dep, ".")
			if !tx.Migrator().HasTable(depTable) {
				continue
			}
			res := tx.Exec(`DELETE FROM ` + depTable + ` WHERE ` + depColumn + ` IN (SELECT t.id FROM ` + item.tabela + ` t WHERE ` + semDono + `)`)
			if res.Error != nil {
				return fmt.Errorf("%s: %w", depTable, res.Error)
			}
			if res.RowsAffected > 0 {
				log.Printf("migracao: %d linha(s) de %s pendurada(s) em %s sem dono removidas antes de criar a chave estrangeira", res.RowsAffected, depTable, item.tabela)
			}
		}
		res := tx.Exec(`DELETE FROM ` + item.tabela + ` t WHERE ` + semDono)
		if res.Error != nil {
			return fmt.Errorf("%s: %w", item.tabela, res.Error)
		}
		if res.RowsAffected > 0 {
			log.Printf("migracao: %d linha(s) de %s com user_id de usuario inexistente removidas antes de criar a chave estrangeira", res.RowsAffected, item.tabela)
		}
	}
	return nil
}

func withSchemaMigrationLock(db *gorm.DB, fn func(tx *gorm.DB) error) (err error) {
	if lockErr := db.Exec("SELECT pg_advisory_lock(?)", schemaMigrationLockID).Error; lockErr != nil {
		return fmt.Errorf("fail to acquire schema migration lock: %w", lockErr)
	}

	defer func() {
		if unlockErr := db.Exec("SELECT pg_advisory_unlock(?)", schemaMigrationLockID).Error; unlockErr != nil {
			releaseErr := fmt.Errorf("fail to release schema migration lock: %w", unlockErr)
			if err == nil {
				err = releaseErr
				return
			}
			err = errors.Join(err, releaseErr)
		}
	}()

	return fn(db)
}
