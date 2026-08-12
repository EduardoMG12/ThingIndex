package indeximport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"thing-index/internal/dto"
	"thing-index/internal/indexio"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ImportAction string

const (
	ImportActionCreated ImportAction = "created"
	ImportActionUpdated ImportAction = "updated"
	ImportActionSkipped ImportAction = "skipped"
	ImportActionFailed  ImportAction = "failed"
)

type ImportFileResult struct {
	Path      string
	Action    ImportAction
	SourceKey string
	Err       error
}

type ImportReport struct {
	DiscoveredFiles  int
	CreatedDocuments int
	UpdatedDocuments int
	SkippedFiles     int
	FailedFiles      int
	Results          []ImportFileResult
}

type Service struct {
	db  *gorm.DB
	now func() time.Time
}

func OpenDatabase(databaseURL string) (*gorm.DB, error) {
	trimmed := strings.TrimSpace(databaseURL)
	if trimmed == "" {
		return nil, errors.New("database_url is required")
	}

	db, err := gorm.Open(postgres.Open(trimmed), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open import database: %w", err)
	}

	return db, nil
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:  db,
		now: time.Now,
	}
}

func (s *Service) EnsureSchema(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(
		&ImportedDocument{},
		&ImportedDocumentPage{},
		&ImportedParagraphNode{},
		&ImportedTableSection{},
		&ImportedMetadataPair{},
		&ImportedChunk{},
		&ImportedOutlineNode{},
	)
}

func (s *Service) ImportPath(ctx context.Context, rootPath string) (*ImportReport, error) {
	if err := s.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("ensure import schema: %w", err)
	}

	files, err := DiscoverExportFiles(rootPath)
	if err != nil {
		return nil, err
	}

	report := &ImportReport{
		DiscoveredFiles: len(files),
		Results:         make([]ImportFileResult, 0, len(files)),
	}

	for _, filePath := range files {
		result := s.ImportFile(ctx, filePath)
		report.Results = append(report.Results, result)

		switch result.Action {
		case ImportActionCreated:
			report.CreatedDocuments++
		case ImportActionUpdated:
			report.UpdatedDocuments++
		case ImportActionSkipped:
			report.SkippedFiles++
		default:
			report.FailedFiles++
		}
	}

	if report.FailedFiles > 0 {
		return report, fmt.Errorf("%d export file(s) failed to import", report.FailedFiles)
	}

	return report, nil
}

func (s *Service) ImportFile(ctx context.Context, filePath string) ImportFileResult {
	result := ImportFileResult{Path: filePath}

	payload, err := os.ReadFile(filePath)
	if err != nil {
		result.Action = ImportActionFailed
		result.Err = fmt.Errorf("read export file: %w", err)
		return result
	}

	exported, err := ParseExportPayload(payload)
	if err != nil {
		result.Action = ImportActionFailed
		result.Err = err
		return result
	}

	result.SourceKey = exported.Source.SourceKey
	action, err := s.importPayload(ctx, payload, exported)
	if err != nil {
		result.Action = ImportActionFailed
		result.Err = err
		return result
	}

	result.Action = action
	return result
}

func ParseExportPayload(payload []byte) (*indexio.DocumentIndexExportEnvelope, error) {
	var exported indexio.DocumentIndexExportEnvelope
	if err := json.Unmarshal(payload, &exported); err != nil {
		return nil, fmt.Errorf("decode export payload: %w", err)
	}
	if err := exported.Validate(); err != nil {
		return nil, fmt.Errorf("validate export payload: %w", err)
	}
	return &exported, nil
}

func (s *Service) importPayload(ctx context.Context, payload []byte, exported *indexio.DocumentIndexExportEnvelope) (ImportAction, error) {
	documentRecord, documentPages, paragraphNodes, tableSections, metadataPairs, chunks, outlineNodes, err := buildImportRecords(exported, payload, s.now().UTC())
	if err != nil {
		return ImportActionFailed, err
	}

	action := ImportActionCreated
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing ImportedDocument
		err := tx.Where("source_key = ?", documentRecord.SourceKey).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load existing imported document: %w", err)
		}

		if err == nil {
			if existing.PayloadHash == documentRecord.PayloadHash {
				action = ImportActionSkipped
				return nil
			}

			action = ImportActionUpdated
			documentRecord.ID = existing.ID

			if err := tx.Save(&documentRecord).Error; err != nil {
				return fmt.Errorf("update imported document: %w", err)
			}

			if err := deleteImportedChildren(tx, existing.ID); err != nil {
				return err
			}
		} else {
			if err := tx.Create(&documentRecord).Error; err != nil {
				return fmt.Errorf("create imported document: %w", err)
			}
		}

		documentID := documentRecord.ID
		if action == ImportActionSkipped {
			return nil
		}

		assignImportedDocumentID(documentID, &documentPages, &paragraphNodes, &tableSections, &metadataPairs, &chunks, &outlineNodes)

		if err := createChildRecords(tx, documentPages); err != nil {
			return err
		}
		if err := createChildRecords(tx, paragraphNodes); err != nil {
			return err
		}
		if err := createChildRecords(tx, tableSections); err != nil {
			return err
		}
		if err := createChildRecords(tx, metadataPairs); err != nil {
			return err
		}
		if err := createChildRecords(tx, chunks); err != nil {
			return err
		}
		if err := createChildRecords(tx, outlineNodes); err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		return ImportActionFailed, txErr
	}

	return action, nil
}

