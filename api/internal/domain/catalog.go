package domain

import (
	"context"
	"io"
	"thing-index/internal/models"
	"time"
)

type DocumentContent struct {
	Document *models.Document
	Reader   io.ReadCloser
}

type CollectionContext struct {
	Collection          *models.Collection
	Documents           []models.Document
	DocumentPages       []models.DocumentPage
	ParagraphNodes      []models.ParagraphNode
	TableSections       []models.TableSection
	MetadataPairs       []models.MetadataPair
	OutlineNodes        []models.DocumentOutlineNode
	CollectionRetrieval *CollectionRetrievalResult
}

type DocumentIndex struct {
	Document        *models.Document
	DocumentPages   []models.DocumentPage
	ParagraphNodes  []models.ParagraphNode
	TableSections   []models.TableSection
	MetadataPairs   []models.MetadataPair
	Chunks          []models.DocumentChunk
	OutlineNodes    []models.DocumentOutlineNode
	DocumentSummary string
	TreeRetrieval   *TreeRetrievalResult
	IndexEvaluation *IndexEvaluationResult
	Monitoring      *IndexMonitoringResult
}

type DocumentMonitoringSummary struct {
	TotalDocuments           int
	IndexedDocuments         int
	ProcessingDocuments      int
	QueuedDocuments          int
	FailedDocuments          int
	PermanentFailedDocuments int
	RetryingDocuments        int
	OrphanedDocuments        int
	DLQDocuments             int
	DocumentsWithLLM         int
	PricingAwareDocuments    int
	TotalLLMRequests         int
	TotalInputTokens         int
	TotalOutputTokens        int
	TotalTokens              int
	TotalEstimatedCostUSD    float64
	AverageDurationMs        int64
	AverageQualityScore      float64
	LastCompletedAt          *time.Time
	ModelBreakdown           []DocumentMonitoringModelAggregate
}

type DocumentMonitoringModelAggregate struct {
	ProviderType  string
	Model         string
	DocumentCount int
	RequestCount  int
	TotalTokens   int
	TotalCostUSD  float64
}

type TreeRetrievalResult struct {
	Query          string
	NodeHits       []TreeNodeHit
	PageHits       []TreePageHit
	TraversalSteps []TreeTraversalStep
	UsedTree       bool
	Strategy       string
}

type TreeNodeHit struct {
	NodeID        string
	Title         string
	Summary       string
	Level         int
	StartPage     *int
	EndPage       *int
	Score         int
	SequenceOrder int
}

type TreePageHit struct {
	PageNumber int
	Summary    string
	Snippet    string
	Score      int
}

type TreeTraversalStep struct {
	NodeID    string
	Title     string
	Level     int
	Score     int
	StartPage *int
	EndPage   *int
}

type IndexEvaluationResult struct {
	CoverageRatio      float64
	TotalNodes         int
	LowConfidenceNodes int
	OversizedNodes     int
	MaxDepth           int
	RetrievalReady     bool
}

type IndexMonitoringResult struct {
	Status                 string
	Stage                  string
	StatusDetail           string
	StartedAt              *time.Time
	CompletedAt            *time.Time
	DurationMs             int64
	EnrichmentStatus       string
	PipelineVersion        string
	PromptProfileVersion   string
	ChunkStrategyVersion   string
	TreeBuilder            string
	TreeVerificationPassed bool
	EstimatedQualityScore  float64
	LLM                    *LLMMonitoringResult
}

type LLMMonitoringResult struct {
	ProviderType          string
	Model                 string
	RequestCount          int
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	EstimatedTotalTokens  int
	PricingAvailable      bool
	InputCostUSD          float64
	OutputCostUSD         float64
	TotalCostUSD          float64
}

type CollectionRetrievalResult struct {
	Query        string
	DocumentHits []CollectionDocumentHit
	NodeHits     []CollectionNodeHit
	PageHits     []CollectionPageHit
	Strategy     string
}

type CollectionDocumentHit struct {
	DocumentID uint
	Title      string
	Score      int
}

type CollectionNodeHit struct {
	DocumentID uint
	NodeID     string
	Title      string
	Summary    string
	Score      int
	StartPage  *int
	EndPage    *int
}

