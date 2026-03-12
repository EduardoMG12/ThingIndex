package domain

import (
	"context"
	"io"
	"thing-index/internal/models"
)

type CollectionContext struct {
	Collection     *models.Collection
	Documents      []models.Document
	ParagraphNodes []models.ParagraphNode
	TableSections  []models.TableSection
	MetadataPairs  []models.MetadataPair
}

type CategoryCatalogRepository interface {
	CreateCategory(ctx context.Context, category *models.DocumentCategory) error
	UpdateCategory(ctx context.Context, category *models.DocumentCategory) error
	GetCategoryByID(ctx context.Context, id uint, userID uint) (*models.DocumentCategory, error)
	GetCategoryByNormalizedName(ctx context.Context, normalizedName string, userID uint) (*models.DocumentCategory, error)
	ListCategoriesByUserID(ctx context.Context, userID uint) ([]models.DocumentCategory, error)
}

type CollectionRepository interface {
	CreateCollection(ctx context.Context, collection *models.Collection) error
	UpdateCollection(ctx context.Context, collection *models.Collection) error
	GetCollectionByID(ctx context.Context, id uint, userID uint) (*models.Collection, error)
	GetCollectionByNormalizedName(ctx context.Context, normalizedName string, userID uint) (*models.Collection, error)
	ListCollectionsByUserID(ctx context.Context, userID uint) ([]models.Collection, error)
	CountDocuments(ctx context.Context, userID uint, collectionID uint) (int64, error)
}

type CatalogDocumentRepository interface {
	CreateDocument(ctx context.Context, doc *models.Document) error
	GetDocumentByID(ctx context.Context, id uint, userID uint) (*models.Document, error)
	GetDocumentsByUserID(ctx context.Context, userID uint) ([]models.Document, error)
	GetDocumentsByCategoryID(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error)
	GetDocumentsByCollectionID(ctx context.Context, collectionID uint, userID uint) ([]models.Document, error)
}

type StructuredDataRepository interface {
	ReplaceParagraphNodes(ctx context.Context, documentID uint, nodes []models.ParagraphNode) error
	ReplaceTableSections(ctx context.Context, documentID uint, sections []models.TableSection) error
	ReplaceMetadataPairs(ctx context.Context, documentID uint, pairs []models.MetadataPair) error
	GetParagraphNodesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.ParagraphNode, error)
	GetTableSectionsByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.TableSection, error)
	GetMetadataPairsByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.MetadataPair, error)
}

type CollectionContextRepository interface {
	GetCollectionContext(ctx context.Context, userID uint, collectionID uint) (*CollectionContext, error)
}

type DocumentCatalogUseCase interface {
	UploadDocument(ctx context.Context, userID uint, title string, file io.Reader, size int64, contentType, categoryName, categoryDescription, collection string) (*models.Document, error)
	CreateCategory(ctx context.Context, userID uint, name, description string) (*models.DocumentCategory, error)
	UpdateCategory(ctx context.Context, userID, categoryID uint, name, description *string) (*models.DocumentCategory, error)
	CreateCollection(ctx context.Context, userID uint, name string, categoryName *string, metadata map[string]any) (*models.Collection, error)
	GetCollection(ctx context.Context, userID, collectionID uint) (*models.Collection, error)
	UpdateCollectionCategory(ctx context.Context, userID uint, collectionIdentifier string, categoryName *string, categoryDescription string) (int64, *models.Collection, error)
	ListCategories(ctx context.Context, userID uint) ([]models.DocumentCategory, error)
	ListDocuments(ctx context.Context, userID uint) ([]models.Document, error)
	GetDocument(ctx context.Context, userID, documentID uint) (*models.Document, error)
	ListDocumentsByCategory(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error)
	ListCollections(ctx context.Context, userID uint) ([]models.Collection, error)
	GetCollectionContext(ctx context.Context, userID, collectionID uint) (*CollectionContext, error)
	StoreParagraphNodes(ctx context.Context, userID, documentID uint, nodes []models.ParagraphNode) ([]models.ParagraphNode, error)
	StoreTableSections(ctx context.Context, userID, documentID uint, sections []models.TableSection) ([]models.TableSection, error)
	StoreMetadataPairs(ctx context.Context, userID, documentID uint, pairs []models.MetadataPair) ([]models.MetadataPair, error)
}
