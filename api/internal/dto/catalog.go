package dto

import (
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type CreateCollectionRequest struct {
	Name     string         `json:"name" binding:"required,min=1,max=255"`
	Category *string        `json:"category" binding:"omitempty,max=255"`
	Metadata map[string]any `json:"metadata"`
}

type ParagraphNodeInput struct {
	SequenceOrder int            `json:"sequence_order" binding:"required,gte=0"`
	PageNumber    *int           `json:"page_number,omitempty" binding:"omitempty,gte=1"`
	Content       string         `json:"content" binding:"required"`
	Metadata      map[string]any `json:"metadata"`
}

type StoreParagraphNodesRequest struct {
	Nodes []ParagraphNodeInput `json:"nodes" binding:"required"`
}

type TableSectionInput struct {
	SequenceOrder int            `json:"sequence_order" binding:"required,gte=0"`
	Title         string         `json:"title" binding:"omitempty,max=255"`
	Headers       []string       `json:"headers"`
	Rows          [][]string     `json:"rows"`
	Metadata      map[string]any `json:"metadata"`
}

type StoreTableSectionsRequest struct {
	Sections []TableSectionInput `json:"sections" binding:"required"`
}

type MetadataPairInput struct {
	SequenceOrder int    `json:"sequence_order" binding:"required,gte=0"`
	Key           string `json:"key" binding:"required,max=255"`
	Value         string `json:"value" binding:"required"`
}

type StoreMetadataPairsRequest struct {
	Pairs []MetadataPairInput `json:"pairs" binding:"required"`
}

type CategoryResponse struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CollectionResponse struct {
	ID            uint              `json:"id"`
	UserID        uint              `json:"user_id,omitempty"`
	Name          string            `json:"name"`
	CategoryID    *uint             `json:"category_id,omitempty"`
	Category      *CategoryResponse `json:"category,omitempty"`
	DocumentCount int64             `json:"document_count"`
	Metadata      map[string]any    `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type DocumentResponse struct {
	ID                  uint                                `json:"id"`
	UserID              uint                                `json:"user_id"`
	Title               string                              `json:"title"`
	FileURL             string                              `json:"file_url"`
	FilePath            string                              `json:"file_path"`
	Size                int64                               `json:"size"`
	MimeType            string                              `json:"mime_type"`
	Status              string                              `json:"status"`
	ProcessingStage     string                              `json:"processing_stage,omitempty"`
	StatusDetail        string                              `json:"status_detail,omitempty"`
	IndexStartedAt      *time.Time                          `json:"index_started_at,omitempty"`
	IndexedAt           *time.Time                          `json:"indexed_at,omitempty"`
	LastProcessingError string                              `json:"last_processing_error,omitempty"`
	CollectionID        *uint                               `json:"collection_id,omitempty"`
	Collection          *CollectionResponse                 `json:"collection,omitempty"`
	CategoryID          *uint                               `json:"category_id,omitempty"`
	Category            *CategoryResponse                   `json:"category,omitempty"`
	ProcessingAttempts  []DocumentProcessingAttemptResponse `json:"processing_attempts,omitempty"`
	CreatedAt           time.Time                           `json:"created_at"`
	UpdatedAt           time.Time                           `json:"updated_at"`
}

type DocumentProcessingAttemptResponse struct {
	ID                uint      `json:"id"`
	Stage             string    `json:"stage,omitempty"`
	FailureKind       string    `json:"failure_kind,omitempty"`
	Disposition       string    `json:"disposition,omitempty"`
	Detail            string    `json:"detail,omitempty"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	RetryCount        int       `json:"retry_count"`
	NextRetryAttempt  int       `json:"next_retry_attempt,omitempty"`
	MaxRetryAttempts  int       `json:"max_retry_attempts,omitempty"`
	RetryDelaySeconds int       `json:"retry_delay_seconds,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type ParagraphNodeResponse struct {
	ID            uint           `json:"id"`
	DocumentID    uint           `json:"document_id"`
	SequenceOrder int            `json:"sequence_order"`
	PageNumber    *int           `json:"page_number,omitempty"`
	Content       string         `json:"content"`
	Metadata      map[string]any `json:"metadata"`
}

type DocumentPageResponse struct {
	ID            uint           `json:"id"`
	DocumentID    uint           `json:"document_id"`
	PageNumber    int            `json:"page_number"`
	Content       string         `json:"content"`
	Summary       string         `json:"summary,omitempty"`
	TokenEstimate int            `json:"token_estimate"`
	Metadata      map[string]any `json:"metadata"`
}

type TableSectionResponse struct {
	ID            uint           `json:"id"`
	DocumentID    uint           `json:"document_id"`
	SequenceOrder int            `json:"sequence_order"`
	Title         string         `json:"title,omitempty"`
	Headers       []string       `json:"headers"`
	Rows          [][]string     `json:"rows"`
	Metadata      map[string]any `json:"metadata"`
}

type MetadataPairResponse struct {
	ID            uint   `json:"id"`
	DocumentID    uint   `json:"document_id"`
	SequenceOrder int    `json:"sequence_order"`
	Key           string `json:"key"`
	Value         string `json:"value"`
}

type DocumentChunkResponse struct {
	ID                   uint           `json:"id"`
	DocumentID           uint           `json:"document_id"`
	SequenceOrder        int            `json:"sequence_order"`
	ChunkType            string         `json:"chunk_type"`
	Title                string         `json:"title,omitempty"`
	Content              string         `json:"content"`
	SourceParagraphStart *int           `json:"source_paragraph_start,omitempty"`
	SourceParagraphEnd   *int           `json:"source_paragraph_end,omitempty"`
	TokenEstimate        int            `json:"token_estimate"`
	Metadata             map[string]any `json:"metadata"`
}

type OutlineNodeResponse struct {
	ID            uint           `json:"id"`
	DocumentID    uint           `json:"document_id"`
	SequenceOrder int            `json:"sequence_order"`
	NodeID        string         `json:"node_id"`
	ParentID      *uint          `json:"parent_id,omitempty"`
	Level         int            `json:"level"`
	Title         string         `json:"title"`
	Slug          string         `json:"slug,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	StartPage     *int           `json:"start_page,omitempty"`
	EndPage       *int           `json:"end_page,omitempty"`
	NodePath      string         `json:"node_path,omitempty"`
	Metadata      map[string]any `json:"metadata"`
}

type CollectionContextResponse struct {
	Collection          *CollectionResponse          `json:"collection"`
	Documents           []DocumentResponse           `json:"documents"`
	DocumentPages       []DocumentPageResponse       `json:"document_pages"`
	ParagraphNodes      []ParagraphNodeResponse      `json:"paragraph_nodes"`
	TableSections       []TableSectionResponse       `json:"table_sections"`
	MetadataPairs       []MetadataPairResponse       `json:"metadata_pairs"`
	OutlineNodes        []OutlineNodeResponse        `json:"outline_nodes"`
	CollectionRetrieval *CollectionRetrievalResponse `json:"collection_retrieval,omitempty"`
}

type DocumentIndexResponse struct {
	Document        *DocumentResponse        `json:"document"`
	DocumentPages   []DocumentPageResponse   `json:"document_pages"`
	ParagraphNodes  []ParagraphNodeResponse  `json:"paragraph_nodes"`
	TableSections   []TableSectionResponse   `json:"table_sections"`
	MetadataPairs   []MetadataPairResponse   `json:"metadata_pairs"`
	Chunks          []DocumentChunkResponse  `json:"chunks"`
	OutlineNodes    []OutlineNodeResponse    `json:"outline_nodes"`
	DocumentSummary string                   `json:"document_summary"`
	TreeRetrieval   *TreeRetrievalResponse   `json:"tree_retrieval,omitempty"`
	IndexEvaluation *IndexEvaluationResponse `json:"index_evaluation,omitempty"`
	Monitoring      *IndexMonitoringResponse `json:"monitoring,omitempty"`
}

type DocumentMonitoringSummaryResponse struct {
	TotalDocuments           int                                        `json:"total_documents"`
	IndexedDocuments         int                                        `json:"indexed_documents"`
	ProcessingDocuments      int                                        `json:"processing_documents"`
	QueuedDocuments          int                                        `json:"queued_documents"`
	FailedDocuments          int                                        `json:"failed_documents"`
	PermanentFailedDocuments int                                        `json:"permanent_failed_documents"`
	RetryingDocuments        int                                        `json:"retrying_documents"`
	OrphanedDocuments        int                                        `json:"orphaned_documents"`
	DLQDocuments             int                                        `json:"dlq_documents"`
	DocumentsWithLLM         int                                        `json:"documents_with_llm"`
	PricingAwareDocuments    int                                        `json:"pricing_aware_documents"`
	TotalLLMRequests         int                                        `json:"total_llm_requests"`
	TotalInputTokens         int                                        `json:"total_input_tokens"`
	TotalOutputTokens        int                                        `json:"total_output_tokens"`
	TotalTokens              int                                        `json:"total_tokens"`
	TotalEstimatedCostUSD    float64                                    `json:"total_estimated_cost_usd"`
	AverageDurationMs        int64                                      `json:"average_duration_ms"`
	AverageQualityScore      float64                                    `json:"average_quality_score"`
	LastCompletedAt          *time.Time                                 `json:"last_completed_at,omitempty"`
	ModelBreakdown           []DocumentMonitoringModelAggregateResponse `json:"model_breakdown"`
}

type DocumentMonitoringModelAggregateResponse struct {
	ProviderType  string  `json:"provider_type,omitempty"`
	Model         string  `json:"model,omitempty"`
	DocumentCount int     `json:"document_count"`
	RequestCount  int     `json:"request_count"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
}

type TreeRetrievalNodeHitResponse struct {
	NodeID        string `json:"node_id"`
	Title         string `json:"title"`
	Summary       string `json:"summary,omitempty"`
	StartPage     *int   `json:"start_page,omitempty"`
	EndPage       *int   `json:"end_page,omitempty"`
	Score         int    `json:"score"`
	SequenceOrder int    `json:"sequence_order"`
}

type TreeRetrievalPageHitResponse struct {
	PageNumber int    `json:"page_number"`
	Summary    string `json:"summary,omitempty"`
	Snippet    string `json:"snippet,omitempty"`
	Score      int    `json:"score"`
}

type TreeRetrievalResponse struct {
	Query          string                         `json:"query"`
	NodeHits       []TreeRetrievalNodeHitResponse `json:"node_hits"`
	PageHits       []TreeRetrievalPageHitResponse `json:"page_hits"`
	TraversalSteps []TreeTraversalStepResponse    `json:"traversal_steps"`
	UsedTree       bool                           `json:"used_tree"`
	Strategy       string                         `json:"strategy"`
}

type TreeTraversalStepResponse struct {
	NodeID    string `json:"node_id"`
	Title     string `json:"title"`
	Level     int    `json:"level"`
	Score     int    `json:"score"`
	StartPage *int   `json:"start_page,omitempty"`
	EndPage   *int   `json:"end_page,omitempty"`
}

type IndexEvaluationResponse struct {
	CoverageRatio        float64 `json:"coverage_ratio"`
	SummaryCoverageRatio float64 `json:"summary_coverage_ratio"`
	TotalNodes           int     `json:"total_nodes"`
	LowConfidenceNodes   int     `json:"low_confidence_nodes"`
	OversizedNodes       int     `json:"oversized_nodes"`
	MaxDepth             int     `json:"max_depth"`
	RetrievalReady       bool    `json:"retrieval_ready"`
}

type IndexMonitoringResponse struct {
	Status                 string                 `json:"status"`
	Stage                  string                 `json:"stage,omitempty"`
	StatusDetail           string                 `json:"status_detail,omitempty"`
	StartedAt              *time.Time             `json:"started_at,omitempty"`
	CompletedAt            *time.Time             `json:"completed_at,omitempty"`
	DurationMs             int64                  `json:"duration_ms"`
	EnrichmentStatus       string                 `json:"enrichment_status,omitempty"`
	PipelineVersion        string                 `json:"pipeline_version,omitempty"`
	PromptProfileVersion   string                 `json:"prompt_profile_version,omitempty"`
	ChunkStrategyVersion   string                 `json:"chunk_strategy_version,omitempty"`
	TreeBuilder            string                 `json:"tree_builder,omitempty"`
	TreeVerificationPassed bool                   `json:"tree_verification_passed"`
	EstimatedQualityScore  float64                `json:"estimated_quality_score"`
	LLM                    *LLMMonitoringResponse `json:"llm,omitempty"`
}

type LLMMonitoringResponse struct {
	ProviderType          string  `json:"provider_type,omitempty"`
	Model                 string  `json:"model,omitempty"`
	RequestCount          int     `json:"request_count"`
	EstimatedInputTokens  int     `json:"estimated_input_tokens"`
	EstimatedOutputTokens int     `json:"estimated_output_tokens"`
	EstimatedTotalTokens  int     `json:"estimated_total_tokens"`
	PricingAvailable      bool    `json:"pricing_available"`
	InputCostUSD          float64 `json:"input_cost_usd"`
	OutputCostUSD         float64 `json:"output_cost_usd"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
}

type CollectionRetrievalResponse struct {
	Query        string                          `json:"query"`
	DocumentHits []CollectionDocumentHitResponse `json:"document_hits"`
	NodeHits     []CollectionNodeHitResponse     `json:"node_hits"`
	PageHits     []CollectionPageHitResponse     `json:"page_hits"`
	Strategy     string                          `json:"strategy"`
}

type CollectionDocumentHitResponse struct {
	DocumentID uint   `json:"document_id"`
	Title      string `json:"title"`
	Score      int    `json:"score"`
}

type CollectionNodeHitResponse struct {
	DocumentID uint   `json:"document_id"`
	NodeID     string `json:"node_id"`
	Title      string `json:"title"`
	Summary    string `json:"summary,omitempty"`
	Score      int    `json:"score"`
	StartPage  *int   `json:"start_page,omitempty"`
	EndPage    *int   `json:"end_page,omitempty"`
}

type CollectionPageHitResponse struct {
	DocumentID uint   `json:"document_id"`
	PageNumber int    `json:"page_number"`
	Snippet    string `json:"snippet,omitempty"`
	Score      int    `json:"score"`
}

func ToCategoryResponse(category *models.DocumentCategory) *CategoryResponse {
	if category == nil {
		return nil
	}

	return &CategoryResponse{
		ID:          category.ID,
		UserID:      category.UserID,
		Name:        category.Name,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}

func ToCollectionResponse(collection *models.Collection) *CollectionResponse {
	if collection == nil {
		return nil
	}

	metadata := collection.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &CollectionResponse{
		ID:            collection.ID,
		UserID:        collection.UserID,
		Name:          collection.Name,
		CategoryID:    collection.CategoryID,
		Category:      ToCategoryResponse(collection.Category),
		DocumentCount: collection.DocumentCount,
		Metadata:      metadata,
		CreatedAt:     collection.CreatedAt,
		UpdatedAt:     collection.UpdatedAt,
	}
}

func ToCategoriesResponse(categories []models.DocumentCategory) []CategoryResponse {
	items := make([]CategoryResponse, 0, len(categories))
	for i := range categories {
		if item := ToCategoryResponse(&categories[i]); item != nil {
			items = append(items, *item)
		}
	}
	return items
}

func ToCollectionsResponse(collections []models.Collection) []CollectionResponse {
	items := make([]CollectionResponse, 0, len(collections))
	for i := range collections {
		if item := ToCollectionResponse(&collections[i]); item != nil {
			items = append(items, *item)
		}
	}
	return items
}

func ToDocumentResponse(doc *models.Document) *DocumentResponse {
	if doc == nil {
		return nil
	}

	var category *models.DocumentCategory
	var categoryID *uint
	if doc.CollectionRef != nil {
		category = doc.CollectionRef.Category
		categoryID = doc.CollectionRef.CategoryID
	}

	return &DocumentResponse{
		ID:                  doc.ID,
		UserID:              doc.UserID,
		Title:               doc.Title,
		FileURL:             doc.FileURL,
		FilePath:            doc.FileURL,
		Size:                doc.Size,
		MimeType:            doc.MimeType,
		Status:              doc.Status,
		ProcessingStage:     doc.ProcessingStage,
		StatusDetail:        doc.StatusDetail,
		IndexStartedAt:      doc.IndexStartedAt,
		IndexedAt:           doc.IndexedAt,
		LastProcessingError: doc.LastProcessingError,
		CollectionID:        doc.CollectionID,
		Collection:          ToCollectionResponse(doc.CollectionRef),
		CategoryID:          categoryID,
		Category:            ToCategoryResponse(category),
		ProcessingAttempts:  ToDocumentProcessingAttemptResponses(doc.ProcessingAttempts),
		CreatedAt:           doc.CreatedAt,
		UpdatedAt:           doc.UpdatedAt,
	}
}

func ToDocumentProcessingAttemptResponses(attempts []models.DocumentProcessingAttempt) []DocumentProcessingAttemptResponse {
	if len(attempts) == 0 {
		return nil
	}

	items := make([]DocumentProcessingAttemptResponse, 0, len(attempts))
	for i := range attempts {
		attempt := attempts[i]
		items = append(items, DocumentProcessingAttemptResponse{
			ID:                attempt.ID,
			Stage:             attempt.Stage,
			FailureKind:       attempt.FailureKind,
			Disposition:       attempt.Disposition,
			Detail:            attempt.Detail,
			ErrorMessage:      attempt.ErrorMessage,
			RetryCount:        attempt.RetryCount,
			NextRetryAttempt:  attempt.NextRetryAttempt,
			MaxRetryAttempts:  attempt.MaxRetryAttempts,
			RetryDelaySeconds: attempt.RetryDelaySeconds,
			CreatedAt:         attempt.CreatedAt,
		})
	}

	return items
}

func ToDocumentsResponse(docs []models.Document) []DocumentResponse {
	items := make([]DocumentResponse, 0, len(docs))
	for i := range docs {
		if item := ToDocumentResponse(&docs[i]); item != nil {
			items = append(items, *item)
		}
	}
	return items
}

func ToParagraphNodeResponses(nodes []models.ParagraphNode) []ParagraphNodeResponse {
	items := make([]ParagraphNodeResponse, 0, len(nodes))
	for _, node := range nodes {
		metadata := node.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}

		items = append(items, ParagraphNodeResponse{
			ID:            node.ID,
			DocumentID:    node.DocumentID,
			SequenceOrder: node.SequenceOrder,
			PageNumber:    node.PageNumber,
			Content:       node.Content,
			Metadata:      metadata,
		})
	}
	return items
}

func ToDocumentPageResponses(pages []models.DocumentPage) []DocumentPageResponse {
	items := make([]DocumentPageResponse, 0, len(pages))
	for _, page := range pages {
		metadata := page.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}

		items = append(items, DocumentPageResponse{
			ID:            page.ID,
			DocumentID:    page.DocumentID,
			PageNumber:    page.PageNumber,
			Content:       page.Content,
			Summary:       page.Summary,
			TokenEstimate: page.TokenEstimate,
			Metadata:      metadata,
		})
	}
	return items
}

