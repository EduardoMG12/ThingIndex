package domain

import (
	"context"
	"io"
	"time"

	"thing-index/internal/models"
)

const (
	DocumentStatusUploaded   = "UPLOADED"
	DocumentStatusQueued     = "QUEUED"
	DocumentStatusProcessing = "PROCESSING"
	DocumentStatusIndexed    = "INDEXED"
	DocumentStatusFailed     = "FAILED"

	OutboxEventStatusPending   = "PENDING"
	OutboxEventStatusPublished = "PUBLISHED"
	OutboxEventStatusFailed    = "FAILED"

	OutboxAggregateDocument           = "document"
	OutboxEventDocumentIndexRequested = "document.index.requested"
)

type DocumentIndexRequestedEvent struct {
	DocumentID uint `json:"document_id"`
	UserID     uint `json:"user_id"`
}

type IndexingResult struct {
	ParagraphNodes []models.ParagraphNode
	TableSections  []models.TableSection
	MetadataPairs  []models.MetadataPair
	Chunks         []models.DocumentChunk
	OutlineNodes   []models.DocumentOutlineNode
}

type DocumentIngestionRepository interface {
	CreateDocumentWithOutbox(ctx context.Context, doc *models.Document, event *models.OutboxEvent) error
}

type OutboxRepository interface {
	ClaimPendingEvents(ctx context.Context, limit int, now time.Time) ([]models.OutboxEvent, error)
	MarkEventPublished(ctx context.Context, eventID uint, publishedAt time.Time) error
	MarkEventFailed(ctx context.Context, eventID uint, errMsg string, availableAt time.Time) error
}

type DocumentProcessingRepository interface {
	GetDocumentByIDForProcessing(ctx context.Context, id uint) (*models.Document, error)
	UpdateDocumentStatus(ctx context.Context, documentID uint, status string) error
	ReplaceIndexingResult(ctx context.Context, documentID uint, result IndexingResult) error
}

type MessagePublisher interface {
	PublishDocumentIndexRequested(ctx context.Context, event DocumentIndexRequestedEvent) error
	Close() error
}

type MessageConsumer interface {
	ConsumeDocumentIndexRequested(ctx context.Context, handler func(context.Context, DocumentIndexRequestedEvent) error) error
	Close() error
}

type DocumentReader interface {
	OpenFile(ctx context.Context, objectPath string) (io.ReadCloser, error)
}

type DocumentIndexer interface {
	Index(ctx context.Context, doc *models.Document, file io.Reader) (IndexingResult, error)
}
