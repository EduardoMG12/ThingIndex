package indexing

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func (i *DeterministicIndexer) indexSpreadsheet(documentID uint, data []byte) (domain.IndexingResult, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return domain.IndexingResult{}, fmt.Errorf("%w: parse spreadsheet: %v", domain.ErrDocumentParsingFailed, err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	headings := make([]outlineDraft, 0)
	paragraphs := make([]textBlock, 0)
	tables := make([]models.TableSection, 0)

	for sheetIndex, sheetName := range workbook.GetSheetList() {
		rows, err := workbook.GetRows(sheetName)
		if err != nil {
			return domain.IndexingResult{}, fmt.Errorf("%w: read spreadsheet rows: %v", domain.ErrDocumentParsingFailed, err)
		}

		sectionKey := slugify(sheetName)
		headings = append(headings, outlineDraft{
			Level:          1,
			Title:          sheetName,
			ParagraphStart: len(paragraphs),
			SectionKey:     sectionKey,
		})
		paragraphs = append(paragraphs, textBlock{
			Content:    fmt.Sprintf("Planilha %s com %d linhas.", sheetName, len(rows)),
			PageNumber: intPtr(sheetIndex + 1),
			SectionKey: sectionKey,
		})

		section := models.TableSection{
			DocumentID:    documentID,
			SequenceOrder: sheetIndex,
			Title:         sheetName,
			Headers:       []string{},
			Rows:          [][]string{},
			Metadata: map[string]any{
				"sheet_name":  sheetName,
				"row_count":   len(rows),
				"section_key": sectionKey,
			},
		}

		if len(rows) > 0 {
			section.Headers = normalizeCells(rows[0])
		}
		if len(rows) > 1 {
			for _, row := range rows[1:] {
				section.Rows = append(section.Rows, normalizeCells(row))
			}
		}

		tables = append(tables, section)
	}

	return buildIndexingResult(documentID, headings, paragraphs, tables), nil
}

func normalizeCells(cells []string) []string {
	result := make([]string, 0, len(cells))
	for _, cell := range cells {
		result = append(result, normalizeText(cell))
	}
	return result
}