func ToTableSectionResponses(sections []models.TableSection) []TableSectionResponse {
	items := make([]TableSectionResponse, 0, len(sections))
	for _, section := range sections {
		headers := section.Headers
		if headers == nil {
			headers = []string{}
		}
		rows := section.Rows
		if rows == nil {
			rows = [][]string{}
		}
		metadata := section.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}

		items = append(items, TableSectionResponse{
			ID:            section.ID,
			DocumentID:    section.DocumentID,
			SequenceOrder: section.SequenceOrder,
			Title:         section.Title,
			Headers:       headers,
			Rows:          rows,
			Metadata:      metadata,
		})
	}
	return items
}

func ToMetadataPairResponses(pairs []models.MetadataPair) []MetadataPairResponse {
	items := make([]MetadataPairResponse, 0, len(pairs))
	for _, pair := range pairs {
		items = append(items, MetadataPairResponse{
			ID:            pair.ID,
			DocumentID:    pair.DocumentID,
			SequenceOrder: pair.SequenceOrder,
			Key:           pair.Key,
			Value:         pair.Value,
		})
	}
	return items
}

func ToDocumentChunkResponses(chunks []models.DocumentChunk) []DocumentChunkResponse {
	items := make([]DocumentChunkResponse, 0, len(chunks))
	for _, chunk := range chunks {
		metadata := chunk.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}

		items = append(items, DocumentChunkResponse{
			ID:                   chunk.ID,
			DocumentID:           chunk.DocumentID,
			SequenceOrder:        chunk.SequenceOrder,
			ChunkType:            chunk.ChunkType,
			Title:                chunk.Title,
			Content:              chunk.Content,
			SourceParagraphStart: chunk.SourceParagraphStart,
			SourceParagraphEnd:   chunk.SourceParagraphEnd,
			TokenEstimate:        chunk.TokenEstimate,
			Metadata:             metadata,
		})
	}
	return items
}

