package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/provider"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type documentUseCase struct {
	repo       domain.DocumentRepository
	storage    domain.StorageService
	providerUC domain.ProviderUseCase
}

func NewDocumentUseCase(repo domain.DocumentRepository, storage domain.StorageService, providerUC domain.ProviderUseCase) domain.DocumentUseCase {
	return &documentUseCase{
		repo:       repo,
		storage:    storage,
		providerUC: providerUC,
	}
}

func normalizeCategoryName(name string) string {
	return strings.TrimSpace(name)
}

func (u *documentUseCase) CreateCategory(ctx context.Context, userID uint, name, description string) (*models.DocumentCategory, error) {
	normalizedName := normalizeCategoryName(name)
	if normalizedName == "" {
		return nil, domain.ErrCategoryRequired
	}

	if _, err := u.repo.GetCategoryByName(ctx, normalizedName, userID); err == nil {
		return nil, domain.ErrCategoryDuplicate
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
	}

	category := &models.DocumentCategory{
		UserID:      userID,
		Name:        normalizedName,
		Description: strings.TrimSpace(description),
	}

	if err := u.repo.CreateCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryCreate, err)
	}

	return category, nil
}

func (u *documentUseCase) UpdateCategory(ctx context.Context, userID, categoryID uint, name, description *string) (*models.DocumentCategory, error) {
	category, err := u.repo.GetCategoryByID(ctx, categoryID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
	}

	if name != nil {
		normalizedName := normalizeCategoryName(*name)
		if normalizedName == "" {
			return nil, domain.ErrCategoryRequired
		}

		existing, err := u.repo.GetCategoryByName(ctx, normalizedName, userID)
		if err == nil && existing.ID != category.ID {
			return nil, domain.ErrCategoryDuplicate
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
		}

		category.Name = normalizedName
	}

	if description != nil {
		category.Description = strings.TrimSpace(*description)
	}

	if err := u.repo.UpdateCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCategoryUpdate, err)
	}

	return category, nil
}

func (u *documentUseCase) UpdateCollectionCategory(ctx context.Context, userID uint, collection string, categoryName *string, categoryDescription string) (int64, *models.DocumentCategory, error) {
	normalizedCollection := strings.TrimSpace(collection)
	if normalizedCollection == "" {
		return 0, nil, domain.ErrCollectionRequired
	}

	var category *models.DocumentCategory
	if categoryName != nil {
		normalizedCategoryName := normalizeCategoryName(*categoryName)

		if normalizedCategoryName != "" {
			existingCategory, err := u.repo.GetCategoryByName(ctx, normalizedCategoryName, userID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					category = &models.DocumentCategory{
						UserID:      userID,
						Name:        normalizedCategoryName,
						Description: strings.TrimSpace(categoryDescription),
					}
					if err := u.repo.CreateCategory(ctx, category); err != nil {
						return 0, nil, fmt.Errorf("%w: %v", domain.ErrCategoryCreate, err)
					}
				} else {
					return 0, nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
				}
			} else {
				category = existingCategory
			}
		}
	}

	var categoryID *uint
	if category != nil {
		categoryID = &category.ID
	}

	rowsAffected, err := u.repo.UpdateDocumentsCategoryByCollection(ctx, userID, normalizedCollection, categoryID)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", domain.ErrCategoryUpdate, err)
	}
	if rowsAffected == 0 {
		return 0, nil, domain.ErrCollectionNotFound
	}

	return rowsAffected, category, nil
}

func (u *documentUseCase) UploadDocument(ctx context.Context, userID uint, title string, file io.Reader, size int64, contentType, categoryName, categoryDescription, collection string) (*models.Document, error) {
	categoryName = normalizeCategoryName(categoryName)
	var category *models.DocumentCategory

	if categoryName != "" {
		existingCategory, err := u.repo.GetCategoryByName(ctx, categoryName, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				category = &models.DocumentCategory{
					UserID:      userID,
					Name:        categoryName,
					Description: strings.TrimSpace(categoryDescription),
				}
				if err := u.repo.CreateCategory(ctx, category); err != nil {
					return nil, fmt.Errorf("%w: %v", domain.ErrCategoryCreate, err)
				}
			} else {
				return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
			}
		} else {
			category = existingCategory
		}
	}

	objectName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String()[:8], extractExtension(contentType))

	storageClient := u.storage
	pType, creds, err := u.providerUC.GetFamilyCredentials(ctx, userID, domain.FamilyStorage)
	if err == nil && pType != "" {
		userSafeStorage, factorErr := provider.NewStorageProvider(pType, creds)
		if factorErr == nil {
			storageClient = userSafeStorage
		}
	}

	fileURL, err := storageClient.UploadFile(ctx, file, size, contentType, objectName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUploadFailed, err)
	}

	doc := &models.Document{
		UserID:     userID,
		Title:      title,
		FileURL:    fileURL,
		Size:       size,
		MimeType:   contentType,
		Status:     "PENDING",
		Collection: strings.TrimSpace(collection),
	}

	if category != nil {
		doc.CategoryID = &category.ID
	}

	if err := u.repo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	doc.Category = category

	return doc, nil
}

func (u *documentUseCase) ListCategories(ctx context.Context, userID uint) ([]models.DocumentCategory, error) {
	return u.repo.GetCategoriesByUserID(ctx, userID)
}

func (u *documentUseCase) ListDocuments(ctx context.Context, userID uint) ([]models.Document, error) {
	return u.repo.GetDocumentsByUserID(ctx, userID)
}

func (u *documentUseCase) ListDocumentsByCategory(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error) {
	return u.repo.GetDocumentsByCategoryID(ctx, categoryID, userID)
}

func (u *documentUseCase) ListCollections(ctx context.Context, userID uint) ([]string, error) {
	return u.repo.ListCollections(ctx, userID)
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
