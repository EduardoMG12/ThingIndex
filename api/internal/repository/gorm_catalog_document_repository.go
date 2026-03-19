package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"time"

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

func (r *GormCatalogDocumentRepository) CreateDocumentWithOutbox(ctx context.Context, doc *models.Document, event *models.OutboxEvent, auditEvent *models.DocumentAuditEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			return err
		}

		event.AggregateID = doc.ID
		if event.Payload == nil {
			event.Payload = map[string]any{}
		}
		event.Payload["document_id"] = doc.ID
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		if auditEvent != nil {
			documentID := doc.ID
			auditEvent.DocumentID = &documentID
			auditEvent.CollectionID = doc.CollectionID
			if auditEvent.Title == "" {
				auditEvent.Title = doc.Title
			}
			return tx.Create(auditEvent).Error
		}

		return nil
	})
}

func (r *GormCatalogDocumentRepository) CreateDocumentAuditEvents(ctx context.Context, events []models.DocumentAuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Create(&events).Error
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

func (r *GormCatalogDocumentRepository) RetryDocumentIndexing(ctx context.Context, userID uint, documentID uint, auditEvent *models.DocumentAuditEvent) (*models.Document, error) {
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

		if auditEvent != nil {
			auditEvent.DocumentID = &document.ID
			auditEvent.CollectionID = document.CollectionID
			auditEvent.Title = document.Title
			if auditEvent.UserID == 0 {
				auditEvent.UserID = document.UserID
			}
			if err := tx.Create(auditEvent).Error; err != nil {
				return err
			}
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

func (r *GormCatalogDocumentRepository) MarkDocumentOrphaned(ctx context.Context, userID uint, documentID uint, detail string) (*models.Document, error) {
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
				"status":                domain.DocumentStatusOrphaned,
				"processing_stage":      "SOURCE_MISSING",
				"status_detail":         detail,
				"last_processing_error": detail,
				"updated_at":            now,
			}).Error; err != nil {
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

func (r *GormCatalogDocumentRepository) DeleteDocument(ctx context.Context, userID uint, documentID uint, auditEvent *models.DocumentAuditEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var document models.Document
		if err := tx.Where("id = ? AND user_id = ?", documentID, userID).First(&document).Error; err != nil {
			return err
		}

		if err := deleteStructuredDocumentData(tx, []uint{documentID}); err != nil {
			return err
		}

		if err := tx.Where("aggregate_type = ? AND aggregate_id = ?", domain.OutboxAggregateDocument, documentID).Delete(&models.OutboxEvent{}).Error; err != nil {
			return err
		}

		if auditEvent != nil {
			if err := tx.Create(auditEvent).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&document).Error
	})
}

func (r *GormCatalogDocumentRepository) DeleteDocumentsByCollection(ctx context.Context, userID uint, collectionID uint, auditEvents []models.DocumentAuditEvent) (int64, error) {
	var deletedCount int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var documents []models.Document
		if err := tx.Where("user_id = ? AND collection_id = ?", userID, collectionID).Find(&documents).Error; err != nil {
			return err
		}

		documentIDs := make([]uint, 0, len(documents))
		for _, document := range documents {
			documentIDs = append(documentIDs, document.ID)
		}

		deletedCount = int64(len(documentIDs))
		if len(documentIDs) == 0 {
			return nil
		}

		if err := deleteStructuredDocumentData(tx, documentIDs); err != nil {
			return err
		}

		if err := tx.Where("aggregate_type = ? AND aggregate_id IN ?", domain.OutboxAggregateDocument, documentIDs).Delete(&models.OutboxEvent{}).Error; err != nil {
			return err
		}

		if len(auditEvents) > 0 {
			if err := tx.Create(&auditEvents).Error; err != nil {
				return err
			}
		}

		return tx.Where("id IN ?", documentIDs).Delete(&models.Document{}).Error
	})
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}

func deleteStructuredDocumentData(tx *gorm.DB, documentIDs []uint) error {
	if len(documentIDs) == 0 {
		return nil
	}

	for _, model := range []any{
		&models.DocumentPage{},
		&models.ParagraphNode{},
		&models.TableSection{},
		&models.MetadataPair{},
		&models.DocumentChunk{},
		&models.DocumentOutlineNode{},
	} {
		if err := tx.Where("document_id IN ?", documentIDs).Delete(model).Error; err != nil {
			return err
		}
	}

	return nil
}
