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

// RequestedAt torna o consumidor idempotente: o RabbitMQ pode reentregar a
// mensagem se o worker morrer antes do ack. Se Document.IndexedAt for posterior
// a RequestedAt, o trabalho ja foi feito. Zero (eventos antigos) = processar.
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
	// RequeueStuckQueuedDocuments cria um evento PENDING novo para cada documento em
	// QUEUED cujo ultimo evento esta PUBLISHED desde antes de `olderThan` (mensagem
	// perdida depois do broker).
	RequeueStuckQueuedDocuments(ctx context.Context, olderThan time.Time) (int64, error)
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

// IndexingPreferences diz o que o enriquecimento por LLM gera para os
// documentos de um usuario. Tudo ligado por padrao; o que esta desligado cai
// no fallback deterministico e fica registrado em enrichment_scope.
type IndexingPreferences struct {
	DocumentSynopsis bool `json:"document_synopsis"`
	PageSummaries    bool `json:"page_summaries"`
	SectionSummaries bool `json:"section_summaries"`
}

func DefaultIndexingPreferences() IndexingPreferences {
	return IndexingPreferences{DocumentSynopsis: true, PageSummaries: true, SectionSummaries: true}
}

type IndexingPreferencesRepository interface {
	GetIndexingPreferences(ctx context.Context, userID uint) (IndexingPreferences, error)
	SaveIndexingPreferences(ctx context.Context, userID uint, prefs IndexingPreferences) error
}

// EnrichmentCacheEntry e um resumo (e topicos) de pagina ou secao guardado
// por hash do texto: reindexar o mesmo texto nao paga a LLM de novo.
type EnrichmentCacheEntry struct {
	Kind        string // "page" | "section" | "query"
	ContentHash string
	Summary     string
	Topics      string // separados por virgula
}

// EnrichmentCacheScope e o escopo de uma entrada do cache. O nome do modelo e
// texto livre e o base_url aponta para qualquer servidor, entao uma chave global
// deixava um usuario envenenar os resumos de outro. UserID isola por usuario;
// Provider (tipo + host) evita reaproveitar resumo de outra origem.
type EnrichmentCacheScope struct {
	UserID   uint
	Provider string
	Model    string
}

type EnrichmentCache interface {
	LookupEnrichment(ctx context.Context, scope EnrichmentCacheScope, kind string, hashes []string) (map[string]EnrichmentCacheEntry, error)
	StoreEnrichment(ctx context.Context, scope EnrichmentCacheScope, entries []EnrichmentCacheEntry) error
}
