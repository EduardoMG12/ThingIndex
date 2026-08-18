package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var allowedUploadMimeTypes = map[string]struct{}{
	"application/pdf": {},
	"text/html":       {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": {},
}

type documentCatalogUseCase struct {
	categoryRepo          domain.CategoryCatalogRepository
	collectionRepo        domain.CollectionRepository
	documentRepo          domain.CatalogDocumentRepository
	structuredDataRepo    domain.StructuredDataRepository
	collectionContextRepo domain.CollectionContextRepository
	storage               domain.StorageService
	storageResolver       domain.UserStorageResolver
	maxUploadSizeBytes    int64
}

func NewDocumentUseCase(
	categoryRepo domain.CategoryCatalogRepository,
	collectionRepo domain.CollectionRepository,
	documentRepo domain.CatalogDocumentRepository,
	structuredDataRepo domain.StructuredDataRepository,
	collectionContextRepo domain.CollectionContextRepository,
	storage domain.StorageService,
	storageResolver domain.UserStorageResolver,
	maxUploadSizeBytes int64,
) domain.DocumentCatalogUseCase {
	return &documentCatalogUseCase{
		categoryRepo:          categoryRepo,
		collectionRepo:        collectionRepo,
		documentRepo:          documentRepo,
		structuredDataRepo:    structuredDataRepo,
		collectionContextRepo: collectionContextRepo,
		storage:               storage,
		storageResolver:       storageResolver,
		maxUploadSizeBytes:    maxUploadSizeBytes,
	}
}

func normalizeDisplayName(name string) string {
	return strings.TrimSpace(name)
}

func normalizeLookupKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

type documentDeletionSourceResult struct {
	removed bool
	detail  string
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}

func (u *documentCatalogUseCase) ensureCategory(ctx context.Context, userID uint, name, description string) (*models.DocumentCategory, error) {
	displayName := normalizeDisplayName(name)
	if displayName == "" {
		return nil, nil
	}

	normalizedName := normalizeLookupKey(displayName)
	category, err := u.categoryRepo.GetCategoryByNormalizedName(ctx, normalizedName, userID)
	if err == nil {
		return category, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
	}

	category = &models.DocumentCategory{
		UserID:         userID,
		Name:           displayName,
		NormalizedName: normalizedName,
		Description:    strings.TrimSpace(description),
	}

	if err := u.categoryRepo.CreateCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryCreate, err)
	}

	return category, nil
}

func (u *documentCatalogUseCase) resolveCollection(ctx context.Context, userID uint, identifier string) (*models.Collection, error) {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return nil, domain.ErrCollectionRequired
	}

	if parsedID, err := strconv.ParseUint(trimmedIdentifier, 10, 32); err == nil {
		collection, findErr := u.collectionRepo.GetCollectionByID(ctx, uint(parsedID), userID)
		if findErr == nil {
			return collection, nil
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, findErr)
		}
	}

	collection, err := u.collectionRepo.GetCollectionByNormalizedName(ctx, normalizeLookupKey(trimmedIdentifier), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}

	return collection, nil
}

func (u *documentCatalogUseCase) ensureCollectionForUpload(ctx context.Context, userID uint, name string, category *models.DocumentCategory) (*models.Collection, error) {
	displayName := normalizeDisplayName(name)
	if displayName == "" {
		return nil, domain.ErrCollectionRequired
	}

	normalizedName := normalizeLookupKey(displayName)
	collection, err := u.collectionRepo.GetCollectionByNormalizedName(ctx, normalizedName, userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
		}

		collection = &models.Collection{
			UserID:         userID,
			Name:           displayName,
			NormalizedName: normalizedName,
			Metadata:       map[string]any{},
		}
		if category != nil {
			collection.CategoryID = &category.ID
			collection.Category = category
		}

		if err := u.collectionRepo.CreateCollection(ctx, collection); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrCollectionCreate, err)
		}

		return collection, nil
	}

	if category == nil {
		return collection, nil
	}

	if collection.CategoryID == nil {
		collection.CategoryID = &category.ID
		collection.Category = category
		if err := u.collectionRepo.UpdateCollection(ctx, collection); err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrCategoryUpdate, err)
		}
		return collection, nil
	}

	if *collection.CategoryID != category.ID {
		return nil, domain.ErrCollectionCategoryConflict
	}

	if collection.Category == nil {
		collection.Category = category
	}

	return collection, nil
}

func (u *documentCatalogUseCase) CreateCategory(ctx context.Context, userID uint, name, description string) (*models.DocumentCategory, error) {
	displayName := normalizeDisplayName(name)
	if displayName == "" {
		return nil, domain.ErrCategoryRequired
	}

	normalizedName := normalizeLookupKey(displayName)
	if _, err := u.categoryRepo.GetCategoryByNormalizedName(ctx, normalizedName, userID); err == nil {
		return nil, domain.ErrCategoryDuplicate
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
	}

	category := &models.DocumentCategory{
		UserID:         userID,
		Name:           displayName,
		NormalizedName: normalizedName,
		Description:    strings.TrimSpace(description),
	}

	if err := u.categoryRepo.CreateCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryCreate, err)
	}

	return category, nil
}

func (u *documentCatalogUseCase) UpdateCategory(ctx context.Context, userID, categoryID uint, name, description *string) (*models.DocumentCategory, error) {
	category, err := u.categoryRepo.GetCategoryByID(ctx, categoryID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
	}

	if name != nil {
		displayName := normalizeDisplayName(*name)
		if displayName == "" {
			return nil, domain.ErrCategoryRequired
		}

		normalizedName := normalizeLookupKey(displayName)
		existing, err := u.categoryRepo.GetCategoryByNormalizedName(ctx, normalizedName, userID)
		if err == nil && existing.ID != category.ID {
			return nil, domain.ErrCategoryDuplicate
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
		}

		category.Name = displayName
		category.NormalizedName = normalizedName
	}

	if description != nil {
		category.Description = strings.TrimSpace(*description)
	}

	if err := u.categoryRepo.UpdateCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryUpdate, err)
	}

	return category, nil
}

func (u *documentCatalogUseCase) CreateCollection(ctx context.Context, userID uint, name string, categoryName *string, metadata map[string]any) (*models.Collection, error) {
	displayName := normalizeDisplayName(name)
	if displayName == "" {
		return nil, domain.ErrCollectionRequired
	}

	normalizedName := normalizeLookupKey(displayName)
	if _, err := u.collectionRepo.GetCollectionByNormalizedName(ctx, normalizedName, userID); err == nil {
		return nil, domain.ErrCollectionDuplicate
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}

	var category *models.DocumentCategory
	var err error
	if categoryName != nil {
		category, err = u.ensureCategory(ctx, userID, *categoryName, "")
		if err != nil {
			return nil, err
		}
	}

	collection := &models.Collection{
		UserID:         userID,
		Name:           displayName,
		NormalizedName: normalizedName,
		Metadata:       cloneMetadata(metadata),
	}
	if category != nil {
		collection.CategoryID = &category.ID
		collection.Category = category
	}

	if err := u.collectionRepo.CreateCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionCreate, err)
	}

	return collection, nil
}

func (u *documentCatalogUseCase) GetCollection(ctx context.Context, userID, collectionID uint) (*models.Collection, error) {
	collection, err := u.collectionRepo.GetCollectionByID(ctx, collectionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}

	count, err := u.collectionRepo.CountDocuments(ctx, userID, collection.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}
	collection.DocumentCount = count

	return collection, nil
}

func (u *documentCatalogUseCase) UpdateCollectionCategory(ctx context.Context, userID uint, collectionIdentifier string, categoryName *string, categoryDescription string) (int64, *models.Collection, error) {
	collection, err := u.resolveCollection(ctx, userID, collectionIdentifier)
	if err != nil {
		return 0, nil, err
	}

	var category *models.DocumentCategory
	if categoryName != nil {
		category, err = u.ensureCategory(ctx, userID, *categoryName, categoryDescription)
		if err != nil {
			return 0, nil, err
		}
	}

	collection.CategoryID = nil
	collection.Category = nil
	if category != nil {
		collection.CategoryID = &category.ID
		collection.Category = category
	}

	if err := u.collectionRepo.UpdateCollection(ctx, collection); err != nil {
		return 0, nil, fmt.Errorf("%w: %v", domain.ErrCategoryUpdate, err)
	}

	count, err := u.collectionRepo.CountDocuments(ctx, userID, collection.ID)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}
	collection.DocumentCount = count

	return count, collection, nil
}

