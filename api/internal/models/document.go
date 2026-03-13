package models

import (
	"time"

	"gorm.io/gorm"
)

type DocumentCategory struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	UserID         uint           `gorm:"not null;uniqueIndex:idx_user_category_normalized" json:"user_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	NormalizedName string         `gorm:"size:255;uniqueIndex:idx_user_category_normalized" json:"-"`
	Description    string         `gorm:"type:text" json:"description"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type Collection struct {
	ID             uint              `gorm:"primarykey" json:"id"`
	UserID         uint              `gorm:"not null;uniqueIndex:idx_user_collection_normalized;index" json:"user_id"`
	CategoryID     *uint             `json:"category_id,omitempty"`
	Category       *DocumentCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Name           string            `gorm:"size:255;not null" json:"name"`
	NormalizedName string            `gorm:"size:255;uniqueIndex:idx_user_collection_normalized" json:"-"`
	Metadata       map[string]any    `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	DeletedAt      gorm.DeletedAt    `gorm:"index" json:"-"`
	DocumentCount  int64             `gorm:"-" json:"document_count,omitempty"`
}

type Document struct {
	ID            uint              `gorm:"primarykey" json:"id"`
	UserID        uint              `gorm:"not null;index" json:"user_id"`
	Title         string            `gorm:"size:255;not null" json:"title"`
	FileURL       string            `gorm:"size:512;not null" json:"file_url"`
	Size          int64             `json:"size"`
	MimeType      string            `gorm:"size:100" json:"mime_type"`
	Status        string            `gorm:"size:50;default:'UPLOADED';index" json:"status"`
	Collection    string            `gorm:"column:collection;size:255;index" json:"-"`
	CollectionID  *uint             `gorm:"index" json:"collection_id,omitempty"`
	CollectionRef *Collection       `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	CategoryID    *uint             `gorm:"column:category_id" json:"-"`
	Category      *DocumentCategory `gorm:"foreignKey:CategoryID" json:"-"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"-"`
}

type ParagraphNode struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	DocumentID    uint           `gorm:"not null;index;uniqueIndex:idx_paragraph_nodes_document_sequence" json:"document_id"`
	SequenceOrder int            `gorm:"not null;uniqueIndex:idx_paragraph_nodes_document_sequence" json:"sequence_order"`
	PageNumber    *int           `json:"page_number,omitempty"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	Metadata      map[string]any `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type TableSection struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	DocumentID    uint           `gorm:"not null;index;uniqueIndex:idx_table_sections_document_sequence" json:"document_id"`
	SequenceOrder int            `gorm:"not null;uniqueIndex:idx_table_sections_document_sequence" json:"sequence_order"`
	Title         string         `gorm:"size:255" json:"title,omitempty"`
	Headers       []string       `gorm:"type:jsonb;serializer:json" json:"headers"`
	Rows          [][]string     `gorm:"type:jsonb;serializer:json" json:"rows"`
	Metadata      map[string]any `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type MetadataPair struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	DocumentID    uint           `gorm:"not null;index;uniqueIndex:idx_metadata_pairs_document_sequence_key,priority:1" json:"document_id"`
	SequenceOrder int            `gorm:"not null;uniqueIndex:idx_metadata_pairs_document_sequence_key,priority:2" json:"sequence_order"`
	Key           string         `gorm:"size:255;not null;uniqueIndex:idx_metadata_pairs_document_sequence_key,priority:3" json:"key"`
	Value         string         `gorm:"type:text;not null" json:"value"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type DocumentChunk struct {
	ID                   uint           `gorm:"primarykey" json:"id"`
	DocumentID           uint           `gorm:"not null;index;uniqueIndex:idx_document_chunks_document_sequence" json:"document_id"`
	SequenceOrder        int            `gorm:"not null;uniqueIndex:idx_document_chunks_document_sequence" json:"sequence_order"`
	ChunkType            string         `gorm:"size:64;not null;index" json:"chunk_type"`
	Title                string         `gorm:"size:255" json:"title,omitempty"`
	Content              string         `gorm:"type:text;not null" json:"content"`
	SourceParagraphStart *int           `json:"source_paragraph_start,omitempty"`
	SourceParagraphEnd   *int           `json:"source_paragraph_end,omitempty"`
	TokenEstimate        int            `gorm:"not null;default:0" json:"token_estimate"`
	Metadata             map[string]any `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

type DocumentOutlineNode struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	DocumentID    uint           `gorm:"not null;index;uniqueIndex:idx_outline_nodes_document_sequence" json:"document_id"`
	SequenceOrder int            `gorm:"not null;uniqueIndex:idx_outline_nodes_document_sequence" json:"sequence_order"`
	Level         int            `gorm:"not null" json:"level"`
	Title         string         `gorm:"size:255;not null" json:"title"`
	Slug          string         `gorm:"size:255" json:"slug,omitempty"`
	Summary       string         `gorm:"type:text" json:"summary,omitempty"`
	Metadata      map[string]any `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type OutboxEvent struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	AggregateType string         `gorm:"size:64;not null;index:idx_outbox_events_aggregate" json:"aggregate_type"`
	AggregateID   uint           `gorm:"not null;index:idx_outbox_events_aggregate" json:"aggregate_id"`
	EventType     string         `gorm:"size:128;not null;index" json:"event_type"`
	Status        string         `gorm:"size:32;not null;index" json:"status"`
	Payload       map[string]any `gorm:"type:jsonb;serializer:json" json:"payload"`
	Attempts      int            `gorm:"not null;default:0" json:"attempts"`
	AvailableAt   time.Time      `gorm:"not null;index" json:"available_at"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
	LastError     string         `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
