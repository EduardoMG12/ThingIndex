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
	DocumentStatusOrphaned   = "ORPHANED"

	OutboxEventStatusPending   = "PENDING"
	OutboxEventStatusInFlight  = "IN_FLIGHT"
	OutboxEventStatusPublished = "PUBLISHED"
	OutboxEventStatusFailed    = "FAILED"

	OutboxAggregateDocument           = "document"
	OutboxEventDocumentIndexRequested = "document.index.requested"
)

// DocumentIndexRequestedEvent e o payload que trafega do outbox ate o worker.
//
// RequestedAt existe para tornar o consumidor idempotente. O claim do outbox
// impede que o mesmo evento seja PUBLICADO duas vezes, mas nao impede que o
// RabbitMQ REENTREGUE a mesma mensagem -- o que acontece normalmente quando o
// worker morre depois de processar e antes do ack. Sem uma referencia temporal
// o consumidor nao tem como distinguir "reentrega do trabalho que ja fiz" de
// "pedido novo de reindexacao", e reprocessar custa uma rodada inteira de LLM.
//
// Comparando RequestedAt com Document.IndexedAt a distincao fica trivial: se o
// documento foi indexado DEPOIS de o evento ser criado, o trabalho ja foi feito.
// A reindexacao manual zera indexed_at (ver ReindexDocument), entao ela nunca e
// confundida com duplicata.
//
// O campo tem `omitempty` e e opcional de proposito: eventos gravados antes
// desta mudanca chegam com RequestedAt zero e caem no comportamento antigo
// (processar), em vez de serem descartados por engano.
type DocumentIndexRequestedEvent struct {
	DocumentID  uint      `json:"document_id"`
	UserID      uint      `json:"user_id"`
	RequestedAt time.Time `json:"requested_at,omitempty"`
}

type IndexingResult struct {
	ParagraphNodes []models.ParagraphNode
	DocumentPages  []models.DocumentPage
	TableSections  []models.TableSection
	MetadataPairs  []models.MetadataPair
	Chunks         []models.DocumentChunk
	OutlineNodes   []models.DocumentOutlineNode
}

type DocumentProcessingState struct {
	Status       string
	Stage        string
	StatusDetail string
	LastError    string
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

type DocumentIngestionRepository interface {
	CreateDocumentWithOutbox(ctx context.Context, doc *models.Document, event *models.OutboxEvent) error
}

type OutboxRepository interface {
	ClaimPendingEvents(ctx context.Context, limit int, now time.Time, visibilityTimeout time.Duration) ([]models.OutboxEvent, error)
	MarkEventPublished(ctx context.Context, eventID uint, publishedAt time.Time) error
	MarkEventFailed(ctx context.Context, eventID uint, errMsg string, availableAt time.Time) error
}

type DocumentProcessingRepository interface {
	GetDocumentByIDForProcessing(ctx context.Context, id uint) (*models.Document, error)
	UpdateDocumentProcessingState(ctx context.Context, documentID uint, state DocumentProcessingState) error
	CreateProcessingAttempt(ctx context.Context, attempt *models.DocumentProcessingAttempt) error
	ReplaceIndexingResult(ctx context.Context, documentID uint, result IndexingResult) error
	// ReapStuckProcessingDocuments marca como FAILED os documentos presos em
	// PROCESSING desde antes de `olderThan`. Devolve quantos foram liberados.
	ReapStuckProcessingDocuments(ctx context.Context, olderThan time.Time) (int64, error)
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