func (u *documentCatalogUseCase) UploadDocument(ctx context.Context, userID uint, title string, file io.Reader, size int64, contentType, categoryName, categoryDescription, collection string) (*models.Document, error) {
	if strings.TrimSpace(title) == "" {
		return nil, domain.ErrTitleRequired
	}
	if size <= 0 {
		return nil, domain.ErrFileRequired
	}
	if size > u.maxUploadSizeBytes {
		return nil, domain.ErrFileTooLarge
	}

	collectionName := normalizeDisplayName(collection)
	if collectionName == "" {
		return nil, domain.ErrCollectionRequired
	}

	validatedReader, resolvedContentType, err := validateUploadStream(file, contentType, title)
	if err != nil {
		return nil, err
	}

	category, err := u.ensureCategory(ctx, userID, categoryName, categoryDescription)
	if err != nil {
		return nil, err
	}

	collectionRef, err := u.ensureCollectionForUpload(ctx, userID, collectionName, category)
	if err != nil {
		return nil, err
	}

	objectName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String()[:8], extractExtension(resolvedContentType))

	storageClient := u.storage
	userStorage, providerErr := u.storageResolver.ResolveStorage(ctx, userID)
	switch {
	case providerErr == nil:
		storageClient = userStorage
	case errors.Is(providerErr, domain.ErrStorageProviderNotConfigured):
		// Use the application storage only when the user has not configured a custom storage backend.
	default:
		return nil, providerErr
	}

	fileURL, err := storageClient.UploadFile(ctx, validatedReader, size, resolvedContentType, objectName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUploadFailed, err)
	}

	doc := &models.Document{
		UserID:          userID,
		Title:           strings.TrimSpace(title),
		FileURL:         fileURL,
		Size:            size,
		MimeType:        resolvedContentType,
		Status:          domain.DocumentStatusQueued,
		ProcessingStage: "QUEUED",
		StatusDetail:    "Documento enviado e aguardando processamento pelo worker.",
		Collection:      collectionRef.Name,
		CollectionID:    &collectionRef.ID,
		CollectionRef:   collectionRef,
	}
	if category != nil {
		doc.CategoryID = &category.ID
	}

	now := time.Now().UTC()

	event := &models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload: map[string]any{
			"document_id": doc.ID,
			"user_id":     userID,
			// Carimbo lido pelo consumidor para descartar reentrega do
			// RabbitMQ de trabalho ja concluido. Ver
			// domain.DocumentIndexRequestedEvent.
			"requested_at": now,
		},
		AvailableAt: now,
	}

	auditEvent := buildDocumentAuditEvent(ctx, doc, "DOCUMENT_UPLOAD", "SUCCESS", "Documento enviado com sucesso e reenfileirado para indexacao.", map[string]any{
		"file_url":         fileURL,
		"mime_type":        resolvedContentType,
		"size_bytes":       size,
		"processing_stage": doc.ProcessingStage,
		"status":           doc.Status,
	})

	if err := u.documentRepo.CreateDocumentWithOutbox(ctx, doc, event, &auditEvent); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	return doc, nil
}

func (u *documentCatalogUseCase) ListCategories(ctx context.Context, userID uint) ([]models.DocumentCategory, error) {
	return u.categoryRepo.ListCategoriesByUserID(ctx, userID)
}

func (u *documentCatalogUseCase) ListDocuments(ctx context.Context, userID uint) ([]models.Document, error) {
	return u.documentRepo.GetDocumentsByUserID(ctx, userID)
}

