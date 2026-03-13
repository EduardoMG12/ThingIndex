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
	ID           uint                `json:"id"`
	UserID       uint                `json:"user_id"`
	Title        string              `json:"title"`
	FileURL      string              `json:"file_url"`
	FilePath     string              `json:"file_path"`
	Size         int64               `json:"size"`
	MimeType     string              `json:"mime_type"`
	Status       string              `json:"status"`
	CollectionID *uint               `json:"collection_id,omitempty"`
	Collection   *CollectionResponse `json:"collection,omitempty"`
	CategoryID   *uint               `json:"category_id,omitempty"`
	Category     *CategoryResponse   `json:"category,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type ParagraphNodeResponse struct {
	ID            uint           `json:"id"`
	DocumentID    uint           `json:"document_id"`
	SequenceOrder int            `json:"sequence_order"`
	PageNumber    *int           `json:"page_number,omitempty"`
	Content       string         `json:"content"`
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
	Level         int            `json:"level"`
	Title         string         `json:"title"`
	Slug          string         `json:"slug,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	Metadata      map[string]any `json:"metadata"`
}

type CollectionContextResponse struct {
	Collection     *CollectionResponse     `json:"collection"`
	Documents      []DocumentResponse      `json:"documents"`
	ParagraphNodes []ParagraphNodeResponse `json:"paragraph_nodes"`
	TableSections  []TableSectionResponse  `json:"table_sections"`
	MetadataPairs  []MetadataPairResponse  `json:"metadata_pairs"`
	OutlineNodes   []OutlineNodeResponse   `json:"outline_nodes"`
}

type DocumentIndexResponse struct {
	Document        *DocumentResponse       `json:"document"`
	ParagraphNodes  []ParagraphNodeResponse `json:"paragraph_nodes"`
	TableSections   []TableSectionResponse  `json:"table_sections"`
	MetadataPairs   []MetadataPairResponse  `json:"metadata_pairs"`
	Chunks          []DocumentChunkResponse `json:"chunks"`
	OutlineNodes    []OutlineNodeResponse   `json:"outline_nodes"`
	DocumentSummary string                  `json:"document_summary"`
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
		ID:           doc.ID,
		UserID:       doc.UserID,
		Title:        doc.Title,
		FileURL:      doc.FileURL,
		FilePath:     doc.FileURL,
		Size:         doc.Size,
		MimeType:     doc.MimeType,
		Status:       doc.Status,
		CollectionID: doc.CollectionID,
		Collection:   ToCollectionResponse(doc.CollectionRef),
		CategoryID:   categoryID,
		Category:     ToCategoryResponse(category),
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
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
			Level:         node.Level,
			Title:         node.Title,
			Slug:          node.Slug,
			Summary:       node.Summary,
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
		Collection:     ToCollectionResponse(ctx.Collection),
		Documents:      ToDocumentsResponse(ctx.Documents),
		ParagraphNodes: ToParagraphNodeResponses(ctx.ParagraphNodes),
		TableSections:  ToTableSectionResponses(ctx.TableSections),
		MetadataPairs:  ToMetadataPairResponses(ctx.MetadataPairs),
		OutlineNodes:   ToOutlineNodeResponses(ctx.OutlineNodes),
	}
}

func ToDocumentIndexResponse(index *domain.DocumentIndex) *DocumentIndexResponse {
	if index == nil {
		return nil
	}

	return &DocumentIndexResponse{
		Document:        ToDocumentResponse(index.Document),
		ParagraphNodes:  ToParagraphNodeResponses(index.ParagraphNodes),
		TableSections:   ToTableSectionResponses(index.TableSections),
		MetadataPairs:   ToMetadataPairResponses(index.MetadataPairs),
		Chunks:          ToDocumentChunkResponses(index.Chunks),
		OutlineNodes:    ToOutlineNodeResponses(index.OutlineNodes),
		DocumentSummary: index.DocumentSummary,
	}
}
