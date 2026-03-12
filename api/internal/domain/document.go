package domain

import (
	"context"
	"io"
	"thing-index/internal/models"
)

type DocumentRepository interface {
	CreateCategory(ctx context.Context, category *models.DocumentCategory) error
	GetCategoryByID(ctx context.Context, id uint) (*models.DocumentCategory, error)
	GetCategoryByName(ctx context.Context, name string) (*models.DocumentCategory, error)
	CreateDocument(ctx context.Context, doc *models.Document) error
}

type DocumentUseCase interface {
	UploadDocument(ctx context.Context, title string, file io.Reader, size int64, contentType, categoryName, categoryDescription string) (*models.Document, error)
}
