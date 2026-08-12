package indexio

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"thing-index/internal/dto"
)

const (
	DocumentIndexExportKind         = "document_index_export"
	DocumentIndexExportVersion      = "v1"
	DocumentIndexExportOriginHeader = "X-ThingIndex-Export-Origin"
)

type DocumentIndexExportEnvelope struct {
	Kind          string                     `json:"kind"`
	Version       string                     `json:"version"`
	ExportedAt    time.Time                  `json:"exported_at"`
	Source        DocumentIndexExportSource  `json:"source"`
	DocumentIndex *dto.DocumentIndexResponse `json:"document_index"`
}

type DocumentIndexExportSource struct {
	Product       string `json:"product"`
	Origin        string `json:"origin,omitempty"`
	SourceKey     string `json:"source_key"`
	DocumentID    uint   `json:"document_id"`
	DocumentTitle string `json:"document_title,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	ExportRoute   string `json:"export_route"`
}

func NewDocumentIndexExport(origin string, index *dto.DocumentIndexResponse, exportedAt time.Time) *DocumentIndexExportEnvelope {
	if index == nil || index.Document == nil {
		return nil
	}

	normalizedOrigin := NormalizeExportOrigin(origin)
	documentID := index.Document.ID

	return &DocumentIndexExportEnvelope{
		Kind:       DocumentIndexExportKind,
		Version:    DocumentIndexExportVersion,
		ExportedAt: exportedAt.UTC(),
		Source: DocumentIndexExportSource{
			Product:       "ThingIndex",
			Origin:        normalizedOrigin,
			SourceKey:     BuildDocumentExportSourceKey(normalizedOrigin, documentID),
			DocumentID:    documentID,
			DocumentTitle: strings.TrimSpace(index.Document.Title),
			MimeType:      strings.TrimSpace(index.Document.MimeType),
			ExportRoute:   fmt.Sprintf("/api/v1/documents/%d/index/export", documentID),
		},
		DocumentIndex: index,
	}
}

func NormalizeExportOrigin(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(trimmed, "/")
	}

	return strings.TrimRight(parsed.Scheme+"://"+parsed.Host, "/")
}

func BuildDocumentExportSourceKey(origin string, documentID uint) string {
	normalizedOrigin := NormalizeExportOrigin(origin)
	if normalizedOrigin == "" {
		return fmt.Sprintf("thingindex:document:%d", documentID)
	}

	return fmt.Sprintf("thingindex:%s:document:%d", url.QueryEscape(normalizedOrigin), documentID)
}

func (e *DocumentIndexExportEnvelope) Validate() error {
	switch {
	case e == nil:
		return errors.New("export payload is required")
	case strings.TrimSpace(e.Kind) != DocumentIndexExportKind:
		return fmt.Errorf("unexpected export kind %q", strings.TrimSpace(e.Kind))
	case strings.TrimSpace(e.Version) != DocumentIndexExportVersion:
		return fmt.Errorf("unexpected export version %q", strings.TrimSpace(e.Version))
	case e.ExportedAt.IsZero():
		return errors.New("exported_at is required")
	case e.DocumentIndex == nil:
		return errors.New("document_index is required")
	case e.DocumentIndex.Document == nil:
		return errors.New("document_index.document is required")
	case e.Source.DocumentID == 0:
		return errors.New("source.document_id is required")
	case strings.TrimSpace(e.Source.SourceKey) == "":
		return errors.New("source.source_key is required")
	}

	if e.DocumentIndex.Document.ID != e.Source.DocumentID {
		return fmt.Errorf("source.document_id %d does not match document_index.document.id %d", e.Source.DocumentID, e.DocumentIndex.Document.ID)
	}

	return nil
}
