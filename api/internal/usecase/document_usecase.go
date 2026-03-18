package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

	event := &models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload: map[string]any{
			"document_id": doc.ID,
			"user_id":     userID,
		},
		AvailableAt: time.Now().UTC(),
	}

	if err := u.documentRepo.CreateDocumentWithOutbox(ctx, doc, event); err != nil {
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
	document, err := u.documentRepo.RetryDocumentIndexing(ctx, userID, documentID)
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

	if err := u.deleteDocumentSource(ctx, userID, doc.FileURL); err != nil {
		return err
	}

	if err := u.documentRepo.DeleteDocument(ctx, userID, documentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrDocumentNotFound
		}
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

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

	for _, document := range documents {
		if err := u.deleteDocumentSource(ctx, userID, document.FileURL); err != nil {
			return 0, nil, err
		}
	}

	deletedCount, err := u.documentRepo.DeleteDocumentsByCollection(ctx, userID, collectionID)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

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

func (u *documentCatalogUseCase) deleteDocumentSource(ctx context.Context, userID uint, fileURL string) error {
	storageClient := u.storage
	usedCustomStorage := false
	userStorage, providerErr := u.storageResolver.ResolveStorage(ctx, userID)
	switch {
	case providerErr == nil:
		storageClient = userStorage
		usedCustomStorage = true
	case errors.Is(providerErr, domain.ErrStorageProviderNotConfigured):
	default:
		return providerErr
	}

	if err := storageClient.DeleteFile(ctx, fileURL); err == nil {
		return nil
	}

	if usedCustomStorage {
		if err := u.storage.DeleteFile(ctx, fileURL); err == nil {
			return nil
		}
	}

	return nil
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
			break
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
		index.TreeRetrieval = buildTreeRetrieval(strings.TrimSpace(query), outlineNodes, documentPages)
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
		collectionContext.CollectionRetrieval = buildCollectionRetrieval(strings.TrimSpace(query), collectionContext.Documents, collectionContext.DocumentPages, collectionContext.OutlineNodes)
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

func buildTreeRetrieval(query string, nodes []models.DocumentOutlineNode, pages []models.DocumentPage) *domain.TreeRetrievalResult {
	queryTerms := tokenizeRetrievalText(query)
	if len(queryTerms) == 0 {
		return nil
	}

	nodeHits, traversalSteps := traverseTree(queryTerms, nodes)
	pageScoreByNumber := make(map[int]int, len(pages))
	for _, step := range traversalSteps {
		if step.StartPage == nil || step.EndPage == nil {
			continue
		}
		for pageNumber := *step.StartPage; pageNumber <= *step.EndPage; pageNumber++ {
			pageScoreByNumber[pageNumber] += step.Score
		}
	}

	pageHits := make([]domain.TreePageHit, 0, len(pages))
	for _, page := range pages {
		score := pageScoreByNumber[page.PageNumber] + retrievalScore(strings.Join([]string{page.Summary, page.Content}, " "), queryTerms)
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

	return &domain.TreeRetrievalResult{
		Query:          query,
		NodeHits:       nodeHits,
		PageHits:       pageHits,
		TraversalSteps: traversalSteps,
		UsedTree:       true,
		Strategy:       "tree-iterative",
	}
}

func traverseTree(queryTerms []string, nodes []models.DocumentOutlineNode) ([]domain.TreeNodeHit, []domain.TreeTraversalStep) {
	nodeByID := make(map[string]models.DocumentOutlineNode, len(nodes))
	childrenByParent := make(map[string][]models.DocumentOutlineNode, len(nodes))
	roots := make([]models.DocumentOutlineNode, 0)

	for _, node := range nodes {
		nodeByID[node.NodeID] = node
		parentNodeID, _ := node.Metadata["parent_node_id"].(string)
		if strings.TrimSpace(parentNodeID) == "" {
			roots = append(roots, node)
			continue
		}
		childrenByParent[parentNodeID] = append(childrenByParent[parentNodeID], node)
	}

	current := rankNodesForTraversal(roots, queryTerms)
	if len(current) == 0 {
		return nil, nil
	}

	hits := make([]domain.TreeNodeHit, 0, 5)
	steps := make([]domain.TreeTraversalStep, 0, 5)
	seen := make(map[string]struct{}, len(nodes))

	for len(current) > 0 {
		best := current[0]
		if _, ok := seen[best.NodeID]; !ok {
			seen[best.NodeID] = struct{}{}
			hits = append(hits, domain.TreeNodeHit{
				NodeID:        best.NodeID,
				Title:         best.Title,
				Summary:       best.Summary,
				Level:         best.Level,
				StartPage:     best.StartPage,
				EndPage:       best.EndPage,
				Score:         best.Score,
				SequenceOrder: best.SequenceOrder,
			})
			steps = append(steps, domain.TreeTraversalStep{
				NodeID:    best.NodeID,
				Title:     best.Title,
				Level:     best.Level,
				Score:     best.Score,
				StartPage: best.StartPage,
				EndPage:   best.EndPage,
			})
		}

		children := childrenByParent[best.NodeID]
		current = rankNodesForTraversal(children, queryTerms)
		if len(hits) >= 5 {
			break
		}
	}

	return hits, steps
}

func rankNodesForTraversal(nodes []models.DocumentOutlineNode, queryTerms []string) []domain.TreeNodeHit {
	hits := make([]domain.TreeNodeHit, 0, len(nodes))
	for _, node := range nodes {
		score := retrievalScore(strings.Join([]string{node.Title, node.Summary}, " "), queryTerms)
		score += metadataIntValue(node.Metadata, "confidence_score") / 25
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
			return hits[i].SequenceOrder < hits[j].SequenceOrder
		}
		return hits[i].Score > hits[j].Score
	})

	if len(hits) > 3 {
		hits = hits[:3]
	}
	return hits
}

func buildIndexEvaluation(metadataPairs []models.MetadataPair, nodes []models.DocumentOutlineNode) *domain.IndexEvaluationResult {
	if len(metadataPairs) == 0 && len(nodes) == 0 {
		return nil
	}

	pairMap := make(map[string]string, len(metadataPairs))
	for _, pair := range metadataPairs {
		if _, exists := pairMap[pair.Key]; !exists {
			pairMap[pair.Key] = pair.Value
		}
	}

	evaluation := &domain.IndexEvaluationResult{
		CoverageRatio:      parseFloatMetric(pairMap["tree_coverage_ratio"]),
		TotalNodes:         parseIntMetric(pairMap["tree_total_nodes"], len(nodes)),
		LowConfidenceNodes: parseIntMetric(pairMap["tree_low_confidence_nodes"], 0),
		OversizedNodes:     parseIntMetric(pairMap["tree_oversized_nodes"], 0),
		MaxDepth:           parseIntMetric(pairMap["tree_max_depth"], 0),
	}
	evaluation.RetrievalReady = evaluation.CoverageRatio >= 0.6 && evaluation.TotalNodes > 0
	return evaluation
}

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
		if _, exists := pairMap[pair.Key]; !exists {
			pairMap[pair.Key] = pair.Value
		}
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

	monitoring.EstimatedQualityScore = estimateIndexQualityScore(evaluation)
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

func estimateIndexQualityScore(evaluation *domain.IndexEvaluationResult) float64 {
	if evaluation == nil {
		return 0
	}

	score := evaluation.CoverageRatio * 70
	if evaluation.RetrievalReady {
		score += 15
	}
	score += float64(max(0, 5-evaluation.LowConfidenceNodes)) * 2
	score += float64(max(0, 3-evaluation.OversizedNodes))

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
) *domain.CollectionRetrievalResult {
	queryTerms := tokenizeRetrievalText(query)
	if len(queryTerms) == 0 {
		return nil
	}

	documentTitles := make(map[uint]string, len(documents))
	for _, doc := range documents {
		documentTitles[doc.ID] = doc.Title
	}

	documentScores := make(map[uint]int, len(documents))
	nodeHits := make([]domain.CollectionNodeHit, 0)
	for _, node := range nodes {
		score := retrievalScore(strings.Join([]string{node.Title, node.Summary}, " "), queryTerms)
		score += metadataIntValue(node.Metadata, "confidence_score") / 25
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

	pageHits := make([]domain.CollectionPageHit, 0)
	for _, page := range pages {
		score := retrievalScore(strings.Join([]string{page.Summary, page.Content}, " "), queryTerms)
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
		if len(part) < 2 {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		tokens = append(tokens, part)
	}
	return tokens
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
	case "text/html":
		if (strings.HasPrefix(sniffedContentType, "text/html") || strings.HasPrefix(sniffedContentType, "text/plain")) &&
			(fileExtension == ".html" || fileExtension == ".htm") {
			return declaredContentType
		}
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		if sniffedContentType == "application/zip" && fileExtension == ".xlsx" {
			return declaredContentType
		}
	}

	switch {
	case strings.HasPrefix(sniffedContentType, "text/html"):
		return "text/html"
	case sniffedContentType == "application/pdf":
		return sniffedContentType
	default:
		return declaredContentType
	}
}