func ToOutlineNodeResponses(nodes []models.DocumentOutlineNode) []OutlineNodeResponse {
	items := make([]OutlineNodeResponse, 0, len(nodes))
	for _, node := range nodes {
		metadata := node.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}

		items = append(items, OutlineNodeResponse{
			ID:            node.ID,
			DocumentID:    node.DocumentID,
			SequenceOrder: node.SequenceOrder,
			NodeID:        node.NodeID,
			ParentID:      node.ParentID,
			Level:         node.Level,
			Title:         node.Title,
			Slug:          node.Slug,
			Summary:       node.Summary,
			StartPage:     node.StartPage,
			EndPage:       node.EndPage,
			NodePath:      node.NodePath,
			Metadata:      metadata,
		})
	}

	return items
}

func ToCollectionContextResponse(ctx *domain.CollectionContext) *CollectionContextResponse {
	if ctx == nil {
		return nil
	}

	return &CollectionContextResponse{
		Collection:          ToCollectionResponse(ctx.Collection),
		Documents:           ToDocumentsResponse(ctx.Documents),
		DocumentPages:       ToDocumentPageResponses(ctx.DocumentPages),
		ParagraphNodes:      ToParagraphNodeResponses(ctx.ParagraphNodes),
		TableSections:       ToTableSectionResponses(ctx.TableSections),
		MetadataPairs:       ToMetadataPairResponses(ctx.MetadataPairs),
		OutlineNodes:        ToOutlineNodeResponses(ctx.OutlineNodes),
		CollectionRetrieval: ToCollectionRetrievalResponse(ctx.CollectionRetrieval),
	}
}

