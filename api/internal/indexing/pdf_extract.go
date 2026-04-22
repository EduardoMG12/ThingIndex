package indexing

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"thing-index/internal/domain"
)

func newCommandPDFTextExtractor() pdfTextExtractor {
	binPath, err := exec.LookPath("pdftotext")
	if err != nil {
		return nil
	}
	return &commandPDFTextExtractor{binPath: binPath}
}

func isPlausibleText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 10 {
		return false
	}

	letters := 0
	garbage := 0
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			letters++
		}
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			garbage++
		}
	}

	total := letters + garbage
	if total == 0 {
		return false
	}

	garbageRatio := float64(garbage) / float64(total)
	return garbageRatio < 0.1
}

func countPagesInExtractedText(text string) int {
	pageSeparator := regexp.MustCompile(`(?m)^-+\s*$`)
	return len(pageSeparator.Split(text, -1))
}

func newCommandPDFOCRExtractor() pdfOCRExtractor {
	rendererPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil
	}
	tesseractPath, err := exec.LookPath("tesseract")
	if err != nil {
		return nil
	}

	return &commandPDFOCRExtractor{
		rendererPath:  rendererPath,
		tesseractPath: tesseractPath,
	}
}

func (i *DeterministicIndexer) applyPDFOCRFallback(
	ctx context.Context,
	data []byte,
	pageCount int,
	pageBlocks map[int][]textBlock,
) (map[int][]textBlock, pdfOCRFallbackStats) {
	pagesNeedingOCR, reason := pagesNeedingPDFOCRFallback(pageCount, pageBlocks)
	stats := pdfOCRFallbackStats{
		Needed:      len(pagesNeedingOCR) > 0,
		Reason:      reason,
		PageNumbers: append([]int(nil), pagesNeedingOCR...),
	}
	if len(pagesNeedingOCR) == 0 {
		return clonePDFPageBlocks(pageBlocks), stats
	}

	merged := clonePDFPageBlocks(pageBlocks)
	if i.pdfOCRExtractor == nil {
		stats.Reason = strings.TrimSpace(stats.Reason + "_ocr_unavailable")
		return merged, stats
	}

	stats.Attempted = true
	stats.Engine = i.pdfOCRExtractor.Name()
	ocrTextByPage, err := i.pdfOCRExtractor.Extract(ctx, data, pagesNeedingOCR)
	if err != nil {
		stats.Reason = strings.TrimSpace(stats.Reason + "_ocr_failed")
		return merged, stats
	}

	appliedPages := make([]int, 0, len(pagesNeedingOCR))
	for _, pageNumber := range pagesNeedingOCR {
		text := strings.TrimSpace(ocrTextByPage[pageNumber])
		if text == "" {
			continue
		}

		blocks := blocksFromOCRPageText(text, pageNumber)
		if len(blocks) == 0 {
			continue
		}
		merged[pageNumber] = blocks
		appliedPages = append(appliedPages, pageNumber)
	}

	stats.Applied = len(appliedPages) > 0
	stats.PageNumbers = appliedPages
	if !stats.Applied {
		stats.Reason = strings.TrimSpace(reason + "_ocr_empty")
	}

	return merged, stats
}

func clonePDFPageBlocks(pageBlocks map[int][]textBlock) map[int][]textBlock {
	if len(pageBlocks) == 0 {
		return map[int][]textBlock{}
	}

	cloned := make(map[int][]textBlock, len(pageBlocks))
	for pageNumber, blocks := range pageBlocks {
		next := make([]textBlock, len(blocks))
		copy(next, blocks)
		cloned[pageNumber] = next
	}
	return cloned
}

func pagesNeedingPDFOCRFallback(pageCount int, pageBlocks map[int][]textBlock) ([]int, string) {
	if pageCount <= 0 {
		return nil, ""
	}

	pages := make([]int, 0, pageCount)
	nativePages := 0
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		blocks := pageBlocks[pageNumber]
		if !pdfPageNeedsOCRFallback(blocks) {
			nativePages++
			continue
		}
		pages = append(pages, pageNumber)
	}

	if len(pages) == 0 {
		return nil, ""
	}
	if nativePages == 0 {
		return pages, "no_native_pdf_text"
	}
	return pages, "degraded_native_pdf_text"
}

func pdfPageNeedsOCRFallback(blocks []textBlock) bool {
	text := strings.TrimSpace(textFromBlocks(blocks))
	if text == "" {
		return true
	}

	letters := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			letters++
		}
	}

	return estimateTextTokens(text) < 12 && letters < 48
}

func textFromBlocks(blocks []textBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		content := strings.TrimSpace(block.Content)
		if content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n")
}

func appendOCRFallbackMetadata(result *domain.IndexingResult, documentID uint, stats pdfOCRFallbackStats) {
	if result == nil || documentID == 0 {
		return
	}

	appendIndexingMetadataPair(result, documentID, "ocr_fallback_needed", strconv.FormatBool(stats.Needed))
	appendIndexingMetadataPair(result, documentID, "ocr_fallback_attempted", strconv.FormatBool(stats.Attempted))
	appendIndexingMetadataPair(result, documentID, "ocr_fallback_applied", strconv.FormatBool(stats.Applied))
	if strings.TrimSpace(stats.Engine) != "" {
		appendIndexingMetadataPair(result, documentID, "ocr_engine", stats.Engine)
	}
	if strings.TrimSpace(stats.Reason) != "" {
		appendIndexingMetadataPair(result, documentID, "ocr_fallback_reason", stats.Reason)
	}
	if len(stats.PageNumbers) > 0 {
		pageNumbers := make([]string, 0, len(stats.PageNumbers))
		for _, pageNumber := range stats.PageNumbers {
			pageNumbers = append(pageNumbers, strconv.Itoa(pageNumber))
		}
		appendIndexingMetadataPair(result, documentID, "ocr_page_count", strconv.Itoa(len(stats.PageNumbers)))
		appendIndexingMetadataPair(result, documentID, "ocr_page_numbers", strings.Join(pageNumbers, ","))
	}
}

func countPDFBlockPageOccurrences(pageBlocks map[int][]textBlock, pageCount int) map[string]int {
	counts := make(map[string]int)
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		blocks := pageBlocks[pageNumber]
		seenOnPage := make(map[string]struct{})
		for _, block := range blocks {
			content := strings.TrimSpace(block.Content)
			if content == "" {
				continue
			}
			seenOnPage[content] = struct{}{}
		}
		for content := range seenOnPage {
			counts[content]++
		}
	}
	return counts
}