func (u *documentCatalogUseCase) ListDocumentAuditEvents(ctx context.Context, userID uint, filter domain.DocumentAuditFilter) ([]models.DocumentAuditEvent, error) {
	result, err := u.QueryDocumentAuditEvents(ctx, userID, filter)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (u *documentCatalogUseCase) QueryDocumentAuditEvents(ctx context.Context, userID uint, filter domain.DocumentAuditFilter) (*domain.DocumentAuditQueryResult, error) {
	normalizedFilter, err := normalizeDocumentAuditFilter(filter)
	if err != nil {
		return nil, err
	}

	return u.documentRepo.QueryDocumentAuditEvents(ctx, userID, normalizedFilter)
}

func (u *documentCatalogUseCase) GetDocument(ctx context.Context, userID, documentID uint) (*models.Document, error) {
	doc, err := u.documentRepo.GetDocumentByID(ctx, documentID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	return doc, nil
}

func (u *documentCatalogUseCase) RetryDocumentIndexing(ctx context.Context, userID, documentID uint) (*models.Document, error) {
	auditEvent := buildDocumentAuditEvent(ctx, &models.Document{ID: documentID, UserID: userID}, "DOCUMENT_RETRY_INDEX", "SUCCESS", "Documento reenfileirado para nova tentativa de indexacao.", map[string]any{
		"processing_stage": "QUEUED",
		"status":           domain.DocumentStatusQueued,
	})

	document, err := u.documentRepo.RetryDocumentIndexing(ctx, userID, documentID, &auditEvent)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	return document, nil
}

func (u *documentCatalogUseCase) DeleteDocument(ctx context.Context, userID, documentID uint) error {
	doc, err := u.GetDocument(ctx, userID, documentID)
	if err != nil {
		return err
	}

	deleteResult, err := u.deleteDocumentSource(ctx, userID, doc.FileURL)
	if err != nil {
		detail := fmt.Sprintf("Falha ao remover o arquivo original do storage durante a exclusao do documento: %v", err)
		log.Printf("document deletion source removal failed: user=%d document=%d file=%q err=%v", userID, documentID, doc.FileURL, err)
		_ = u.documentRepo.CreateDocumentAuditEvents(ctx, []models.DocumentAuditEvent{
			buildDocumentDeletionAuditEvent(ctx, doc, "FAILED", detail, "single", map[string]any{
				"file_url": doc.FileURL,
			}),
		})
		return err
	}

	successAudit := buildDocumentDeletionAuditEvent(ctx, doc, "SUCCESS", "Documento excluido com remocao do arquivo original e limpeza dos dados derivados.", "single", map[string]any{
		"file_url":            doc.FileURL,
		"storage_delete_path": doc.FileURL,
	})
	if deleteResult.detail != "" {
		successAudit.Metadata["storage_delete_detail"] = deleteResult.detail
	}

	if err := u.documentRepo.DeleteDocument(ctx, userID, documentID, &successAudit); err != nil {
		log.Printf("document deletion repository cleanup failed after source removal: user=%d document=%d err=%v", userID, documentID, err)
		if deleteResult.removed {
			_, _ = u.documentRepo.MarkDocumentOrphaned(ctx, userID, documentID, "O arquivo original foi removido, mas a exclusao final do documento falhou antes da limpeza completa.")
		}
		_ = u.documentRepo.CreateDocumentAuditEvents(ctx, []models.DocumentAuditEvent{
			buildDocumentDeletionAuditEvent(ctx, doc, "FAILED", fmt.Sprintf("O arquivo original foi removido, mas a limpeza final do documento falhou: %v", err), "single", map[string]any{
				"file_url": doc.FileURL,
			}),
		})
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrDocumentNotFound
		}
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	log.Printf("document deleted successfully: user=%d document=%d file=%q", userID, documentID, doc.FileURL)
	return nil
}

func (u *documentCatalogUseCase) DeleteCollectionDocuments(ctx context.Context, userID, collectionID uint) (int64, *models.Collection, error) {
	collection, err := u.GetCollection(ctx, userID, collectionID)
	if err != nil {
		return 0, nil, err
	}

	documents, err := u.documentRepo.GetDocumentsByCollectionID(ctx, collectionID, userID)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	successAudits := make([]models.DocumentAuditEvent, 0, len(documents))
	failureAudits := make([]models.DocumentAuditEvent, 0)
	successfullyRemovedSources := make([]models.Document, 0, len(documents))

	for _, document := range documents {
		deleteResult, deleteErr := u.deleteDocumentSource(ctx, userID, document.FileURL)
		if deleteErr != nil {
			detail := fmt.Sprintf("Falha ao remover um arquivo do storage durante a exclusao em lote da collection: %v", deleteErr)
			log.Printf("collection document deletion source removal failed: user=%d collection=%d document=%d file=%q err=%v", userID, collectionID, document.ID, document.FileURL, deleteErr)
			failureAudits = append(failureAudits, buildDocumentDeletionAuditEvent(ctx, &document, "FAILED", detail, "collection-bulk", map[string]any{
				"collection_id": collectionID,
				"file_url":      document.FileURL,
			}))
			for _, partiallyRemoved := range successfullyRemovedSources {
				_, _ = u.documentRepo.MarkDocumentOrphaned(ctx, partiallyRemoved.UserID, partiallyRemoved.ID, "O arquivo original foi removido durante uma exclusao em lote interrompida antes da limpeza completa.")
				failureAudits = append(failureAudits, buildDocumentDeletionAuditEvent(ctx, &partiallyRemoved, "FAILED", "O arquivo original foi removido, mas a exclusao em lote foi interrompida antes da limpeza final do documento.", "collection-bulk", map[string]any{
					"collection_id": collectionID,
					"file_url":      partiallyRemoved.FileURL,
				}))
			}
			_ = u.documentRepo.CreateDocumentAuditEvents(ctx, failureAudits)
			return 0, nil, deleteErr
		}

		successfullyRemovedSources = append(successfullyRemovedSources, document)
		audit := buildDocumentDeletionAuditEvent(ctx, &document, "SUCCESS", "Documento excluido em lote com remocao do arquivo original e limpeza dos dados derivados.", "collection-bulk", map[string]any{
			"collection_id":       collectionID,
			"file_url":            document.FileURL,
			"storage_delete_path": document.FileURL,
		})
		if deleteResult.detail != "" {
			audit.Metadata["storage_delete_detail"] = deleteResult.detail
		}
		successAudits = append(successAudits, audit)
	}

	deletedCount, err := u.documentRepo.DeleteDocumentsByCollection(ctx, userID, collectionID, successAudits)
	if err != nil {
		log.Printf("collection document deletion repository cleanup failed after source removal: user=%d collection=%d err=%v", userID, collectionID, err)
		failureAudits = make([]models.DocumentAuditEvent, 0, len(successfullyRemovedSources))
		for _, partiallyRemoved := range successfullyRemovedSources {
			_, _ = u.documentRepo.MarkDocumentOrphaned(ctx, partiallyRemoved.UserID, partiallyRemoved.ID, "O arquivo original foi removido, mas a exclusao em lote falhou antes da limpeza completa.")
			failureAudits = append(failureAudits, buildDocumentDeletionAuditEvent(ctx, &partiallyRemoved, "FAILED", fmt.Sprintf("O arquivo original foi removido, mas a exclusao em lote falhou antes da limpeza completa: %v", err), "collection-bulk", map[string]any{
				"collection_id": collectionID,
				"file_url":      partiallyRemoved.FileURL,
			}))
		}
		_ = u.documentRepo.CreateDocumentAuditEvents(ctx, failureAudits)
		return 0, nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	log.Printf("collection documents deleted successfully: user=%d collection=%d deleted=%d", userID, collectionID, deletedCount)
	return deletedCount, collection, nil
}

func (u *documentCatalogUseCase) GetDocumentContent(ctx context.Context, userID, documentID uint) (*domain.DocumentContent, error) {
	doc, err := u.GetDocument(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}

	reader, err := u.openDocumentReader(ctx, userID, doc)
	if err != nil {
		return nil, err
	}

	return &domain.DocumentContent{
		Document: doc,
		Reader:   reader,
	}, nil
}

func (u *documentCatalogUseCase) openDocumentReader(ctx context.Context, userID uint, doc *models.Document) (io.ReadCloser, error) {
	storageClient := u.storage
	usedCustomStorage := false
	userStorage, providerErr := u.storageResolver.ResolveStorage(ctx, userID)
	switch {
	case providerErr == nil:
		storageClient = userStorage
		usedCustomStorage = true
	case errors.Is(providerErr, domain.ErrStorageProviderNotConfigured):
	default:
		return nil, providerErr
	}

	reader, err := storageClient.OpenFile(ctx, doc.FileURL)
	if err == nil {
		return reader, nil
	}

	if usedCustomStorage {
		fallbackReader, fallbackErr := u.storage.OpenFile(ctx, doc.FileURL)
		if fallbackErr == nil {
			return fallbackReader, nil
		}
		if u.shouldMarkDocumentAsOrphaned(ctx, doc, storageClient, u.storage) {
			return nil, domain.ErrDocumentSourceMissing
		}
		return nil, fmt.Errorf("%w: custom storage open failed: %v; fallback open failed: %v", domain.ErrUploadFailed, err, fallbackErr)
	}

	if u.shouldMarkDocumentAsOrphaned(ctx, doc, storageClient, nil) {
		return nil, domain.ErrDocumentSourceMissing
	}

	return nil, fmt.Errorf("%w: %v", domain.ErrUploadFailed, err)
}

func (u *documentCatalogUseCase) shouldMarkDocumentAsOrphaned(ctx context.Context, doc *models.Document, primary domain.StorageService, secondary domain.StorageService) bool {
	existsInPrimary, primaryErr := primary.FileExists(ctx, doc.FileURL)
	if primaryErr == nil && existsInPrimary {
		return false
	}

	if secondary != nil {
		existsInSecondary, secondaryErr := secondary.FileExists(ctx, doc.FileURL)
		if secondaryErr == nil && existsInSecondary {
			return false
		}
		if secondaryErr != nil {
			return false
		}
	}

	if primaryErr != nil {
		return false
	}

	_, _ = u.documentRepo.MarkDocumentOrphaned(ctx, doc.UserID, doc.ID, "O arquivo original nao foi encontrado no storage configurado.")
	return true
}

func (u *documentCatalogUseCase) deleteDocumentSource(ctx context.Context, userID uint, fileURL string) (documentDeletionSourceResult, error) {
	storageClient := u.storage
	usedCustomStorage := false
	userStorage, providerErr := u.storageResolver.ResolveStorage(ctx, userID)
	switch {
	case providerErr == nil:
		storageClient = userStorage
		usedCustomStorage = true
	case errors.Is(providerErr, domain.ErrStorageProviderNotConfigured):
	default:
		return documentDeletionSourceResult{}, providerErr
	}

	primaryErr := storageClient.DeleteFile(ctx, fileURL)
	if primaryErr == nil {
		return documentDeletionSourceResult{
			removed: true,
			detail:  storageDeletionDetail(usedCustomStorage, false),
		}, nil
	}

	if usedCustomStorage {
		if err := u.storage.DeleteFile(ctx, fileURL); err == nil {
			return documentDeletionSourceResult{
				removed: true,
				detail:  storageDeletionDetail(true, true),
			}, nil
		}
		return documentDeletionSourceResult{}, fmt.Errorf("%w: custom storage delete failed: %v", domain.ErrDocumentSourceDeleteFailed, primaryErr)
	}

	return documentDeletionSourceResult{}, fmt.Errorf("%w: %v", domain.ErrDocumentSourceDeleteFailed, primaryErr)
}

func storageDeletionDetail(usedCustomStorage bool, usedFallback bool) string {
	switch {
	case usedCustomStorage && usedFallback:
		return "Arquivo removido pelo storage padrao apos falha no storage customizado."
	case usedCustomStorage:
		return "Arquivo removido pelo storage customizado ativo do usuario."
	default:
		return "Arquivo removido pelo storage padrao."
	}
}

func buildDocumentAuditEvent(ctx context.Context, doc *models.Document, action, status, detail string, metadata map[string]any) models.DocumentAuditEvent {
	var collectionID *uint
	if doc != nil {
		collectionID = doc.CollectionID
	}

	eventMetadata := cloneMetadata(metadata)
	if eventMetadata == nil {
		eventMetadata = map[string]any{}
	}

	var documentID *uint
	var title string
	var userID uint
	var actorID *uint
	actorType := "USER"
	var correlationID string
	var origin string
	if doc != nil {
		documentID = &doc.ID
		title = doc.Title
		userID = doc.UserID
		actorID = &doc.UserID
	}

	if extracted, ok := domain.ExtractAuditContext(ctx); ok {
		if extracted.ActorType != "" {
			actorType = extracted.ActorType
		}
		actorID = extracted.ActorID
		correlationID = extracted.CorrelationID
		origin = extracted.Origin
		if extracted.RequestMethod != "" {
			eventMetadata["request_method"] = extracted.RequestMethod
		}
		if extracted.RoutePath != "" {
			eventMetadata["route_path"] = extracted.RoutePath
		}
		if extracted.RemoteAddr != "" {
			eventMetadata["remote_addr"] = extracted.RemoteAddr
		}
	}

	return models.DocumentAuditEvent{
		UserID:        userID,
		DocumentID:    documentID,
		CollectionID:  collectionID,
		ActorType:     actorType,
		ActorID:       actorID,
		Action:        action,
		Status:        status,
		CorrelationID: correlationID,
		Origin:        origin,
		Title:         title,
		Detail:        detail,
		Metadata:      eventMetadata,
	}
}

func buildDocumentDeletionAuditEvent(ctx context.Context, doc *models.Document, status, detail, mode string, metadata map[string]any) models.DocumentAuditEvent {
	eventMetadata := cloneMetadata(metadata)
	if eventMetadata == nil {
		eventMetadata = map[string]any{}
	}
	if mode != "" {
		eventMetadata["mode"] = mode
	}

	return buildDocumentAuditEvent(ctx, doc, "DOCUMENT_DELETE", status, detail, eventMetadata)
}

func (u *documentCatalogUseCase) GetDocumentIndex(ctx context.Context, userID, documentID uint, query string) (*domain.DocumentIndex, error) {
	doc, err := u.GetDocument(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}

	documentIDs := []uint{documentID}
	documentPages, err := u.structuredDataRepo.GetDocumentPagesByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	paragraphNodes, err := u.structuredDataRepo.GetParagraphNodesByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	tableSections, err := u.structuredDataRepo.GetTableSectionsByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	metadataPairs, err := u.structuredDataRepo.GetMetadataPairsByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	chunks, err := u.structuredDataRepo.GetDocumentChunksByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	outlineNodes, err := u.structuredDataRepo.GetOutlineNodesByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	documentSummary := ""
	for _, pair := range metadataPairs {
		if pair.Key == "document_summary" {
			documentSummary = pair.Value
		}
	}

	index := &domain.DocumentIndex{
		Document:        doc,
		DocumentPages:   documentPages,
		ParagraphNodes:  paragraphNodes,
		TableSections:   tableSections,
		MetadataPairs:   metadataPairs,
		Chunks:          chunks,
		OutlineNodes:    outlineNodes,
		DocumentSummary: documentSummary,
	}

	if strings.TrimSpace(query) != "" {
		index.TreeRetrieval = buildTreeRetrieval(strings.TrimSpace(query), outlineNodes, documentPages, chunks)
	}
	index.IndexEvaluation = buildIndexEvaluation(metadataPairs, outlineNodes)
	index.Monitoring = buildIndexMonitoring(doc, metadataPairs, index.IndexEvaluation)

	return index, nil
}

func (u *documentCatalogUseCase) GetDocumentMonitoringSummary(ctx context.Context, userID uint) (*domain.DocumentMonitoringSummary, error) {
	documents, err := u.documentRepo.GetDocumentsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user documents: %w", err)
	}

	documentIDs := make([]uint, 0, len(documents))
	for _, doc := range documents {
		documentIDs = append(documentIDs, doc.ID)
	}

	metadataPairs, err := u.structuredDataRepo.GetMetadataPairsByDocumentIDs(ctx, documentIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	return aggregateDocumentMonitoringSummary(documents, metadataPairs), nil
}

func aggregateDocumentMonitoringSummary(
	documents []models.Document,
	metadataPairs []models.MetadataPair,
) *domain.DocumentMonitoringSummary {
	summary := &domain.DocumentMonitoringSummary{
		TotalDocuments: len(documents),
		ModelBreakdown: make([]domain.DocumentMonitoringModelAggregate, 0),
	}
	if len(documents) == 0 {
		return summary
	}

	metadataByDocument := make(map[uint][]models.MetadataPair, len(documents))
	for _, pair := range metadataPairs {
		metadataByDocument[pair.DocumentID] = append(metadataByDocument[pair.DocumentID], pair)
	}

	modelAggregateByKey := make(map[string]*domain.DocumentMonitoringModelAggregate)
	var durationSamples int64
	var qualitySamples int

	for i := range documents {
		doc := &documents[i]
		switch doc.Status {
		case domain.DocumentStatusIndexed:
			summary.IndexedDocuments++
		case domain.DocumentStatusProcessing:
			summary.ProcessingDocuments++
		case domain.DocumentStatusQueued:
			summary.QueuedDocuments++
		case domain.DocumentStatusFailed:
			summary.FailedDocuments++
		case domain.DocumentStatusOrphaned:
			summary.OrphanedDocuments++
		}
		if doc.Status == domain.DocumentStatusFailed && doc.ProcessingStage != "MOVED_TO_DLQ" {
			summary.PermanentFailedDocuments++
		}
		if doc.ProcessingStage == "RETRY_SCHEDULED" {
			summary.RetryingDocuments++
		}
		if doc.ProcessingStage == "MOVED_TO_DLQ" {
			summary.DLQDocuments++
		}

		evaluation := buildIndexEvaluation(metadataByDocument[doc.ID], nil)
		monitoring := buildIndexMonitoring(doc, metadataByDocument[doc.ID], evaluation)
		if monitoring == nil {
			continue
		}

		if monitoring.LLM != nil {
			summary.DocumentsWithLLM++
			summary.TotalLLMRequests += monitoring.LLM.RequestCount
			summary.TotalInputTokens += monitoring.LLM.EstimatedInputTokens
			summary.TotalOutputTokens += monitoring.LLM.EstimatedOutputTokens
			summary.TotalTokens += monitoring.LLM.EstimatedTotalTokens
			summary.TotalEstimatedCostUSD += monitoring.LLM.TotalCostUSD
			if monitoring.LLM.PricingAvailable {
				summary.PricingAwareDocuments++
			}

			key := monitoring.LLM.ProviderType + "::" + monitoring.LLM.Model
			aggregate := modelAggregateByKey[key]
			if aggregate == nil {
				aggregate = &domain.DocumentMonitoringModelAggregate{
					ProviderType: monitoring.LLM.ProviderType,
					Model:        monitoring.LLM.Model,
				}
				modelAggregateByKey[key] = aggregate
			}
			aggregate.DocumentCount++
			aggregate.RequestCount += monitoring.LLM.RequestCount
			aggregate.TotalTokens += monitoring.LLM.EstimatedTotalTokens
			aggregate.TotalCostUSD += monitoring.LLM.TotalCostUSD
		}

		if monitoring.DurationMs > 0 {
			summary.AverageDurationMs += monitoring.DurationMs
			durationSamples++
		}
		if monitoring.EstimatedQualityScore > 0 {
			summary.AverageQualityScore += monitoring.EstimatedQualityScore
			qualitySamples++
		}
		if monitoring.CompletedAt != nil && (summary.LastCompletedAt == nil || monitoring.CompletedAt.After(*summary.LastCompletedAt)) {
			completedAt := *monitoring.CompletedAt
			summary.LastCompletedAt = &completedAt
		}
	}

	if durationSamples > 0 {
		summary.AverageDurationMs /= durationSamples
	}
	if qualitySamples > 0 {
		summary.AverageQualityScore /= float64(qualitySamples)
	}

	for _, aggregate := range modelAggregateByKey {
		summary.ModelBreakdown = append(summary.ModelBreakdown, *aggregate)
	}

	sort.Slice(summary.ModelBreakdown, func(i, j int) bool {
		if summary.ModelBreakdown[i].TotalCostUSD == summary.ModelBreakdown[j].TotalCostUSD {
			if summary.ModelBreakdown[i].TotalTokens == summary.ModelBreakdown[j].TotalTokens {
				return summary.ModelBreakdown[i].Model < summary.ModelBreakdown[j].Model
			}
			return summary.ModelBreakdown[i].TotalTokens > summary.ModelBreakdown[j].TotalTokens
		}
		return summary.ModelBreakdown[i].TotalCostUSD > summary.ModelBreakdown[j].TotalCostUSD
	})

	return summary
}

func (u *documentCatalogUseCase) ListDocumentsByCategory(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error) {
	return u.documentRepo.GetDocumentsByCategoryID(ctx, categoryID, userID)
}

func (u *documentCatalogUseCase) ListCollections(ctx context.Context, userID uint) ([]models.Collection, error) {
	return u.collectionRepo.ListCollectionsByUserID(ctx, userID)
}

func (u *documentCatalogUseCase) GetCollectionContext(ctx context.Context, userID, collectionID uint, query string) (*domain.CollectionContext, error) {
	collectionContext, err := u.collectionContextRepo.GetCollectionContext(ctx, userID, collectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}

	if strings.TrimSpace(query) != "" {
		collectionContext.CollectionRetrieval = buildCollectionRetrieval(
			strings.TrimSpace(query),
			collectionContext.Documents,
			collectionContext.DocumentPages,
			collectionContext.OutlineNodes,
			collectionContext.Chunks,
			collectionContext.TableSections,
		)
	}

	return collectionContext, nil
}

func (u *documentCatalogUseCase) StoreParagraphNodes(ctx context.Context, userID, documentID uint, nodes []models.ParagraphNode) ([]models.ParagraphNode, error) {
	if _, err := u.GetDocument(ctx, userID, documentID); err != nil {
		return nil, err
	}

	for i := range nodes {
		nodes[i].DocumentID = documentID
		nodes[i].Metadata = cloneMetadata(nodes[i].Metadata)
	}

	if err := u.structuredDataRepo.ReplaceParagraphNodes(ctx, documentID, nodes); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	return nodes, nil
}

func (u *documentCatalogUseCase) StoreTableSections(ctx context.Context, userID, documentID uint, sections []models.TableSection) ([]models.TableSection, error) {
	if _, err := u.GetDocument(ctx, userID, documentID); err != nil {
		return nil, err
	}

	for i := range sections {
		sections[i].DocumentID = documentID
		sections[i].Headers = append([]string(nil), sections[i].Headers...)
		sections[i].Rows = append([][]string(nil), sections[i].Rows...)
		sections[i].Metadata = cloneMetadata(sections[i].Metadata)
	}

	if err := u.structuredDataRepo.ReplaceTableSections(ctx, documentID, sections); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	return sections, nil
}

func (u *documentCatalogUseCase) StoreMetadataPairs(ctx context.Context, userID, documentID uint, pairs []models.MetadataPair) ([]models.MetadataPair, error) {
	if _, err := u.GetDocument(ctx, userID, documentID); err != nil {
		return nil, err
	}

	for i := range pairs {
		pairs[i].DocumentID = documentID
	}

	if err := u.structuredDataRepo.ReplaceMetadataPairs(ctx, documentID, pairs); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrStructuredDataStore, err)
	}

	return pairs, nil
}

func buildTreeRetrieval(query string, nodes []models.DocumentOutlineNode, pages []models.DocumentPage, chunks []models.DocumentChunk) *domain.TreeRetrievalResult {
	queryTerms := tokenizeRetrievalText(query)
	if len(queryTerms) == 0 {
		return nil
	}

	directEvidence := buildTreeDirectEvidence(chunks, queryTerms)
	baseNodeHits, _ := traverseTree(queryTerms, nodes, nil)
	nodeHits, traversalSteps := traverseTree(queryTerms, nodes, directEvidence.NodeScoreByID)
	pageScoreByNumber := make(map[int]int, len(pages))
	for _, step := range traversalSteps {
		if step.StartPage == nil || step.EndPage == nil {
			continue
		}
		for pageNumber := *step.StartPage; pageNumber <= *step.EndPage; pageNumber++ {
			pageScoreByNumber[pageNumber] += step.Score
		}
	}

	bestDirectPageScore := 0
	pageHits := make([]domain.TreePageHit, 0, len(pages))
	for _, page := range pages {
		directScore := scoreTreePage(page, queryTerms) + directEvidence.PageScoreByNumber[page.PageNumber]
		if directScore > bestDirectPageScore {
			bestDirectPageScore = directScore
		}

		score := pageScoreByNumber[page.PageNumber] + directScore
		if score == 0 {
			continue
		}

		pageHits = append(pageHits, domain.TreePageHit{
			PageNumber: page.PageNumber,
			Summary:    page.Summary,
			Snippet:    clampRetrievalSnippet(page.Content, 220),
			Score:      score,
		})
	}

	sort.Slice(pageHits, func(i, j int) bool {
		if pageHits[i].Score == pageHits[j].Score {
			return pageHits[i].PageNumber < pageHits[j].PageNumber
		}
		return pageHits[i].Score > pageHits[j].Score
	})
	if len(pageHits) > 5 {
		pageHits = pageHits[:5]
	}

	strategy := "tree-iterative"
	if shouldPreferDirectTreeEvidence(baseNodeHits, nodeHits, nodes, directEvidence, bestDirectPageScore) {
		strategy = "tree-degraded-direct-evidence"
	} else if len(nodeHits) == 0 && bestDirectPageScore > 0 {
		strategy = "tree-direct-evidence"
	}

	return &domain.TreeRetrievalResult{
		Query:          query,
		NodeHits:       nodeHits,
		PageHits:       pageHits,
		TraversalSteps: traversalSteps,
		UsedTree:       true,
		Strategy:       strategy,
	}
}

type treeDirectEvidence struct {
	NodeScoreByID     map[string]int
	PageScoreByNumber map[int]int
	MaxNodeScore      int
	MaxPageScore      int
}

func buildTreeDirectEvidence(chunks []models.DocumentChunk, queryTerms []string) treeDirectEvidence {
	evidence := treeDirectEvidence{
		NodeScoreByID:     make(map[string]int),
		PageScoreByNumber: make(map[int]int),
	}

	for _, chunk := range chunks {
		score := scoreTreeChunk(chunk, queryTerms)
		if score == 0 {
			continue
		}

		if score > evidence.MaxPageScore {
			evidence.MaxPageScore = score
		}

		if nodeID := metadataStringValue(chunk.Metadata, "outline_node_id"); nodeID != "" {
			evidence.NodeScoreByID[nodeID] += score
			if evidence.NodeScoreByID[nodeID] > evidence.MaxNodeScore {
				evidence.MaxNodeScore = evidence.NodeScoreByID[nodeID]
			}
		}

		startPage, endPage, ok := chunkPageRange(chunk)
		if !ok {
			continue
		}
		for pageNumber := startPage; pageNumber <= endPage; pageNumber++ {
			evidence.PageScoreByNumber[pageNumber] += score
			if evidence.PageScoreByNumber[pageNumber] > evidence.MaxPageScore {
				evidence.MaxPageScore = evidence.PageScoreByNumber[pageNumber]
			}
		}
	}

	return evidence
}

func traverseTree(queryTerms []string, nodes []models.DocumentOutlineNode, directNodeScores map[string]int) ([]domain.TreeNodeHit, []domain.TreeTraversalStep) {
	nodeByID := make(map[string]models.DocumentOutlineNode, len(nodes))
	parentByNodeID := make(map[string]string, len(nodes))
	allHits := rankNodesForTraversal(nodes, queryTerms, directNodeScores)
	if len(allHits) == 0 {
		return nil, nil
	}

	scoreByNodeID := make(map[string]domain.TreeNodeHit, len(allHits))
	for _, node := range nodes {
		nodeByID[node.NodeID] = node
		if parentNodeID, ok := node.Metadata["parent_node_id"].(string); ok && strings.TrimSpace(parentNodeID) != "" {
			parentByNodeID[node.NodeID] = parentNodeID
		}
	}
	for _, hit := range allHits {
		scoreByNodeID[hit.NodeID] = hit
	}

	steps := buildTraversalSteps(allHits[0], nodeByID, parentByNodeID, scoreByNodeID)
	if len(allHits) > 5 {
		allHits = allHits[:5]
	}

	return allHits, steps
}

func rankNodesForTraversal(nodes []models.DocumentOutlineNode, queryTerms []string, directNodeScores map[string]int) []domain.TreeNodeHit {
	hits := make([]domain.TreeNodeHit, 0, len(nodes))
	for _, node := range nodes {
		score := retrievalScore(outlineNodeSearchText(node), queryTerms)
		score += metadataIntValue(node.Metadata, "confidence_score") / 25
		directScore := directNodeScores[node.NodeID]
		if directScore > 0 {
			score += directScore
			score += 2
		}
		if isWeakTreeNode(node, queryTerms, score) && directScore == 0 {
			score = max(0, score-2)
		}
		if score == 0 {
			continue
		}
		hits = append(hits, domain.TreeNodeHit{
			NodeID:        node.NodeID,
			Title:         node.Title,
			Summary:       node.Summary,
			Level:         node.Level,
			StartPage:     node.StartPage,
			EndPage:       node.EndPage,
			Score:         score,
			SequenceOrder: node.SequenceOrder,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			if hits[i].Level == hits[j].Level {
				return hits[i].SequenceOrder < hits[j].SequenceOrder
			}
			return hits[i].Level > hits[j].Level
		}
		return hits[i].Score > hits[j].Score
	})

	return hits
}

func shouldPreferDirectTreeEvidence(
	baseNodeHits []domain.TreeNodeHit,
	nodeHits []domain.TreeNodeHit,
	nodes []models.DocumentOutlineNode,
	directEvidence treeDirectEvidence,
	bestDirectPageScore int,
) bool {
	if len(nodeHits) == 0 {
		return false
	}

	nodeByID := make(map[string]models.DocumentOutlineNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.NodeID] = node
	}

	if len(baseNodeHits) == 0 {
		return directEvidence.MaxNodeScore > 0 || bestDirectPageScore > 0
	}

	baseTop := baseNodeHits[0]
	baseTopNode := nodeByID[baseTop.NodeID]
	if !isWeakTreeNode(baseTopNode, nil, baseTop.Score) {
		return false
	}

	if directEvidence.MaxNodeScore == 0 && bestDirectPageScore == 0 {
		return false
	}

	if nodeHits[0].NodeID != baseTop.NodeID {
		return true
	}

	return bestDirectPageScore >= baseTop.Score || directEvidence.MaxNodeScore >= baseTop.Score
}

