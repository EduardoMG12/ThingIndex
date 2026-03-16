package db

import (
	"fmt"

	"thing-index/internal/config"
	"thing-index/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
		config.AppConfig.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("fail to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.DocumentCategory{},
		&models.Collection{},
		&models.Document{},
		&models.DocumentPage{},
		&models.ParagraphNode{},
		&models.TableSection{},
		&models.MetadataPair{},
		&models.DocumentChunk{},
		&models.DocumentOutlineNode{},
		&models.OutboxEvent{},
		&models.UserProviderConfig{},
		&models.PricingSource{},
		&models.ModelPrice{},
	)
	if err != nil {
		return nil, fmt.Errorf("fail to run automatic migrations: %w", err)
	}

	if err := ensureDocumentCatalog(db); err != nil {
		return nil, fmt.Errorf("fail to backfill document catalog: %w", err)
	}
	if err := ensureProviderSchema(db); err != nil {
		return nil, fmt.Errorf("fail to ensure provider schema: %w", err)
	}

	return db, nil
}
