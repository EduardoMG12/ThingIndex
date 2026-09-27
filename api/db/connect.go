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
		if err := tx.AutoMigrate(ManagedModels()...); err != nil {
			return fmt.Errorf("fail to run automatic migrations: %w", err)
		}

		if err := ensureDocumentCatalog(tx); err != nil {
			return fmt.Errorf("fail to backfill document catalog: %w", err)
		}
		if err := ensureProviderSchema(tx); err != nil {
			return fmt.Errorf("fail to ensure provider schema: %w", err)
		}

		return nil
	})
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