func isWeakTreeNode(node models.DocumentOutlineNode, queryTerms []string, score int) bool {
	if fallbackStructure, ok := node.Metadata["fallback_structure"].(bool); ok && fallbackStructure {
		return true
	}
	if metadataStringValue(node.Metadata, "segment_type") == "page" {
		return true
	}
	if confidenceBand := metadataStringValue(node.Metadata, "confidence_band"); confidenceBand == "low" {
		return true
	}
	confidenceScore := metadataIntValue(node.Metadata, "confidence_score")
	if confidenceScore > 0 && confidenceScore < 60 {
		return true
	}

	minScore := 2
	if queryTermCount := len(queryTerms); queryTermCount >= 4 {
		minScore = 3
	} else if queryTermCount == 0 {
		minScore = 1
	}

	return score < minScore
}

func buildTraversalSteps(
	best domain.TreeNodeHit,
	nodeByID map[string]models.DocumentOutlineNode,
	parentByNodeID map[string]string,
	scoreByNodeID map[string]domain.TreeNodeHit,
) []domain.TreeTraversalStep {
	if strings.TrimSpace(best.NodeID) == "" {
		return nil
	}

	path := make([]string, 0, 4)
	seen := make(map[string]struct{}, len(parentByNodeID)+1)
	currentNodeID := best.NodeID
	for strings.TrimSpace(currentNodeID) != "" {
		if _, ok := seen[currentNodeID]; ok {
			break
		}
		seen[currentNodeID] = struct{}{}
		path = append(path, currentNodeID)
		currentNodeID = strings.TrimSpace(parentByNodeID[currentNodeID])
	}

	steps := make([]domain.TreeTraversalStep, 0, len(path))
	for idx := len(path) - 1; idx >= 0; idx-- {
		nodeID := path[idx]
		hit, ok := scoreByNodeID[nodeID]
		if !ok {
			node := nodeByID[nodeID]
			hit = domain.TreeNodeHit{
				NodeID:    node.NodeID,
				Title:     node.Title,
				Level:     node.Level,
				StartPage: node.StartPage,
				EndPage:   node.EndPage,
			}
		}
		steps = append(steps, domain.TreeTraversalStep{
			NodeID:    hit.NodeID,
			Title:     hit.Title,
			Level:     hit.Level,
			Score:     hit.Score,
			StartPage: hit.StartPage,
			EndPage:   hit.EndPage,
		})
	}

	return steps
}

