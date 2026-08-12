package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const schemaMigrationLockID int64 = 8452314179902

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
		config.AppConfig.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})
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
		if err := tx.AutoMigrate(
			&models.User{},
			&models.DocumentCategory{},
			&models.Collection{},
			&models.Document{},
			&models.DocumentProcessingAttempt{},
			&models.DocumentPage{},
			&models.ParagraphNode{},
			&models.TableSection{},
			&models.MetadataPair{},
			&models.DocumentChunk{},
			&models.DocumentOutlineNode{},
			&models.OutboxEvent{},
			&models.DocumentAuditEvent{},
			&models.UserProviderConfig{},
			&models.ProviderModelCatalog{},
			&models.ProviderAuditEvent{},
			&models.PricingSource{},
			&models.ModelPrice{},
			&models.ModelPriceHistory{},
			&models.ChatConversation{},
			&models.ChatConversationEntry{},
		); err != nil {
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
