package db

import (
	"fmt"
	"log"

	"thing-index/internal/config"
	"thing-index/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("fail to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.DocumentCategory{},
		&models.Collection{},
		&models.Document{},
		&models.ParagraphNode{},
		&models.TableSection{},
		&models.MetadataPair{},
		&models.UserProviderConfig{},
	)
	if err != nil {
		log.Printf("Warning: fail to run automatic migrations: %v\n", err)
	}

	if err := ensureDocumentCatalog(db); err != nil {
		log.Printf("Warning: fail to backfill document catalog: %v\n", err)
	}

	return db, nil
}