func buildIndexEvaluation(metadataPairs []models.MetadataPair, nodes []models.DocumentOutlineNode) *domain.IndexEvaluationResult {
	if len(metadataPairs) == 0 && len(nodes) == 0 {
		return nil
	}

	pairMap := make(map[string]string, len(metadataPairs))
	for _, pair := range metadataPairs {
		pairMap[pair.Key] = pair.Value
	}

	coverageRatio := parseFloatMetric(pairMap["tree_coverage_ratio"])
	if coverageRatio == 0 {
		coverageRatio = estimateCoverageRatioFromNodes(nodes)
	}

	summaryCoverageRatio := parseFloatMetric(pairMap["tree_summary_coverage_ratio"])
	if summaryCoverageRatio == 0 {
		summaryCoverageRatio = estimateSummaryCoverageFromNodes(nodes)
	}

	maxDepth := parseIntMetric(pairMap["tree_max_depth"], 0)
	if maxDepth == 0 {
		maxDepth = estimateTreeDepth(nodes)
	}

	evaluation := &domain.IndexEvaluationResult{
		CoverageRatio:        coverageRatio,
		SummaryCoverageRatio: summaryCoverageRatio,
		TotalNodes:           parseIntMetric(pairMap["tree_total_nodes"], len(nodes)),
		LowConfidenceNodes:   parseIntMetric(pairMap["tree_low_confidence_nodes"], 0),
		OversizedNodes:       parseIntMetric(pairMap["tree_oversized_nodes"], 0),
		MaxDepth:             maxDepth,
	}
	evaluation.RetrievalReady = evaluation.CoverageRatio >= 0.6 && evaluation.TotalNodes > 0
	return evaluation
}

