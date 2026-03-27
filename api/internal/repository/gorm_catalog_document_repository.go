package repository

import (
	"context"
	"strings"
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

func (r *GormCatalogDocumentRepository) ListDocumentAuditEvents(ctx context.Context, userID uint, filter domain.DocumentAuditFilter) ([]models.DocumentAuditEvent, error) {
	result, err := r.QueryDocumentAuditEvents(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (r *GormCatalogDocumentRepository) QueryDocumentAuditEvents(ctx context.Context, userID uint, filter domain.DocumentAuditFilter) (*domain.DocumentAuditQueryResult, error) {
	itemsQuery, err := r.buildDocumentAuditQuery(ctx, userID, filter, true)
	if err != nil {
		return nil, err
	}

	summaryQuery, err := r.buildDocumentAuditQuery(ctx, userID, filter, false)
	if err != nil {
		return nil, err
	}

	limit := clampAuditLimit(filter.Limit)
	var events []models.DocumentAuditEvent
	if err := itemsQuery.
		Order("created_at DESC, id DESC").
		Limit(limit + 1).
		Find(&events).Error; err != nil {
		return nil, err
	}

	pageInfo := domain.AuditPageInfo{
		Limit: limit,
	}
	if len(events) > limit {
		lastItem := events[limit-1]
		pageInfo.HasNext = true
		pageInfo.NextCursor = domain.EncodeAuditCursor(domain.AuditCursor{
			CreatedAt: lastItem.CreatedAt,
			ID:        lastItem.ID,
		})
		events = events[:limit]
	}
	pageInfo.Returned = len(events)

	summary, err := r.buildDocumentAuditSummary(summaryQuery)
	if err != nil {
		return nil, err
	}

	return &domain.DocumentAuditQueryResult{
		Items:    events,
		Summary:  summary,
		PageInfo: pageInfo,
	}, nil
}

func (r *GormCatalogDocumentRepository) buildDocumentAuditQuery(ctx context.Context, userID uint, filter domain.DocumentAuditFilter, includeCursor bool) (*gorm.DB, error) {
	query := r.db.WithContext(ctx).
		Model(&models.DocumentAuditEvent{}).
		Where("user_id = ?", userID)

	if filter.DocumentID != nil {
		query = query.Where("document_id = ?", *filter.DocumentID)
	}
	if filter.CollectionID != nil {
		query = query.Where("collection_id = ?", *filter.CollectionID)
	}
	if action := strings.ToUpper(strings.TrimSpace(filter.Action)); action != "" {
		query = query.Where("action = ?", action)
	}
	if search := escapeLikePattern(filter.Query); search != "" {
		pattern := "%" + search + "%"
		query = query.Where(
			`(
				LOWER(COALESCE(title, '')) LIKE ? ESCAPE '\' OR
				LOWER(COALESCE(detail, '')) LIKE ? ESCAPE '\' OR
				LOWER(COALESCE(origin, '')) LIKE ? ESCAPE '\' OR
				LOWER(COALESCE(correlation_id, '')) LIKE ? ESCAPE '\'
			)`,
			pattern,
			pattern,
			pattern,
			pattern,
		)
	}

	return applyCommonAuditFilters(query, filter.AuditPageFilter, includeCursor)
}

func (r *GormCatalogDocumentRepository) buildDocumentAuditSummary(query *gorm.DB) (domain.DocumentAuditSummary, error) {
	type summaryRow struct {
		TotalEvents       int64
		UniqueDocuments   int64
		UniqueCollections int64
		UniqueRequests    int64
		LastEventAt       *time.Time
	}

	var row summaryRow
	if err := query.
		Session(&gorm.Session{}).
		Select(
			"COUNT(*) AS total_events",
			"COUNT(DISTINCT document_id) AS unique_documents",
			"COUNT(DISTINCT collection_id) AS unique_collections",
			"COUNT(DISTINCT NULLIF(correlation_id, '')) AS unique_requests",
			"MAX(created_at) AS last_event_at",
		).
		Scan(&row).Error; err != nil {
		return domain.DocumentAuditSummary{}, err
	}

	actionBreakdown, err := loadAuditBuckets(query, "action")
	if err != nil {
		return domain.DocumentAuditSummary{}, err
	}
	statusBreakdown, err := loadAuditBuckets(query, "status")
	if err != nil {
		return domain.DocumentAuditSummary{}, err
	}

	return domain.DocumentAuditSummary{
		TotalEvents:       row.TotalEvents,
		UniqueDocuments:   row.UniqueDocuments,
		UniqueCollections: row.UniqueCollections,
		UniqueRequests:    row.UniqueRequests,
		LastEventAt:       row.LastEventAt,
		ActionBreakdown:   actionBreakdown,
		StatusBreakdown:   statusBreakdown,
	}, nil
}

func (r *GormCatalogDocumentRepository) GetDocumentByID(ctx context.Context, id uint, userID uint) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).
		Preload("CollectionRef").
		Preload("CollectionRef.Category").
		Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
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
		Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
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
		Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
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
		Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
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
		Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
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
			Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at DESC")
			}).
			Where("id = ? AND user_id = ?", documentID, userID).
			First(&document).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		if err := tx.Exec(
			`UPDATE documents
			 SET status = ?, processing_stage = ?, status_detail = ?, last_processing_error = ?, index_started_at = NULL, indexed_at = NULL, updated_at = ?
			 WHERE id = ?`,
			domain.DocumentStatusQueued,
			"QUEUED",
			"Documento reenfileirado para nova tentativa de indexacao.",
			"",
			now,
			documentID,
		).Error; err != nil {
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

		document = models.Document{}
		return tx.
			Preload("CollectionRef").
			Preload("CollectionRef.Category").
			Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at DESC")
			}).
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
			Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at DESC")
			}).
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
			Preload("ProcessingAttempts", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at DESC")
			}).
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
		&models.DocumentProcessingAttempt{},
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