type CollectionPageHit struct {
	DocumentID uint
	PageNumber int
	Snippet    string
	Score      int
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
	CreateDocumentWithOutbox(ctx context.Context, doc *models.Document, event *models.OutboxEvent, auditEvent *models.DocumentAuditEvent) error
	CreateDocumentAuditEvents(ctx context.Context, events []models.DocumentAuditEvent) error
	ListDocumentAuditEvents(ctx context.Context, userID uint, filter DocumentAuditFilter) ([]models.DocumentAuditEvent, error)
	QueryDocumentAuditEvents(ctx context.Context, userID uint, filter DocumentAuditFilter) (*DocumentAuditQueryResult, error)
	GetDocumentByID(ctx context.Context, id uint, userID uint) (*models.Document, error)
	GetDocumentByIDForProcessing(ctx context.Context, id uint) (*models.Document, error)
	GetDocumentsByUserID(ctx context.Context, userID uint) ([]models.Document, error)
	GetDocumentsByCategoryID(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error)
	GetDocumentsByCollectionID(ctx context.Context, collectionID uint, userID uint) ([]models.Document, error)
	UpdateDocumentStatus(ctx context.Context, documentID uint, status string) error
	RetryDocumentIndexing(ctx context.Context, userID uint, documentID uint, auditEvent *models.DocumentAuditEvent) (*models.Document, error)
	MarkDocumentOrphaned(ctx context.Context, userID uint, documentID uint, detail string) (*models.Document, error)
	DeleteDocument(ctx context.Context, userID uint, documentID uint, auditEvent *models.DocumentAuditEvent) error
	DeleteDocumentsByCollection(ctx context.Context, userID uint, collectionID uint, auditEvents []models.DocumentAuditEvent) (int64, error)
}

type StructuredDataRepository interface {
	ReplaceDocumentPages(ctx context.Context, documentID uint, pages []models.DocumentPage) error
	ReplaceParagraphNodes(ctx context.Context, documentID uint, nodes []models.ParagraphNode) error
	ReplaceTableSections(ctx context.Context, documentID uint, sections []models.TableSection) error
	ReplaceMetadataPairs(ctx context.Context, documentID uint, pairs []models.MetadataPair) error
	ReplaceDocumentChunks(ctx context.Context, documentID uint, chunks []models.DocumentChunk) error
	ReplaceOutlineNodes(ctx context.Context, documentID uint, nodes []models.DocumentOutlineNode) error
	GetDocumentPagesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.DocumentPage, error)
	GetParagraphNodesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.ParagraphNode, error)
	GetTableSectionsByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.TableSection, error)
	GetMetadataPairsByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.MetadataPair, error)
	GetDocumentChunksByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.DocumentChunk, error)
	GetOutlineNodesByDocumentIDs(ctx context.Context, documentIDs []uint) ([]models.DocumentOutlineNode, error)
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
	ListDocumentAuditEvents(ctx context.Context, userID uint, filter DocumentAuditFilter) ([]models.DocumentAuditEvent, error)
	QueryDocumentAuditEvents(ctx context.Context, userID uint, filter DocumentAuditFilter) (*DocumentAuditQueryResult, error)
	GetDocument(ctx context.Context, userID, documentID uint) (*models.Document, error)
	GetDocumentContent(ctx context.Context, userID, documentID uint) (*DocumentContent, error)
	GetDocumentIndex(ctx context.Context, userID, documentID uint, query string) (*DocumentIndex, error)
	GetDocumentMonitoringSummary(ctx context.Context, userID uint) (*DocumentMonitoringSummary, error)
	RetryDocumentIndexing(ctx context.Context, userID, documentID uint) (*models.Document, error)
	DeleteDocument(ctx context.Context, userID, documentID uint) error
	DeleteCollectionDocuments(ctx context.Context, userID, collectionID uint) (int64, *models.Collection, error)
	ListDocumentsByCategory(ctx context.Context, categoryID uint, userID uint) ([]models.Document, error)
	ListCollections(ctx context.Context, userID uint) ([]models.Collection, error)
	GetCollectionContext(ctx context.Context, userID, collectionID uint, query string) (*CollectionContext, error)
	StoreParagraphNodes(ctx context.Context, userID, documentID uint, nodes []models.ParagraphNode) ([]models.ParagraphNode, error)
	StoreTableSections(ctx context.Context, userID, documentID uint, sections []models.TableSection) ([]models.TableSection, error)
	StoreMetadataPairs(ctx context.Context, userID, documentID uint, pairs []models.MetadataPair) ([]models.MetadataPair, error)
}
