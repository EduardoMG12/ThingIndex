package repository

import (
	"context"
	"time"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type GormCatalogDocumentRepository struct {
	db *gorm.DB
}

func NewGormCatalogDocumentRepository(db *gorm.DB) domain.CatalogDocumentRepository {
	return &GormCatalogDocumentRepository{db: db}
}

func (r *GormCatalogDocumentRepository) CreateDocument(ctx context.Context, doc *models.Document) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *GormCatalogDocumentRepository) CreateDocumentWithOutbox(ctx context.Context, doc *models.Document, event *models.OutboxEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			return err
		}

		event.AggregateID = doc.ID
		if event.Payload == nil {
			event.Payload = map[string]any{}
		}
		event.Payload["document_id"] = doc.ID
		return tx.Create(event).Error
	})
}

func (r *GormCatalogDocumentRepository) GetDocumentByID(ctx context.Context, id uint, userID uint) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&doc).Error; err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentByIDForProcessing(ctx context.Context, id uint) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		First(&doc, id).Error; err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentsByUserID(ctx context.Context, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentsByCategoryID(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Joins("JOIN collections ON collections.id = documents.collection_id").
		Where("documents.user_id = ? AND collections.category_id = ?", userID, categoryID).
		Order("documents.created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentsByCollectionID(ctx context.Context, collectionID uint, userID uint) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Order("created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}

func (r *GormCatalogDocumentRepository) UpdateDocumentStatus(ctx context.Context, documentID uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("id = ?", documentID).
		Update("status", status).Error
}

func (r *GormCatalogDocumentRepository) RetryDocumentIndexing(ctx context.Context, userID uint, documentID uint) (*models.Document, error) {
	var document models.Document

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Preload("CollectionRef").
			Preload("CollectionRef.Category").
			Where("id = ? AND user_id = ?", documentID, userID).
			First(&document).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		if err := tx.Model(&models.Document{}).
			Where("id = ?", documentID).
			Updates(map[string]any{
				"status":                domain.DocumentStatusQueued,
				"processing_stage":      "QUEUED",
				"status_detail":         "Documento reenfileirado para nova tentativa de indexacao.",
				"last_processing_error": "",
				"index_started_at":      nil,
				"indexed_at":            nil,
				"updated_at":            now,
			}).Error; err != nil {
			return err
		}

		event := &models.OutboxEvent{
			AggregateType: domain.OutboxAggregateDocument,
			AggregateID:   documentID,
			EventType:     domain.OutboxEventDocumentIndexRequested,
			Status:        domain.OutboxEventStatusPending,
			Payload: map[string]any{
				"document_id": documentID,
				"user_id":     userID,
			},
			AvailableAt: now,
		}

		if err := tx.Create(event).Error; err != nil {
			return err
		}

		return tx.
			Preload("CollectionRef").
			Preload("CollectionRef.Category").
			Where("id = ?", documentID).
			First(&document).Error
	})
	if err != nil {
		return nil, err
	}

	return &document, nil
}
