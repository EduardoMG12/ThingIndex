package indexing

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

func blocksFromPDFPage(page pdf.Page, pageNumber *int) []textBlock {
	content := page.Content()
	if len(content.Text) == 0 {
		return nil
	}

	rowBuckets := make(map[int][]pdf.Text)
	rowPositions := make([]int, 0)
	for _, text := range content.Text {
		if strings.TrimSpace(text.S) == "" {
			continue
		}

		key := int(math.Round(text.Y))
		if _, exists := rowBuckets[key]; !exists {
			rowPositions = append(rowPositions, key)
		}
		rowBuckets[key] = append(rowBuckets[key], text)
	}

	if len(rowPositions) == 0 {
		return nil
	}

	sort.Slice(rowPositions, func(i, j int) bool {
		return rowPositions[i] > rowPositions[j]
	})

	lines := make([]string, 0, len(rowPositions))
	for _, position := range rowPositions {
		texts := rowBuckets[position]
		sort.Slice(texts, func(i, j int) bool {
			if texts[i].X == texts[j].X {
				return texts[i].Y > texts[j].Y
			}
			return texts[i].X < texts[j].X
		})

		line := lineFromPDFTexts(texts)
		if line != "" {
			lines = append(lines, line)
		}
	}

	blocks := make([]textBlock, 0, len(lines))
	lineBuffer := make([]string, 0, 4)

	flush := func() {
		if len(lineBuffer) == 0 {
			return
		}
		content := normalizeText(strings.Join(lineBuffer, "\n"))
		lineBuffer = lineBuffer[:0]
		if content == "" {
			return
		}
		var pageRef *int
		if pageNumber != nil {
			pageCopy := *pageNumber
			pageRef = &pageCopy
		}
		blocks = append(blocks, textBlock{
			Content:    content,
			PageNumber: pageRef,
			Source:     "pdf_text",
		})
	}

	for _, line := range lines {
		if line == "" {
			flush()
			continue
		}

		lineBuffer = append(lineBuffer, line)
		if looksLikeHeading(line) || strings.HasSuffix(line, ".") || strings.HasSuffix(line, ":") {
			flush()
		}
	}

	flush()
	return blocks
}

func blocksFromOCRPageText(text string, pageNumber int) []textBlock {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	blocks := make([]textBlock, 0, len(lines))
	lineBuffer := make([]string, 0, 4)

	flush := func() {
		if len(lineBuffer) == 0 {
			return
		}
		content := normalizeText(strings.Join(lineBuffer, "\n"))
		lineBuffer = lineBuffer[:0]
		if content == "" {
			return
		}
		blocks = append(blocks, textBlock{
			Content:    content,
			PageNumber: intPtr(pageNumber),
			Source:     "ocr",
		})
	}

	for _, rawLine := range lines {
		line := normalizeText(rawLine)
		if line == "" {
			flush()
			continue
		}

		if looksLikePDFHeading(line) {
			flush()
			blocks = append(blocks, textBlock{
				Content:    line,
				PageNumber: intPtr(pageNumber),
				Source:     "ocr",
			})
			continue
		}

		lineBuffer = append(lineBuffer, line)
		if strings.HasSuffix(line, ".") || strings.HasSuffix(line, ":") {
			flush()
		}
	}

	flush()
	return blocks
}

func lineFromPDFTexts(texts []pdf.Text) string {
	if len(texts) == 0 {
		return ""
	}

	var builder strings.Builder
	lastEndX := 0.0
	hasContent := false
	lastText := ""

	for _, fragment := range texts {
		text := normalizeText(fragment.S)
		if text != "" {
			if hasContent {
				gap := fragment.X - lastEndX
				if shouldInsertPDFSpace(gap, lastText, text, fragment) {
					builder.WriteRune(' ')
				}
			}

			builder.WriteString(text)
			lastEndX = fragment.X + estimatedPDFFragmentWidth(fragment, text)
			hasContent = true
			lastText = text
		}
	}

	return normalizeText(builder.String())
}

func shouldInsertPDFSpace(gap float64, previousText, currentText string, fragment pdf.Text) bool {
	if gap <= 0 {
		return false
	}

	if gap > pdfSpaceThreshold(fragment) {
		return true
	}

	if gap > 1.5 && (isStandalonePunctuation(previousText) || isStandalonePunctuation(currentText)) {
		return true
	}

	return false
}

func pdfSpaceThreshold(fragment pdf.Text) float64 {
	fontSize := fragment.FontSize
	if fontSize <= 0 {
		fontSize = 8
	}

	return math.Max(fontSize*0.3, 1.5)
}

func estimatedPDFFragmentWidth(fragment pdf.Text, text string) float64 {
	width := fragment.W
	if width > 0 {
		return width
	}

	runeCount := len([]rune(text))
	if runeCount == 0 {
		return 0
	}

	fontSize := fragment.FontSize
	if fontSize <= 0 {
		fontSize = 8
	}

	return math.Max(fontSize*0.45*float64(runeCount), 1)
}

func isStandalonePunctuation(text string) bool {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) != 1 {
		return false
	}

	r := runes[0]
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}
