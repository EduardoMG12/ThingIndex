package usecase

import (
	"context"
	"fmt"
	"io"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/pkg/provider"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type documentUseCase struct {
	repo         domain.DocumentRepository
	storage      domain.StorageService
	providerUC   domain.ProviderUseCase
}

func NewDocumentUseCase(repo domain.DocumentRepository, storage domain.StorageService, providerUC domain.ProviderUseCase) domain.DocumentUseCase {
	return &documentUseCase{
		repo:       repo,
		storage:    storage,
		providerUC: providerUC,
	}
}

func (u *documentUseCase) UploadDocument(ctx context.Context, userID uint, title string, file io.Reader, size int64, contentType, categoryName, categoryDescription string) (*models.Document, error) {
	category, err := u.repo.GetCategoryByName(ctx, categoryName, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			category = &models.DocumentCategory{
				UserID:      userID,
				Name:        categoryName,
				Description: categoryDescription,
			}
			if err := u.repo.CreateCategory(ctx, category); err != nil {
				return nil, fmt.Errorf("%w: %v", domain.ErrCategoryCreate, err)
			}
		} else {
			return nil, fmt.Errorf("%w: %v", domain.ErrCategoryFind, err)
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
		CategoryID: category.ID,
	}

	if err := u.repo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	doc.Category = *category

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

func extractExtension(contentType string) string {
	switch contentType {
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	default:
		return ".bin"
	}
}
