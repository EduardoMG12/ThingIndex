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
	ID                  uint                        `gorm:"primarykey" json:"id"`
	UserID              uint                        `gorm:"not null;index" json:"user_id"`
	Title               string                      `gorm:"size:255;not null" json:"title"`
	FileURL             string                      `gorm:"size:512;not null" json:"file_url"`
	Size                int64                       `json:"size"`
	MimeType            string                      `gorm:"size:100" json:"mime_type"`
	Status              string                      `gorm:"size:50;default:'UPLOADED';index" json:"status"`
	ProcessingStage     string                      `gorm:"size:64;default:'QUEUED'" json:"processing_stage,omitempty"`
	StatusDetail        string                      `gorm:"size:255" json:"status_detail,omitempty"`
	IndexStartedAt      *time.Time                  `json:"index_started_at,omitempty"`
	IndexedAt           *time.Time                  `json:"indexed_at,omitempty"`
	LastProcessingError string                      `gorm:"type:text" json:"last_processing_error,omitempty"`
	Collection          string                      `gorm:"column:collection;size:255;index" json:"-"`
	CollectionID        *uint                       `gorm:"index" json:"collection_id,omitempty"`
	CollectionRef       *Collection                 `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	CategoryID          *uint                       `gorm:"column:category_id" json:"-"`
	Category            *DocumentCategory           `gorm:"foreignKey:CategoryID" json:"-"`
	ProcessingAttempts  []DocumentProcessingAttempt `gorm:"foreignKey:DocumentID" json:"-"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
	DeletedAt           gorm.DeletedAt              `gorm:"index" json:"-"`
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

type DocumentPage struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	DocumentID    uint           `gorm:"not null;index;uniqueIndex:idx_document_pages_document_page" json:"document_id"`
	PageNumber    int            `gorm:"not null;uniqueIndex:idx_document_pages_document_page" json:"page_number"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	Summary       string         `gorm:"type:text" json:"summary,omitempty"`
	TokenEstimate int            `gorm:"not null;default:0" json:"token_estimate"`
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
	ID            uint                 `gorm:"primarykey" json:"id"`
	DocumentID    uint                 `gorm:"not null;index;uniqueIndex:idx_outline_nodes_document_sequence" json:"document_id"`
	SequenceOrder int                  `gorm:"not null;uniqueIndex:idx_outline_nodes_document_sequence" json:"sequence_order"`
	NodeID        string               `gorm:"size:128;not null;uniqueIndex:idx_outline_nodes_document_node" json:"node_id"`
	ParentID      *uint                `gorm:"index" json:"parent_id,omitempty"`
	Parent        *DocumentOutlineNode `gorm:"foreignKey:ParentID" json:"-"`
	Level         int                  `gorm:"not null" json:"level"`
	Title         string               `gorm:"size:255;not null" json:"title"`
	Slug          string               `gorm:"size:255" json:"slug,omitempty"`
	Summary       string               `gorm:"type:text" json:"summary,omitempty"`
	StartPage     *int                 `json:"start_page,omitempty"`
	EndPage       *int                 `json:"end_page,omitempty"`
	NodePath      string               `gorm:"size:1024;index" json:"node_path,omitempty"`
	Metadata      map[string]any       `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	DeletedAt     gorm.DeletedAt       `gorm:"index" json:"-"`
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

type DocumentProcessingAttempt struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	DocumentID        uint           `gorm:"not null;index:idx_document_processing_attempts_document_created,priority:1" json:"document_id"`
	Document          *Document      `gorm:"foreignKey:DocumentID" json:"-"`
	UserID            uint           `gorm:"not null;index" json:"user_id"`
	Stage             string         `gorm:"size:64;index" json:"stage,omitempty"`
	FailureKind       string         `gorm:"size:32;index" json:"failure_kind,omitempty"`
	Disposition       string         `gorm:"size:32;index" json:"disposition,omitempty"`
	Detail            string         `gorm:"type:text" json:"detail,omitempty"`
	ErrorMessage      string         `gorm:"type:text" json:"error_message,omitempty"`
	RetryCount        int            `gorm:"not null;default:0" json:"retry_count"`
	NextRetryAttempt  int            `gorm:"not null;default:0" json:"next_retry_attempt"`
	MaxRetryAttempts  int            `gorm:"not null;default:0" json:"max_retry_attempts"`
	RetryDelaySeconds int            `gorm:"not null;default:0" json:"retry_delay_seconds"`
	CreatedAt         time.Time      `gorm:"index:idx_document_processing_attempts_document_created,priority:2" json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

type DocumentAuditEvent struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	DocumentID    *uint          `gorm:"index" json:"document_id,omitempty"`
	CollectionID  *uint          `gorm:"index" json:"collection_id,omitempty"`
	ActorType     string         `gorm:"size:32;not null;default:'USER';index" json:"actor_type"`
	ActorID       *uint          `gorm:"index" json:"actor_id,omitempty"`
	Action        string         `gorm:"size:64;not null;index" json:"action"`
	Status        string         `gorm:"size:32;not null;index" json:"status"`
	CorrelationID string         `gorm:"size:64;index" json:"correlation_id,omitempty"`
	Origin        string         `gorm:"size:255;index" json:"origin,omitempty"`
	Title         string         `gorm:"size:255" json:"title,omitempty"`
	Detail        string         `gorm:"type:text" json:"detail,omitempty"`
	Metadata      map[string]any `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