const (
	indexMonitoringTreeStateUnknown  = "unknown"
	indexMonitoringTreeStateHealthy  = "healthy"
	indexMonitoringTreeStateFallback = "fallback"
	indexMonitoringTreeStateDegraded = "degraded"

	indexMonitoringConcernFallbackBuilder   = "fallback_builder"
	indexMonitoringConcernVerification      = "verification_failed"
	indexMonitoringConcernLowCoverage       = "low_coverage"
	indexMonitoringConcernLowSummaries      = "low_summary_coverage"
	indexMonitoringConcernLowConfidence     = "low_confidence_nodes"
	indexMonitoringConcernShallowTree       = "shallow_tree"
	indexMonitoringConcernOversizedNodes    = "oversized_nodes"
	indexMonitoringConcernRetrievalNotReady = "retrieval_not_ready"

	monitoringFallbackTreeBuilder = "fallback-page-segments"
)

func buildIndexMonitoring(
	doc *models.Document,
	metadataPairs []models.MetadataPair,
	evaluation *domain.IndexEvaluationResult,
) *domain.IndexMonitoringResult {
	if doc == nil && len(metadataPairs) == 0 {
		return nil
	}

	pairMap := make(map[string]string, len(metadataPairs))
	for _, pair := range metadataPairs {
		pairMap[pair.Key] = pair.Value
	}

	monitoring := &domain.IndexMonitoringResult{
		EnrichmentStatus:       strings.TrimSpace(pairMap["enrichment_status"]),
		PipelineVersion:        strings.TrimSpace(pairMap["index_pipeline_version"]),
		PromptProfileVersion:   strings.TrimSpace(pairMap["prompt_profile_version"]),
		ChunkStrategyVersion:   strings.TrimSpace(pairMap["chunk_strategy_version"]),
		TreeBuilder:            strings.TrimSpace(pairMap["tree_builder"]),
		TreeVerificationPassed: strings.EqualFold(strings.TrimSpace(pairMap["tree_verification_passed"]), "true"),
	}

	if doc != nil {
		monitoring.Status = doc.Status
		monitoring.Stage = doc.ProcessingStage
		monitoring.StatusDetail = doc.StatusDetail
		monitoring.StartedAt = doc.IndexStartedAt
		monitoring.CompletedAt = doc.IndexedAt
		if doc.IndexStartedAt != nil && doc.IndexedAt != nil {
			monitoring.DurationMs = doc.IndexedAt.Sub(*doc.IndexStartedAt).Milliseconds()
		}
	}

	if monitoring.StartedAt == nil {
		if startedAt := parseTimeMetric(pairMap["index_started_at"]); startedAt != nil {
			monitoring.StartedAt = startedAt
		}
	}
	if monitoring.CompletedAt == nil {
		if completedAt := parseTimeMetric(pairMap["index_completed_at"]); completedAt != nil {
			monitoring.CompletedAt = completedAt
		}
	}
	if monitoring.DurationMs == 0 {
		monitoring.DurationMs = parseInt64Metric(pairMap["index_duration_ms"], 0)
	}

	llmMonitoring := &domain.LLMMonitoringResult{
		ProviderType:          strings.TrimSpace(pairMap["llm_provider_type"]),
		Model:                 strings.TrimSpace(pairMap["llm_model"]),
		RequestCount:          parseIntMetric(pairMap["llm_request_count"], 0),
		EstimatedInputTokens:  parseIntMetric(pairMap["llm_input_tokens_estimate"], 0),
		EstimatedOutputTokens: parseIntMetric(pairMap["llm_output_tokens_estimate"], 0),
		EstimatedTotalTokens:  parseIntMetric(pairMap["llm_total_tokens_estimate"], 0),
		PricingAvailable:      strings.EqualFold(strings.TrimSpace(pairMap["llm_pricing_available"]), "true"),
		InputCostUSD:          parseFloatMetric(pairMap["llm_input_cost_usd"]),
		OutputCostUSD:         parseFloatMetric(pairMap["llm_output_cost_usd"]),
		TotalCostUSD:          parseFloatMetric(pairMap["llm_total_cost_usd"]),
	}
	if llmMonitoring.ProviderType != "" || llmMonitoring.Model != "" || llmMonitoring.RequestCount > 0 {
		monitoring.LLM = llmMonitoring
	}

	monitoring.TreeState, monitoring.TreeStateDetail, monitoring.TreeConcernCodes = classifyIndexMonitoringTree(
		monitoring.TreeBuilder,
		monitoring.TreeVerificationPassed,
		evaluation,
	)
	monitoring.EstimatedQualityScore = adjustIndexMonitoringQualityScore(
		estimateIndexQualityScore(evaluation),
		monitoring.TreeState,
		monitoring.TreeConcernCodes,
	)
	if monitoring.Status == "" &&
		monitoring.Stage == "" &&
		monitoring.EnrichmentStatus == "" &&
		monitoring.PipelineVersion == "" &&
		monitoring.LLM == nil &&
		monitoring.EstimatedQualityScore == 0 {
		return nil
	}

	return monitoring
}

func classifyIndexMonitoringTree(
	treeBuilder string,
	treeVerificationPassed bool,
	evaluation *domain.IndexEvaluationResult,
) (string, string, []string) {
	treeBuilder = strings.TrimSpace(treeBuilder)
	if treeBuilder == "" && evaluation == nil {
		return indexMonitoringTreeStateUnknown, "No structural telemetry is available yet.", nil
	}

	concerns := buildIndexMonitoringTreeConcernCodes(treeBuilder, treeVerificationPassed, evaluation)
	switch {
	case isFallbackMonitoringTreeBuilder(treeBuilder):
		return indexMonitoringTreeStateFallback, buildIndexMonitoringTreeDetail(indexMonitoringTreeStateFallback, treeBuilder, evaluation, concerns), concerns
	case len(concerns) == 0:
		return indexMonitoringTreeStateHealthy, buildIndexMonitoringTreeDetail(indexMonitoringTreeStateHealthy, treeBuilder, evaluation, concerns), nil
	default:
		return indexMonitoringTreeStateDegraded, buildIndexMonitoringTreeDetail(indexMonitoringTreeStateDegraded, treeBuilder, evaluation, concerns), concerns
	}
}

func buildIndexMonitoringTreeConcernCodes(
	treeBuilder string,
	treeVerificationPassed bool,
	evaluation *domain.IndexEvaluationResult,
) []string {
	concerns := make([]string, 0, 6)
	if isFallbackMonitoringTreeBuilder(treeBuilder) {
		concerns = append(concerns, indexMonitoringConcernFallbackBuilder)
	}
	if treeBuilder != "" && !treeVerificationPassed {
		concerns = append(concerns, indexMonitoringConcernVerification)
	}
	if evaluation == nil {
		return concerns
	}
	if !evaluation.RetrievalReady {
		concerns = append(concerns, indexMonitoringConcernRetrievalNotReady)
	}
	if evaluation.CoverageRatio < 0.75 {
		concerns = append(concerns, indexMonitoringConcernLowCoverage)
	}
	if evaluation.SummaryCoverageRatio < 0.5 {
		concerns = append(concerns, indexMonitoringConcernLowSummaries)
	}
	if evaluation.TotalNodes <= 1 || evaluation.MaxDepth <= 1 {
		concerns = append(concerns, indexMonitoringConcernShallowTree)
	}
	if evaluation.OversizedNodes > 0 {
		concerns = append(concerns, indexMonitoringConcernOversizedNodes)
	}

	lowConfidenceThreshold := max(2, evaluation.TotalNodes/3)
	if evaluation.LowConfidenceNodes >= lowConfidenceThreshold && evaluation.LowConfidenceNodes > 0 {
		concerns = append(concerns, indexMonitoringConcernLowConfidence)
	}

	return concerns
}

