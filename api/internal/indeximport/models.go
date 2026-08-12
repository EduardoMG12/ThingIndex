package indeximport

import "time"

type ImportedDocument struct {
	ID                  uint      `gorm:"primarykey"`
	SourceKey           string    `gorm:"size:512;not null;uniqueIndex:idx_import_documents_source_key"`
	SourceOrigin        string    `gorm:"size:255;index"`
	SourceProduct       string    `gorm:"size:64;not null"`
	SourceDocumentID    uint      `gorm:"not null;index"`
	SourceExportRoute   string    `gorm:"size:255"`
	ExportVersion       string    `gorm:"size:32;not null;index"`
	PayloadHash         string    `gorm:"size:64;not null"`
	ExportedAt          time.Time `gorm:"not null;index"`
	ImportedAt          time.Time `gorm:"not null;index"`
	Title               string    `gorm:"size:255;not null;index"`
	MimeType            string    `gorm:"size:100;index"`
	Status              string    `gorm:"size:50;index"`
	FileURL             string    `gorm:"size:1024"`
	FilePath            string    `gorm:"size:1024"`
	Size                int64
	CollectionName      string `gorm:"size:255;index"`
	CategoryName        string `gorm:"size:255;index"`
	DocumentSummary     string `gorm:"type:text"`
	MonitoringJSON      string `gorm:"type:text"`
	IndexEvaluationJSON string `gorm:"type:text"`
	RawDocumentJSON     string `gorm:"type:text"`
}

func (ImportedDocument) TableName() string {
	return "thingindex_import_documents"
}

type ImportedDocumentPage struct {
	ID                 uint   `gorm:"primarykey"`
	ImportedDocumentID uint   `gorm:"not null;index;uniqueIndex:idx_import_document_pages_document_page,priority:1"`
	PageNumber         int    `gorm:"not null;uniqueIndex:idx_import_document_pages_document_page,priority:2"`
	Content            string `gorm:"type:text;not null"`
	Summary            string `gorm:"type:text"`
	TokenEstimate      int    `gorm:"not null;default:0"`
	MetadataJSON       string `gorm:"type:text"`
}

func (ImportedDocumentPage) TableName() string {
	return "thingindex_import_document_pages"
}

type ImportedParagraphNode struct {
	ID                 uint `gorm:"primarykey"`
	ImportedDocumentID uint `gorm:"not null;index;uniqueIndex:idx_import_paragraph_nodes_document_sequence,priority:1"`
	SequenceOrder      int  `gorm:"not null;uniqueIndex:idx_import_paragraph_nodes_document_sequence,priority:2"`
	PageNumber         *int
	Content            string `gorm:"type:text;not null"`
	MetadataJSON       string `gorm:"type:text"`
}

func (ImportedParagraphNode) TableName() string {
	return "thingindex_import_paragraph_nodes"
}

type ImportedTableSection struct {
	ID                 uint   `gorm:"primarykey"`
	ImportedDocumentID uint   `gorm:"not null;index;uniqueIndex:idx_import_table_sections_document_sequence,priority:1"`
	SequenceOrder      int    `gorm:"not null;uniqueIndex:idx_import_table_sections_document_sequence,priority:2"`
	Title              string `gorm:"size:255"`
	HeadersJSON        string `gorm:"type:text"`
	RowsJSON           string `gorm:"type:text"`
	MetadataJSON       string `gorm:"type:text"`
}

func (ImportedTableSection) TableName() string {
	return "thingindex_import_table_sections"
}

type ImportedMetadataPair struct {
	ID                 uint   `gorm:"primarykey"`
	ImportedDocumentID uint   `gorm:"not null;index;uniqueIndex:idx_import_metadata_pairs_document_sequence_key,priority:1"`
	SequenceOrder      int    `gorm:"not null;uniqueIndex:idx_import_metadata_pairs_document_sequence_key,priority:2"`
	Key                string `gorm:"size:255;not null;uniqueIndex:idx_import_metadata_pairs_document_sequence_key,priority:3"`
	Value              string `gorm:"type:text;not null"`
}

func (ImportedMetadataPair) TableName() string {
	return "thingindex_import_metadata_pairs"
}

type ImportedChunk struct {
	ID                   uint   `gorm:"primarykey"`
	ImportedDocumentID   uint   `gorm:"not null;index;uniqueIndex:idx_import_chunks_document_sequence,priority:1"`
	SequenceOrder        int    `gorm:"not null;uniqueIndex:idx_import_chunks_document_sequence,priority:2"`
	ChunkType            string `gorm:"size:64;not null;index"`
	Title                string `gorm:"size:255"`
	Content              string `gorm:"type:text;not null"`
	SourceParagraphStart *int
	SourceParagraphEnd   *int
	TokenEstimate        int    `gorm:"not null;default:0"`
	MetadataJSON         string `gorm:"type:text"`
}

func (ImportedChunk) TableName() string {
	return "thingindex_import_chunks"
}

type ImportedOutlineNode struct {
	ID                 uint   `gorm:"primarykey"`
	ImportedDocumentID uint   `gorm:"not null;index;uniqueIndex:idx_import_outline_nodes_document_sequence,priority:1;uniqueIndex:idx_import_outline_nodes_document_node,priority:1"`
	SequenceOrder      int    `gorm:"not null;uniqueIndex:idx_import_outline_nodes_document_sequence,priority:2"`
	NodeID             string `gorm:"size:128;not null;uniqueIndex:idx_import_outline_nodes_document_node,priority:2"`
	ParentNodeID       string `gorm:"size:128;index"`
	Level              int    `gorm:"not null"`
	Title              string `gorm:"size:255;not null"`
	Slug               string `gorm:"size:255"`
	Summary            string `gorm:"type:text"`
	StartPage          *int
	EndPage            *int
	NodePath           string `gorm:"size:1024;index"`
	MetadataJSON       string `gorm:"type:text"`
}

func (ImportedOutlineNode) TableName() string {
	return "thingindex_import_outline_nodes"
}
