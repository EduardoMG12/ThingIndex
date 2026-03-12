package domain

import (
	"context"
	"io"
	"thing-index/internal/models"
)

type DocumentRepository interface {
	CreateCategory(ctx context.Context, category *models.DocumentCategory) error
	UpdateCategory(ctx context.Context, category *models.DocumentCategory) error
	GetCategoryByID(ctx context.Context, id uint, userID uint) (*models.DocumentCategory, error)
	GetCategoriesByUserID(ctx context.Context, userID uint) ([]models.DocumentCategory, error)
	GetCategoryByName(ctx context.Context, name string, userID uint) (*models.DocumentCategory, error)

	CreateDocument(ctx context.Context, doc *models.Document) error
	GetDocumentsByUserID(ctx context.Context, userID uint) ([]models.Document, error)
	GetDocumentsByCategoryID(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error)
}

type DocumentUseCase interface {
	UploadDocument(ctx context.Context, userID uint, title string, file io.Reader, size int64, contentType, categoryName, categoryDescription string) (*models.Document, error)
	CreateCategory(ctx context.Context, userID uint, name, description string) (*models.DocumentCategory, error)
	UpdateCategory(ctx context.Context, userID, categoryID uint, name, description *string) (*models.DocumentCategory, error)
	ListCategories(ctx context.Context, userID uint) ([]models.DocumentCategory, error)
	ListDocuments(ctx context.Context, userID uint) ([]models.Document, error)
	ListDocumentsByCategory(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error)
}