func ToDocumentIndexResponse(index *domain.DocumentIndex) *DocumentIndexResponse {
	if index == nil {
		return nil
	}

	return &DocumentIndexResponse{
		Document:        ToDocumentResponse(index.Document),
		DocumentPages:   ToDocumentPageResponses(index.DocumentPages),
		ParagraphNodes:  ToParagraphNodeResponses(index.ParagraphNodes),
		TableSections:   ToTableSectionResponses(index.TableSections),
		MetadataPairs:   ToMetadataPairResponses(index.MetadataPairs),
		Chunks:          ToDocumentChunkResponses(index.Chunks),
		OutlineNodes:    ToOutlineNodeResponses(index.OutlineNodes),
		DocumentSummary: index.DocumentSummary,
		TreeRetrieval:   ToTreeRetrievalResponse(index.TreeRetrieval),
		IndexEvaluation: ToIndexEvaluationResponse(index.IndexEvaluation),
		Monitoring:      ToIndexMonitoringResponse(index.Monitoring),
	}
}

func ToDocumentMonitoringSummaryResponse(summary *domain.DocumentMonitoringSummary) *DocumentMonitoringSummaryResponse {
	if summary == nil {
		return nil
	}

	modelBreakdown := make([]DocumentMonitoringModelAggregateResponse, 0, len(summary.ModelBreakdown))
	for _, model := range summary.ModelBreakdown {
		modelBreakdown = append(modelBreakdown, DocumentMonitoringModelAggregateResponse{
			ProviderType:  model.ProviderType,
			Model:         model.Model,
			DocumentCount: model.DocumentCount,
			RequestCount:  model.RequestCount,
			TotalTokens:   model.TotalTokens,
			TotalCostUSD:  model.TotalCostUSD,
		})
	}

	return &DocumentMonitoringSummaryResponse{
		TotalDocuments:           summary.TotalDocuments,
		IndexedDocuments:         summary.IndexedDocuments,
		ProcessingDocuments:      summary.ProcessingDocuments,
		QueuedDocuments:          summary.QueuedDocuments,
		FailedDocuments:          summary.FailedDocuments,
		PermanentFailedDocuments: summary.PermanentFailedDocuments,
		RetryingDocuments:        summary.RetryingDocuments,
		OrphanedDocuments:        summary.OrphanedDocuments,
		DLQDocuments:             summary.DLQDocuments,
		DocumentsWithLLM:         summary.DocumentsWithLLM,
		PricingAwareDocuments:    summary.PricingAwareDocuments,
		TotalLLMRequests:         summary.TotalLLMRequests,
		TotalInputTokens:         summary.TotalInputTokens,
		TotalOutputTokens:        summary.TotalOutputTokens,
		TotalTokens:              summary.TotalTokens,
		TotalEstimatedCostUSD:    summary.TotalEstimatedCostUSD,
		AverageDurationMs:        summary.AverageDurationMs,
		AverageQualityScore:      summary.AverageQualityScore,
		LastCompletedAt:          summary.LastCompletedAt,
		ModelBreakdown:           modelBreakdown,
	}
}