func deleteImportedChildren(tx *gorm.DB, importedDocumentID uint) error {
	for _, model := range []any{
		&ImportedDocumentPage{},
		&ImportedParagraphNode{},
		&ImportedTableSection{},
		&ImportedMetadataPair{},
		&ImportedChunk{},
		&ImportedOutlineNode{},
	} {
		if err := tx.Where("imported_document_id = ?", importedDocumentID).Delete(model).Error; err != nil {
			return fmt.Errorf("delete imported child rows: %w", err)
		}
	}

	return nil
}

func createChildRecords[T any](tx *gorm.DB, records []T) error {
	if len(records) == 0 {
		return nil
	}
	if err := tx.Create(&records).Error; err != nil {
		return fmt.Errorf("create imported child rows: %w", err)
	}
	return nil
}

func assignImportedDocumentID(
	importedDocumentID uint,
	pages *[]ImportedDocumentPage,
	paragraphs *[]ImportedParagraphNode,
	tables *[]ImportedTableSection,
	metadataPairs *[]ImportedMetadataPair,
	chunks *[]ImportedChunk,
	outlineNodes *[]ImportedOutlineNode,
) {
	for i := range *pages {
		(*pages)[i].ImportedDocumentID = importedDocumentID
	}
	for i := range *paragraphs {
		(*paragraphs)[i].ImportedDocumentID = importedDocumentID
	}
	for i := range *tables {
		(*tables)[i].ImportedDocumentID = importedDocumentID
	}
	for i := range *metadataPairs {
		(*metadataPairs)[i].ImportedDocumentID = importedDocumentID
	}
	for i := range *chunks {
		(*chunks)[i].ImportedDocumentID = importedDocumentID
	}
	for i := range *outlineNodes {
		(*outlineNodes)[i].ImportedDocumentID = importedDocumentID
	}
}

func buildImportRecords(
	exported *indexio.DocumentIndexExportEnvelope,
	payload []byte,
	importedAt time.Time,
) (
	ImportedDocument,
	[]ImportedDocumentPage,
	[]ImportedParagraphNode,
	[]ImportedTableSection,
	[]ImportedMetadataPair,
	[]ImportedChunk,
	[]ImportedOutlineNode,
	error,
) {
	if exported == nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, errors.New("export payload is required")
	}

	hash := sha256.Sum256(payload)
	documentJSON, err := marshalCompactJSON(exported.DocumentIndex.Document)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, fmt.Errorf("marshal document payload: %w", err)
	}
	monitoringJSON, err := marshalCompactJSON(exported.DocumentIndex.Monitoring)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, fmt.Errorf("marshal monitoring payload: %w", err)
	}
	indexEvaluationJSON, err := marshalCompactJSON(exported.DocumentIndex.IndexEvaluation)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, fmt.Errorf("marshal index evaluation payload: %w", err)
	}

	documentRecord := ImportedDocument{
		SourceKey:           exported.Source.SourceKey,
		SourceOrigin:        exported.Source.Origin,
		SourceProduct:       strings.TrimSpace(exported.Source.Product),
		SourceDocumentID:    exported.Source.DocumentID,
		SourceExportRoute:   strings.TrimSpace(exported.Source.ExportRoute),
		ExportVersion:       strings.TrimSpace(exported.Version),
		PayloadHash:         hex.EncodeToString(hash[:]),
		ExportedAt:          exported.ExportedAt.UTC(),
		ImportedAt:          importedAt.UTC(),
		Title:               strings.TrimSpace(exported.DocumentIndex.Document.Title),
		MimeType:            strings.TrimSpace(exported.DocumentIndex.Document.MimeType),
		Status:              strings.TrimSpace(exported.DocumentIndex.Document.Status),
		FileURL:             strings.TrimSpace(exported.DocumentIndex.Document.FileURL),
		FilePath:            strings.TrimSpace(exported.DocumentIndex.Document.FilePath),
		Size:                exported.DocumentIndex.Document.Size,
		CollectionName:      nestedCollectionName(exported.DocumentIndex.Document.Collection),
		CategoryName:        nestedCategoryName(exported.DocumentIndex.Document.Category),
		DocumentSummary:     strings.TrimSpace(exported.DocumentIndex.DocumentSummary),
		MonitoringJSON:      monitoringJSON,
		IndexEvaluationJSON: indexEvaluationJSON,
		RawDocumentJSON:     documentJSON,
	}

	documentPages, err := mapDocumentPages(exported.DocumentIndex.DocumentPages)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, err
	}
	paragraphNodes, err := mapParagraphNodes(exported.DocumentIndex.ParagraphNodes)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, err
	}
	tableSections, err := mapTableSections(exported.DocumentIndex.TableSections)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, err
	}
	metadataPairs := mapMetadataPairs(exported.DocumentIndex.MetadataPairs)
	chunks, err := mapChunks(exported.DocumentIndex.Chunks)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, err
	}
	outlineNodes, err := mapOutlineNodes(exported.DocumentIndex.OutlineNodes)
	if err != nil {
		return ImportedDocument{}, nil, nil, nil, nil, nil, nil, err
	}

	return documentRecord, documentPages, paragraphNodes, tableSections, metadataPairs, chunks, outlineNodes, nil
}

