package db

import (
	"errors"
	"fmt"
	"log"
	"os"
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