func ToTreeRetrievalResponse(result *domain.TreeRetrievalResult) *TreeRetrievalResponse {
	if result == nil {
		return nil
	}

	nodeHits := make([]TreeRetrievalNodeHitResponse, 0, len(result.NodeHits))
	for _, hit := range result.NodeHits {
		nodeHits = append(nodeHits, TreeRetrievalNodeHitResponse{
			NodeID:        hit.NodeID,
			Title:         hit.Title,
			Summary:       hit.Summary,
			StartPage:     hit.StartPage,
			EndPage:       hit.EndPage,
			Score:         hit.Score,
			SequenceOrder: hit.SequenceOrder,
		})
	}

	pageHits := make([]TreeRetrievalPageHitResponse, 0, len(result.PageHits))
	for _, hit := range result.PageHits {
		pageHits = append(pageHits, TreeRetrievalPageHitResponse{
			PageNumber: hit.PageNumber,
			Summary:    hit.Summary,
			Snippet:    hit.Snippet,
			Score:      hit.Score,
		})
	}

	steps := make([]TreeTraversalStepResponse, 0, len(result.TraversalSteps))
	for _, step := range result.TraversalSteps {
		steps = append(steps, TreeTraversalStepResponse{
			NodeID:    step.NodeID,
			Title:     step.Title,
			Level:     step.Level,
			Score:     step.Score,
			StartPage: step.StartPage,
			EndPage:   step.EndPage,
		})
	}

	return &TreeRetrievalResponse{
		Query:          result.Query,
		NodeHits:       nodeHits,
		PageHits:       pageHits,
		TraversalSteps: steps,
		UsedTree:       result.UsedTree,
		Strategy:       result.Strategy,
	}
}