func mapDocumentPages(items []dto.DocumentPageResponse) ([]ImportedDocumentPage, error) {
	result := make([]ImportedDocumentPage, 0, len(items))
	for _, item := range items {
		metadataJSON, err := marshalCompactJSON(item.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal page metadata: %w", err)
		}
		result = append(result, ImportedDocumentPage{
			PageNumber:    item.PageNumber,
			Content:       item.Content,
			Summary:       item.Summary,
			TokenEstimate: item.TokenEstimate,
			MetadataJSON:  metadataJSON,
		})
	}
	return result, nil
}

func mapParagraphNodes(items []dto.ParagraphNodeResponse) ([]ImportedParagraphNode, error) {
	result := make([]ImportedParagraphNode, 0, len(items))
	for _, item := range items {
		metadataJSON, err := marshalCompactJSON(item.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal paragraph metadata: %w", err)
		}
		result = append(result, ImportedParagraphNode{
			SequenceOrder: item.SequenceOrder,
			PageNumber:    item.PageNumber,
			Content:       item.Content,
			MetadataJSON:  metadataJSON,
		})
	}
	return result, nil
}

func mapTableSections(items []dto.TableSectionResponse) ([]ImportedTableSection, error) {
	result := make([]ImportedTableSection, 0, len(items))
	for _, item := range items {
		headersJSON, err := marshalCompactJSON(item.Headers)
		if err != nil {
			return nil, fmt.Errorf("marshal table headers: %w", err)
		}
		rowsJSON, err := marshalCompactJSON(item.Rows)
		if err != nil {
			return nil, fmt.Errorf("marshal table rows: %w", err)
		}
		metadataJSON, err := marshalCompactJSON(item.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal table metadata: %w", err)
		}

		result = append(result, ImportedTableSection{
			SequenceOrder: item.SequenceOrder,
			Title:         item.Title,
			HeadersJSON:   headersJSON,
			RowsJSON:      rowsJSON,
			MetadataJSON:  metadataJSON,
		})
	}
	return result, nil
}

func mapMetadataPairs(items []dto.MetadataPairResponse) []ImportedMetadataPair {
	result := make([]ImportedMetadataPair, 0, len(items))
	for _, item := range items {
		result = append(result, ImportedMetadataPair{
			SequenceOrder: item.SequenceOrder,
			Key:           item.Key,
			Value:         item.Value,
		})
	}
	return result
}

func mapChunks(items []dto.DocumentChunkResponse) ([]ImportedChunk, error) {
	result := make([]ImportedChunk, 0, len(items))
	for _, item := range items {
		metadataJSON, err := marshalCompactJSON(item.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal chunk metadata: %w", err)
		}
		result = append(result, ImportedChunk{
			SequenceOrder:        item.SequenceOrder,
			ChunkType:            item.ChunkType,
			Title:                item.Title,
			Content:              item.Content,
			SourceParagraphStart: item.SourceParagraphStart,
			SourceParagraphEnd:   item.SourceParagraphEnd,
			TokenEstimate:        item.TokenEstimate,
			MetadataJSON:         metadataJSON,
		})
	}
	return result, nil
}

func mapOutlineNodes(items []dto.OutlineNodeResponse) ([]ImportedOutlineNode, error) {
	result := make([]ImportedOutlineNode, 0, len(items))
	nodeIDsByExportID := make(map[uint]string, len(items))
	for _, item := range items {
		nodeIDsByExportID[item.ID] = item.NodeID
	}

	for _, item := range items {
		metadataJSON, err := marshalCompactJSON(item.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal outline metadata: %w", err)
		}

		parentNodeID := ""
		if item.ParentID != nil {
			parentNodeID = nodeIDsByExportID[*item.ParentID]
		}

		result = append(result, ImportedOutlineNode{
			SequenceOrder: item.SequenceOrder,
			NodeID:        item.NodeID,
			ParentNodeID:  parentNodeID,
			Level:         item.Level,
			Title:         item.Title,
			Slug:          item.Slug,
			Summary:       item.Summary,
			StartPage:     item.StartPage,
			EndPage:       item.EndPage,
			NodePath:      item.NodePath,
			MetadataJSON:  metadataJSON,
		})
	}

	return result, nil
}

func nestedCollectionName(collection *dto.CollectionResponse) string {
	if collection == nil {
		return ""
	}
	return strings.TrimSpace(collection.Name)
}

func nestedCategoryName(category *dto.CategoryResponse) string {
	if category == nil {
		return ""
	}
	return strings.TrimSpace(category.Name)
}

func marshalCompactJSON(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}
