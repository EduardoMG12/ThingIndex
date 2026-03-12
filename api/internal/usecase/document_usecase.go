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
	"strconv"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/provider"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxUploadSizeBytes int64 = 20 * 1024 * 1024

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
	providerUC            domain.ProviderUseCase
}

func NewDocumentUseCase(
	categoryRepo domain.CategoryCatalogRepository,
	collectionRepo domain.CollectionRepository,
	documentRepo domain.CatalogDocumentRepository,
	structuredDataRepo domain.StructuredDataRepository,
	collectionContextRepo domain.CollectionContextRepository,
	storage domain.StorageService,
	providerUC domain.ProviderUseCase,
) domain.DocumentCatalogUseCase {
	return &documentCatalogUseCase{
		categoryRepo:          categoryRepo,
		collectionRepo:        collectionRepo,
		documentRepo:          documentRepo,
		structuredDataRepo:    structuredDataRepo,
		collectionContextRepo: collectionContextRepo,
		storage:               storage,
		providerUC:            providerUC,
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
	if size > maxUploadSizeBytes {
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
	pType, creds, providerErr := u.providerUC.GetFamilyCredentials(ctx, userID, domain.FamilyStorage)
	if providerErr == nil && pType != "" {
		userSafeStorage, factorErr := provider.NewStorageProvider(pType, creds)
		if factorErr == nil {
			storageClient = userSafeStorage
		}
	}

	fileURL, err := storageClient.UploadFile(ctx, validatedReader, size, resolvedContentType, objectName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUploadFailed, err)
	}

	doc := &models.Document{
		UserID:        userID,
		Title:         strings.TrimSpace(title),
		FileURL:       fileURL,
		Size:          size,
		MimeType:      resolvedContentType,
		Status:        "PENDING",
		Collection:    collectionRef.Name,
		CollectionID:  &collectionRef.ID,
		CollectionRef: collectionRef,
	}
	if category != nil {
		doc.CategoryID = &category.ID
	}

	if err := u.documentRepo.CreateDocument(ctx, doc); err != nil {
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

func (u *documentCatalogUseCase) ListDocumentsByCategory(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error) {
	return u.documentRepo.GetDocumentsByCategoryID(ctx, categoryID, userID)
}

func (u *documentCatalogUseCase) ListCollections(ctx context.Context, userID uint) ([]models.Collection, error) {
	return u.collectionRepo.ListCollectionsByUserID(ctx, userID)
}

func (u *documentCatalogUseCase) GetCollectionContext(ctx context.Context, userID, collectionID uint) (*domain.CollectionContext, error) {
	collectionContext, err := u.collectionContextRepo.GetCollectionContext(ctx, userID, collectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
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