func ToIndexEvaluationResponse(result *domain.IndexEvaluationResult) *IndexEvaluationResponse {
	if result == nil {
		return nil
	}

	return &IndexEvaluationResponse{
		CoverageRatio:        result.CoverageRatio,
		SummaryCoverageRatio: result.SummaryCoverageRatio,
		TotalNodes:           result.TotalNodes,
		LowConfidenceNodes:   result.LowConfidenceNodes,
		OversizedNodes:       result.OversizedNodes,
		MaxDepth:             result.MaxDepth,
		RetrievalReady:       result.RetrievalReady,
	}
}

func ToIndexMonitoringResponse(result *domain.IndexMonitoringResult) *IndexMonitoringResponse {
	if result == nil {
		return nil
	}

	return &IndexMonitoringResponse{
		Status:                 result.Status,
		Stage:                  result.Stage,
		StatusDetail:           result.StatusDetail,
		StartedAt:              result.StartedAt,
		CompletedAt:            result.CompletedAt,
		DurationMs:             result.DurationMs,
		EnrichmentStatus:       result.EnrichmentStatus,
		PipelineVersion:        result.PipelineVersion,
		PromptProfileVersion:   result.PromptProfileVersion,
		ChunkStrategyVersion:   result.ChunkStrategyVersion,
		TreeBuilder:            result.TreeBuilder,
		TreeVerificationPassed: result.TreeVerificationPassed,
		EstimatedQualityScore:  result.EstimatedQualityScore,
		LLM:                    ToLLMMonitoringResponse(result.LLM),
	}
}