func buildIndexMonitoringTreeDetail(
	treeState string,
	treeBuilder string,
	evaluation *domain.IndexEvaluationResult,
	concerns []string,
) string {
	metrics := describeIndexMonitoringTreeMetrics(evaluation)
	switch treeState {
	case indexMonitoringTreeStateHealthy:
		if treeBuilder != "" && metrics != "" {
			return fmt.Sprintf("Verified tree from builder %s with %s.", treeBuilder, metrics)
		}
		if metrics != "" {
			return fmt.Sprintf("Verified tree with %s.", metrics)
		}
		return "Verified tree is available."
	case indexMonitoringTreeStateFallback:
		if metrics != "" {
			return fmt.Sprintf("Fallback builder %s is active, so hierarchy was replaced by page segments; %s.", treeBuilder, metrics)
		}
		return fmt.Sprintf("Fallback builder %s is active, so hierarchy was replaced by page segments.", treeBuilder)
	case indexMonitoringTreeStateDegraded:
		fragments := describeIndexMonitoringTreeConcerns(concerns, treeBuilder, evaluation)
		switch {
		case len(fragments) > 0 && metrics != "":
			return fmt.Sprintf("Tree is degraded because %s; %s.", strings.Join(fragments, "; "), metrics)
		case len(fragments) > 0:
			return fmt.Sprintf("Tree is degraded because %s.", strings.Join(fragments, "; "))
		case metrics != "":
			return fmt.Sprintf("Tree is degraded; %s.", metrics)
		default:
			return "Tree is degraded."
		}
	default:
		if metrics != "" {
			return fmt.Sprintf("Structural telemetry is partial; %s.", metrics)
		}
		return "No structural telemetry is available yet."
	}
}

func describeIndexMonitoringTreeMetrics(evaluation *domain.IndexEvaluationResult) string {
	if evaluation == nil {
		return ""
	}

	parts := []string{
		fmt.Sprintf("coverage %.0f%%", evaluation.CoverageRatio*100),
		fmt.Sprintf("summary coverage %.0f%%", evaluation.SummaryCoverageRatio*100),
		fmt.Sprintf("%d nodes", evaluation.TotalNodes),
	}
	if evaluation.MaxDepth > 0 {
		parts = append(parts, fmt.Sprintf("depth %d", evaluation.MaxDepth))
	}
	return strings.Join(parts, ", ")
}

func describeIndexMonitoringTreeConcerns(
	concerns []string,
	treeBuilder string,
	evaluation *domain.IndexEvaluationResult,
) []string {
	if len(concerns) == 0 {
		return nil
	}

	fragments := make([]string, 0, len(concerns))
	for _, concern := range concerns {
		switch concern {
		case indexMonitoringConcernFallbackBuilder:
			if treeBuilder != "" {
				fragments = append(fragments, fmt.Sprintf("builder=%s", treeBuilder))
			}
		case indexMonitoringConcernVerification:
			fragments = append(fragments, "verification=false")
		case indexMonitoringConcernRetrievalNotReady:
			fragments = append(fragments, "retrieval_ready=false")
		case indexMonitoringConcernLowCoverage:
			if evaluation != nil {
				fragments = append(fragments, fmt.Sprintf("coverage=%.0f%%", evaluation.CoverageRatio*100))
			}
		case indexMonitoringConcernLowSummaries:
			if evaluation != nil {
				fragments = append(fragments, fmt.Sprintf("summary_coverage=%.0f%%", evaluation.SummaryCoverageRatio*100))
			}
		case indexMonitoringConcernLowConfidence:
			if evaluation != nil {
				fragments = append(fragments, fmt.Sprintf("low_confidence_nodes=%d/%d", evaluation.LowConfidenceNodes, evaluation.TotalNodes))
			}
		case indexMonitoringConcernShallowTree:
			if evaluation != nil {
				fragments = append(fragments, fmt.Sprintf("nodes=%d depth=%d", evaluation.TotalNodes, evaluation.MaxDepth))
			}
		case indexMonitoringConcernOversizedNodes:
			if evaluation != nil {
				fragments = append(fragments, fmt.Sprintf("oversized_nodes=%d", evaluation.OversizedNodes))
			}
		}
	}

	return fragments
}

func adjustIndexMonitoringQualityScore(score float64, treeState string, concerns []string) float64 {
	if score <= 0 {
		return 0
	}

	switch treeState {
	case indexMonitoringTreeStateFallback:
		score -= 12
		if score > 55 {
			score = 55
		}
	case indexMonitoringTreeStateDegraded:
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernVerification) {
			score -= 8
		}
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernRetrievalNotReady) {
			score -= 6
		}
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernLowCoverage) {
			score -= 6
		}
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernLowSummaries) {
			score -= 6
		}
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernLowConfidence) {
			score -= 4
		}
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernShallowTree) {
			score -= 4
		}
		if containsIndexMonitoringConcern(concerns, indexMonitoringConcernOversizedNodes) {
			score -= 4
		}
		if score > 72 {
			score = 72
		}
	case indexMonitoringTreeStateUnknown:
		if score > 65 {
			score = 65
		}
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func containsIndexMonitoringConcern(concerns []string, target string) bool {
	for _, concern := range concerns {
		if concern == target {
			return true
		}
	}
	return false
}

func isFallbackMonitoringTreeBuilder(treeBuilder string) bool {
	return strings.EqualFold(strings.TrimSpace(treeBuilder), monitoringFallbackTreeBuilder)
}

func estimateIndexQualityScore(evaluation *domain.IndexEvaluationResult) float64 {
	if evaluation == nil {
		return 0
	}

	score := evaluation.CoverageRatio * 35
	score += min(1, float64(evaluation.TotalNodes)/6) * 15
	score += evaluation.SummaryCoverageRatio * 20
	score += estimateDepthQualityBonus(evaluation.MaxDepth)
	if evaluation.RetrievalReady {
		score += 8
	}
	score -= float64(evaluation.LowConfidenceNodes) * 4
	score -= float64(evaluation.OversizedNodes) * 6
	if evaluation.TotalNodes <= 1 {
		score -= 10
	}
	if evaluation.SummaryCoverageRatio < 0.4 {
		score -= 8
	}

	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func buildCollectionRetrieval(
	query string,
	documents []models.Document,
	pages []models.DocumentPage,
	nodes []models.DocumentOutlineNode,
	chunks []models.DocumentChunk,
	tables []models.TableSection,
) *domain.CollectionRetrievalResult {
	queryTerms := tokenizeRetrievalText(query)
	if len(queryTerms) == 0 {
		return nil
	}

	documentScores := make(map[uint]int, len(documents))
	documentTitles := make(map[uint]string, len(documents))
	for _, doc := range documents {
		documentTitles[doc.ID] = doc.Title
		if score := retrievalScore(doc.Title, queryTerms); score > 0 {
			documentScores[doc.ID] += score
		}
	}

	nodeHits := make([]domain.CollectionNodeHit, 0)
	for _, node := range nodes {
		score := scoreCollectionNode(node, queryTerms)
		if score == 0 {
			continue
		}

		documentScores[node.DocumentID] += score
		nodeHits = append(nodeHits, domain.CollectionNodeHit{
			DocumentID: node.DocumentID,
			NodeID:     node.NodeID,
			Title:      node.Title,
			Summary:    node.Summary,
			Score:      score,
			StartPage:  node.StartPage,
			EndPage:    node.EndPage,
		})
	}

	sort.Slice(nodeHits, func(i, j int) bool {
		if nodeHits[i].Score == nodeHits[j].Score {
			if nodeHits[i].DocumentID == nodeHits[j].DocumentID {
				return nodeHits[i].NodeID < nodeHits[j].NodeID
			}
			return nodeHits[i].DocumentID < nodeHits[j].DocumentID
		}
		return nodeHits[i].Score > nodeHits[j].Score
	})
	if len(nodeHits) > 8 {
		nodeHits = nodeHits[:8]
	}

	pageHits := make([]domain.CollectionPageHit, 0)
	for _, page := range pages {
		score := scoreCollectionPage(page, queryTerms)
		if score == 0 {
			continue
		}
		documentScores[page.DocumentID] += score
		pageHits = append(pageHits, domain.CollectionPageHit{
			DocumentID: page.DocumentID,
			PageNumber: page.PageNumber,
			Snippet:    clampRetrievalSnippet(page.Content, 220),
			Score:      score,
		})
	}
	sort.Slice(pageHits, func(i, j int) bool {
		if pageHits[i].Score == pageHits[j].Score {
			if pageHits[i].DocumentID == pageHits[j].DocumentID {
				return pageHits[i].PageNumber < pageHits[j].PageNumber
			}
			return pageHits[i].DocumentID < pageHits[j].DocumentID
		}
		return pageHits[i].Score > pageHits[j].Score
	})
	if len(pageHits) > 8 {
		pageHits = pageHits[:8]
	}

	for _, chunk := range chunks {
		score := scoreCollectionChunk(chunk, queryTerms)
		if score == 0 {
			continue
		}
		documentScores[chunk.DocumentID] += score
	}

	for _, table := range tables {
		score := scoreCollectionTable(table, queryTerms)
		if score == 0 {
			continue
		}
		documentScores[table.DocumentID] += score
	}

	documentHits := make([]domain.CollectionDocumentHit, 0, len(documentScores))
	for documentID, score := range documentScores {
		documentHits = append(documentHits, domain.CollectionDocumentHit{
			DocumentID: documentID,
			Title:      documentTitles[documentID],
			Score:      score,
		})
	}
	sort.Slice(documentHits, func(i, j int) bool {
		if documentHits[i].Score == documentHits[j].Score {
			return documentHits[i].DocumentID < documentHits[j].DocumentID
		}
		return documentHits[i].Score > documentHits[j].Score
	})
	if len(documentHits) > 5 {
		documentHits = documentHits[:5]
	}

	return &domain.CollectionRetrievalResult{
		Query:        query,
		DocumentHits: documentHits,
		NodeHits:     nodeHits,
		PageHits:     pageHits,
		Strategy:     "collection-tree",
	}
}

func parseIntMetric(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseFloatMetric(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseInt64Metric(value string, fallback int64) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseTimeMetric(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	return &parsed
}

func estimateCoverageRatioFromNodes(nodes []models.DocumentOutlineNode) float64 {
	if len(nodes) == 0 {
		return 0
	}

	coveredPages := make(map[int]struct{}, len(nodes))
	minPage := 0
	maxPage := 0

	for _, node := range nodes {
		if node.StartPage == nil || node.EndPage == nil {
			continue
		}

		start := *node.StartPage
		end := *node.EndPage
		if start <= 0 || end <= 0 {
			continue
		}
		if end < start {
			start, end = end, start
		}
		if minPage == 0 || start < minPage {
			minPage = start
		}
		if end > maxPage {
			maxPage = end
		}
		for page := start; page <= end; page++ {
			coveredPages[page] = struct{}{}
		}
	}

	if minPage == 0 || maxPage == 0 {
		return 1
	}

	totalPages := maxPage - minPage + 1
	if totalPages <= 0 {
		return 0
	}

	return float64(len(coveredPages)) / float64(totalPages)
}

func estimateTreeDepth(nodes []models.DocumentOutlineNode) int {
	maxDepth := 0
	for _, node := range nodes {
		if node.Level > maxDepth {
			maxDepth = node.Level
		}
	}
	return maxDepth
}

func metadataIntValue(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	value, ok := metadata[key]
	if !ok {
		return 0
	}

	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func metadataStringValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(text)
}

func outlineNodeSearchText(node models.DocumentOutlineNode) string {
	retrievalText := metadataStringValue(node.Metadata, "retrieval_text")
	if retrievalText == "" {
		retrievalText = node.Summary
	}
	return strings.TrimSpace(strings.Join([]string{node.Title, retrievalText}, " "))
}

func tokenizeRetrievalText(value string) []string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return nil
	}

	normalized = strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "\n", " ", "\t", " ").Replace(normalized)
	parts := strings.Fields(normalized)
	seen := make(map[string]struct{}, len(parts))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, expanded := range expandNumericRetrievalTerms(part) {
			if len(expanded) < 1 {
				continue
			}
			if len(expanded) < 2 && !isNumericToken(expanded) {
				continue
			}
			if _, ok := seen[expanded]; ok {
				continue
			}
			seen[expanded] = struct{}{}
			tokens = append(tokens, expanded)
		}
	}
	return tokens
}

