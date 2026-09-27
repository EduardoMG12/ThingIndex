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
	Chunks              []models.DocumentChunk
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
	CoverageRatio        float64
	SummaryCoverageRatio float64
	TotalNodes           int
	LowConfidenceNodes   int
	OversizedNodes       int
	MaxDepth             int
	RetrievalReady       bool
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
	TreeState              string
	TreeStateDetail        string
	TreeConcernCodes       []string
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
	// TableRowHits sao linhas de tabela com pagina: a unidade mais precisa que
	// o acervo tem ("Disciplina: X; Pre-requisito: Y", Matriz p. 1).
	TableRowHits []CollectionTableRowHit
	Strategy     string
}

type CollectionTableRowHit struct {
	DocumentID uint
	TableID    uint
	RowIndex   int
	PageNumber int
	Title      string
	// Text e a linha renderizada com o cabecalho colado em cada celula.
	Text  string
	Score int
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

// NodeConnection e uma relacao DERIVADA entre um no do sumario e um no de outro
// documento do acervo.
//
// "Derivada" e a palavra que importa: a conexao e recalculada a cada consulta
// pelo scorer relacional, e nunca persistida. O principio II da constituicao
// admite grafos apenas como trilha comparativa, jamais como camada primaria de
// recuperacao -- uma tabela de arestas seria exatamente isso e refutaria a tese
// do trabalho. Aqui a aresta e a saida do recuperador, nao uma estrutura
// consultada por ele.
//
// Nao existe repositorio, migration ou modelo GORM correspondente a este tipo,
// e essa ausencia e deliberada.
type NodeConnection struct {
	DocumentID uint
	NodeID     string
	Title      string
	Summary    string
	StartPage  *int
	Score      int
	// MatchedTerms sao os termos da origem encontrados no destino. Respondem
	// "por que esta flecha existe?" -- sem isso a conexao seria tao opaca
	// quanto a vizinhanca vetorial que este trabalho critica.
	MatchedTerms []string
	// Kind: "termo-raro" ou "mesma-norma" (os dois trechos citam a mesma lei,
	// resolucao, decreto ou portaria).
	Kind string
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
	// EnsureCollection cria a colecao ou devolve a existente, resolvendo a
	// corrida no banco.
	//
	// Existe porque buscar-e-criar no usecase e um check-then-act: entre a
	// busca que nao encontra e a criacao, outra requisicao cria a mesma colecao
	// e a segunda recebe violacao de unicidade. Medido com 20 uploads
	// simultaneos para a mesma colecao: um deles voltava HTTP 500.
	//
	// So o banco pode arbitrar quem ganha a corrida; reexecutar a busca no
	// usecase apenas moveria a janela.
	EnsureCollection(ctx context.Context, collection *models.Collection) error
	UpdateCollection(ctx context.Context, collection *models.Collection) error
	GetCollectionByID(ctx context.Context, id uint, userID uint) (*models.Collection, error)
	// UpdateCollectionMetadataKey grava uma unica chave de collections.metadata
	// preservando as outras; ErrCollectionNotFound se a colecao nao e do usuario.
	UpdateCollectionMetadataKey(ctx context.Context, userID, collectionID uint, key string, valueJSON []byte) error
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
	RetryDocumentIndexing(ctx context.Context, userID uint, documentID uint, auditEvent *models.DocumentAuditEvent) (*models.Document, error)
	MarkDocumentOrphaned(ctx context.Context, userID uint, documentID uint, detail string) (*models.Document, error)
	DeleteDocument(ctx context.Context, userID uint, documentID uint, auditEvent *models.DocumentAuditEvent) error
	DeleteDocumentsByCollection(ctx context.Context, userID uint, collectionID uint, auditEvents []models.DocumentAuditEvent) (int64, error)
}

type StructuredDataRepository interface {
	ReplaceDocumentPages(ctx context.Context, documentID uint, pages []models.DocumentPage) error
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
	// GetNodeConnections devolve conexoes DERIVADAS entre um no e nos de outros
	// documentos da colecao. Nada e persistido; ver NodeConnection.
	GetNodeConnections(ctx context.Context, userID, collectionID uint, nodeID string, minScore int) ([]NodeConnection, error)
}