func ToLLMMonitoringResponse(result *domain.LLMMonitoringResult) *LLMMonitoringResponse {
	if result == nil {
		return nil
	}

	return &LLMMonitoringResponse{
		ProviderType:          result.ProviderType,
		Model:                 result.Model,
		RequestCount:          result.RequestCount,
		EstimatedInputTokens:  result.EstimatedInputTokens,
		EstimatedOutputTokens: result.EstimatedOutputTokens,
		EstimatedTotalTokens:  result.EstimatedTotalTokens,
		PricingAvailable:      result.PricingAvailable,
		InputCostUSD:          result.InputCostUSD,
		OutputCostUSD:         result.OutputCostUSD,
		TotalCostUSD:          result.TotalCostUSD,
	}
}

func ToCollectionRetrievalResponse(result *domain.CollectionRetrievalResult) *CollectionRetrievalResponse {
	if result == nil {
		return nil
	}

	documentHits := make([]CollectionDocumentHitResponse, 0, len(result.DocumentHits))
	for _, hit := range result.DocumentHits {
		documentHits = append(documentHits, CollectionDocumentHitResponse{
			DocumentID: hit.DocumentID,
			Title:      hit.Title,
			Score:      hit.Score,
		})
	}

	nodeHits := make([]CollectionNodeHitResponse, 0, len(result.NodeHits))
	for _, hit := range result.NodeHits {
		nodeHits = append(nodeHits, CollectionNodeHitResponse{
			DocumentID: hit.DocumentID,
			NodeID:     hit.NodeID,
			Title:      hit.Title,
			Summary:    hit.Summary,
			Score:      hit.Score,
			StartPage:  hit.StartPage,
			EndPage:    hit.EndPage,
		})
	}

	pageHits := make([]CollectionPageHitResponse, 0, len(result.PageHits))
	for _, hit := range result.PageHits {
		pageHits = append(pageHits, CollectionPageHitResponse{
			DocumentID: hit.DocumentID,
			PageNumber: hit.PageNumber,
			Snippet:    hit.Snippet,
			Score:      hit.Score,
		})
	}

	return &CollectionRetrievalResponse{
		Query:        result.Query,
		DocumentHits: documentHits,
		NodeHits:     nodeHits,
		PageHits:     pageHits,
		Strategy:     result.Strategy,
	}
}