func expandNumericRetrievalTerms(term string) []string {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil
	}

	expansions := []string{term}
	switch term {
	case "zero":
		expansions = append(expansions, "0")
	case "one":
		expansions = append(expansions, "1")
	case "two":
		expansions = append(expansions, "2")
	case "three":
		expansions = append(expansions, "3")
	case "four":
		expansions = append(expansions, "4")
	case "five":
		expansions = append(expansions, "5")
	case "six":
		expansions = append(expansions, "6")
	case "seven":
		expansions = append(expansions, "7")
	case "eight":
		expansions = append(expansions, "8")
	case "nine":
		expansions = append(expansions, "9")
	case "ten":
		expansions = append(expansions, "10")
	case "0":
		expansions = append(expansions, "zero")
	case "1":
		expansions = append(expansions, "one")
	case "2":
		expansions = append(expansions, "two")
	case "3":
		expansions = append(expansions, "three")
	case "4":
		expansions = append(expansions, "four")
	case "5":
		expansions = append(expansions, "five")
	case "6":
		expansions = append(expansions, "six")
	case "7":
		expansions = append(expansions, "seven")
	case "8":
		expansions = append(expansions, "eight")
	case "9":
		expansions = append(expansions, "nine")
	case "10":
		expansions = append(expansions, "ten")
	}

	return expansions
}

func isNumericToken(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func retrievalScore(text string, queryTerms []string) int {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" || len(queryTerms) == 0 {
		return 0
	}

	score := 0
	for _, term := range queryTerms {
		if strings.Contains(normalized, term) {
			score++
		}
	}
	return score
}

func estimateSummaryCoverageFromNodes(nodes []models.DocumentOutlineNode) float64 {
	if len(nodes) == 0 {
		return 0
	}

	summarized := 0
	for _, node := range nodes {
		if strings.TrimSpace(node.Summary) != "" {
			summarized++
		}
	}

	return float64(summarized) / float64(len(nodes))
}

func estimateDepthQualityBonus(maxDepth int) float64 {
	switch {
	case maxDepth >= 4:
		return 10
	case maxDepth == 3:
		return 8
	case maxDepth == 2:
		return 6
	case maxDepth == 1:
		return 3
	default:
		return 0
	}
}

func scoreCollectionNode(node models.DocumentOutlineNode, queryTerms []string) int {
	score := retrievalScore(outlineNodeSearchText(node), queryTerms) * 3
	score += metadataIntValue(node.Metadata, "confidence_score") / 25
	if score > 0 && node.Level > 1 {
		score++
	}
	return score
}

func scoreCollectionPage(page models.DocumentPage, queryTerms []string) int {
	return retrievalScore(strings.Join([]string{page.Summary, page.Content}, " "), queryTerms) * 2
}

func scoreTreePage(page models.DocumentPage, queryTerms []string) int {
	return scoreCollectionPage(page, queryTerms)
}

func scoreCollectionChunk(chunk models.DocumentChunk, queryTerms []string) int {
	if chunk.ChunkType == "table_section" {
		return 0
	}
	return retrievalScore(strings.Join([]string{chunk.Title, chunk.Content}, " "), queryTerms) * 2
}

func scoreTreeChunk(chunk models.DocumentChunk, queryTerms []string) int {
	score := scoreCollectionChunk(chunk, queryTerms)
	if score == 0 {
		return 0
	}
	if metadataIntValue(chunk.Metadata, "related_table_count") > 0 {
		score++
	}
	return score
}

func chunkPageRange(chunk models.DocumentChunk) (int, int, bool) {
	startPage := metadataIntValue(chunk.Metadata, "page_start")
	endPage := metadataIntValue(chunk.Metadata, "page_end")
	if startPage <= 0 || endPage < startPage {
		return 0, 0, false
	}
	return startPage, endPage, true
}

func scoreCollectionTable(table models.TableSection, queryTerms []string) int {
	content := strings.TrimSpace(strings.Join([]string{
		table.Title,
		strings.Join(table.Headers, " "),
		renderTableRowsForRetrieval(table.Rows, 3),
	}, " "))
	return retrievalScore(content, queryTerms) * 5
}

func renderTableRowsForRetrieval(rows [][]string, limit int) string {
	if len(rows) == 0 || limit <= 0 {
		return ""
	}

	parts := make([]string, 0, min(limit, len(rows)))
	for idx := 0; idx < len(rows) && idx < limit; idx++ {
		parts = append(parts, strings.Join(rows[idx], " "))
	}

	return strings.Join(parts, " ")
}

func clampRetrievalSnippet(value string, maxLen int) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len([]rune(cleaned)) <= maxLen {
		return cleaned
	}
	return strings.TrimSpace(string([]rune(cleaned)[:maxLen]))
}

func extractExtension(contentType string) string {
	switch contentType {
	case "application/pdf":
		return ".pdf"
	case "text/html":
		return ".html"
	case "text/plain":
		return ".txt"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	default:
		return ".bin"
	}
}

func validateUploadStream(file io.Reader, declaredContentType, title string) (io.Reader, string, error) {
	head := make([]byte, 512)
	readBytes, readErr := io.ReadFull(file, head)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return nil, "", fmt.Errorf("%w: %v", domain.ErrUploadFailed, readErr)
	}
	if readBytes == 0 {
		return nil, "", domain.ErrFileRequired
	}

	sniffedContentType := http.DetectContentType(head[:readBytes])
	normalizedDeclared := normalizeMediaType(declaredContentType)
	resolvedContentType := resolveContentType(normalizedDeclared, sniffedContentType, title)
	if _, ok := allowedUploadMimeTypes[resolvedContentType]; !ok {
		return nil, "", domain.ErrDocumentInvalidMime
	}

	return io.MultiReader(bytes.NewReader(head[:readBytes]), file), resolvedContentType, nil
}

func normalizeMediaType(value string) string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(trimmedValue)
	if err != nil {
		return trimmedValue
	}

	return mediaType
}

func resolveContentType(declaredContentType, sniffedContentType, title string) string {
	fileExtension := strings.ToLower(filepath.Ext(title))

	switch declaredContentType {
	case "application/pdf":
		if sniffedContentType == declaredContentType {
			return declaredContentType
		}
		return sniffedContentType
	case "text/html":
		if (strings.HasPrefix(sniffedContentType, "text/html") || strings.HasPrefix(sniffedContentType, "text/plain")) &&
			(fileExtension == ".html" || fileExtension == ".htm") {
			return declaredContentType
		}
		return sniffedContentType
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		if sniffedContentType == "application/zip" && fileExtension == ".xlsx" {
			return declaredContentType
		}
		return sniffedContentType
	}

	switch {
	case strings.HasPrefix(sniffedContentType, "text/html"):
		return "text/html"
	case sniffedContentType == "application/pdf":
		return sniffedContentType
	case sniffedContentType == "application/zip" && fileExtension == ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return sniffedContentType
	}
}
