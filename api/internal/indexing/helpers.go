package indexing

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/ledongthuc/pdf"
)

const (
	maxOutlineNodeIDLength   = 128
	maxOutlineSlugLength     = 255
	maxOutlineNodePathLength = 1024
)

func (e *commandPDFTextExtractor) Name() string {
	return "pdftotext"
}

func (e *commandPDFTextExtractor) Extract(data []byte) (pdfTextExtraction, error) {
	if e == nil {
		return pdfTextExtraction{}, fmt.Errorf("pdftotext not available")
	}

	tempDir, err := os.MkdirTemp("", "thingindex-pdf-extract-*")
	if err != nil {
		return pdfTextExtraction{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	pdfPath := filepath.Join(tempDir, "document.pdf")
	txtPath := filepath.Join(tempDir, "document.txt")

	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return pdfTextExtraction{}, fmt.Errorf("write temporary pdf: %w", err)
	}

	// -bbox-layout entrega paginas explicitas e a caixa de cada bloco/linha ja no
	// espaco renderizado (origem superior esquerda, /Rotate e CropBox aplicados).
	cmd := exec.Command(e.binPath, "-bbox-layout", "-enc", "UTF-8", pdfPath, txtPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return pdfTextExtraction{}, fmt.Errorf("pdftotext failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	rawText, err := os.ReadFile(txtPath)
	if err != nil {
		return pdfTextExtraction{}, fmt.Errorf("read extracted text: %w", err)
	}

	extraction, err := parsePDFBBoxLayout(rawText)
	if err != nil {
		return pdfTextExtraction{}, err
	}

	if !isPlausibleText(textFromPageBlocks(extraction.PageBlocks)) {
		// Sem camada de texto (PDF digitalizado) nao e erro: o fallback de OCR
		// decide pagina a pagina. Retornar erro aqui pularia o OCR.
		if extraction.PageCount == 0 {
			return pdfTextExtraction{}, fmt.Errorf("pdftotext returned no pages")
		}
		extraction.PageBlocks = map[int][]textBlock{}
	}

	return extraction, nil
}

type pdfBBoxDocument struct {
	Pages []pdfBBoxPage `xml:"body>doc>page"`
}

type pdfBBoxPage struct {
	Width  float64        `xml:"width,attr"`
	Height float64        `xml:"height,attr"`
	Blocks []pdfBBoxBlock `xml:"flow>block"`
}

type pdfBBoxBlock struct {
	XMin  float64       `xml:"xMin,attr"`
	YMin  float64       `xml:"yMin,attr"`
	XMax  float64       `xml:"xMax,attr"`
	YMax  float64       `xml:"yMax,attr"`
	Lines []pdfBBoxLine `xml:"line"`
}

type pdfBBoxLine struct {
	XMin  float64       `xml:"xMin,attr"`
	YMin  float64       `xml:"yMin,attr"`
	XMax  float64       `xml:"xMax,attr"`
	YMax  float64       `xml:"yMax,attr"`
	Words []pdfBBoxWord `xml:"word"`
}

// pdfBBoxWord: a deteccao de tabela precisa da coordenada de cada palavra.
type pdfBBoxWord struct {
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
	Text string  `xml:",chardata"`
}

func (l pdfBBoxLine) text() string {
	parts := make([]string, 0, len(l.Words))
	for _, w := range l.Words {
		parts = append(parts, w.Text)
	}
	return strings.Join(parts, " ")
}

// sanitizeXMLControlChars troca por espaco os caracteres de controle que o
// XML 1.0 proibe (C0 exceto tab, LF e CR). O pdftotext copia o que esta no
// PDF (ex.: ESC dentro de <word>) e o decoder abortaria o documento inteiro.
func sanitizeXMLControlChars(raw []byte) []byte {
	saida := make([]byte, len(raw))
	for i, b := range raw {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			saida[i] = ' '
			continue
		}
		saida[i] = b
	}
	return saida
}

func parsePDFBBoxLayout(raw []byte) (pdfTextExtraction, error) {
	var document pdfBBoxDocument
	decoder := xml.NewDecoder(bytes.NewReader(sanitizeXMLControlChars(raw)))
	// A saida e XHTML e declara entidades HTML que o parser estrito rejeita.
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	if err := decoder.Decode(&document); err != nil {
		return pdfTextExtraction{}, fmt.Errorf("parse pdftotext bbox output: %w", err)
	}

	extraction := pdfTextExtraction{
		PageCount:  len(document.Pages),
		PageBlocks: make(map[int][]textBlock, len(document.Pages)),
		PageTables: make(map[int][]models.TableSection, len(document.Pages)),
	}

	for pageIndex, page := range document.Pages {
		pageNumber := pageIndex + 1
		blocks := make([]textBlock, 0, len(page.Blocks))

		for _, block := range page.Blocks {
			lines := make([]string, 0, len(block.Lines))
			lineBoxes := make([]textLine, 0, len(block.Lines))
			for _, line := range block.Lines {
				if text := strings.TrimSpace(line.text()); text != "" {
					lines = append(lines, text)
					lineBoxes = append(lineBoxes, textLine{Text: text, Box: textBox{XMin: line.XMin, YMin: line.YMin, XMax: line.XMax, YMax: line.YMax, PageWidth: page.Width, PageHeight: page.Height}})
				}
			}

			content := normalizeText(strings.Join(lines, "\n"))
			if content == "" {
				continue
			}

			pageRef := pageNumber
			blocks = append(blocks, textBlock{
				Content:    content,
				PageNumber: &pageRef,
				Source:     "pdftotext",
				LineCount:  len(lines),
				Lines:      lineBoxes,
				Box: &textBox{
					XMin:       block.XMin,
					YMin:       block.YMin,
					XMax:       block.XMax,
					YMax:       block.YMax,
					PageWidth:  page.Width,
					PageHeight: page.Height,
				},
			})
		}

		if blocks = mergeBlocksOnSameRow(blocks); len(blocks) > 0 {
			extraction.PageBlocks[pageNumber] = blocks
		}
		if tables := detectPDFTables(page, pageNumber); len(tables) > 0 {
			extraction.PageTables[pageNumber] = tables
		}
	}

	return extraction, nil
}

// mergeBlocksOnSameRow remonta uma linha visual que o poppler quebrou em varios
// blocos na folga horizontal (o primeiro pedaco virava titulo falso).
// So junta blocos de uma linha; multilinha ja e paragrafo.
func mergeBlocksOnSameRow(blocks []textBlock) []textBlock {
	if len(blocks) < 2 {
		return blocks
	}

	ordered := make([]textBlock, len(blocks))
	copy(ordered, blocks)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := ordered[i].Box, ordered[j].Box
		if left == nil || right == nil {
			return false
		}
		return left.YMin < right.YMin
	})

	// Agrupa por faixa vertical antes de ordenar por X: fragmentos da mesma
	// linha tem baselines com decimos de diferenca.
	rows := make([][]textBlock, 0, len(ordered))
	for _, block := range ordered {
		placed := false
		if len(rows) > 0 {
			row := rows[len(rows)-1]
			if sameVisualRow(&row[0], &block) {
				rows[len(rows)-1] = append(row, block)
				placed = true
			}
		}
		if !placed {
			rows = append(rows, []textBlock{block})
		}
	}

	merged := make([]textBlock, 0, len(rows))
	for _, row := range rows {
		sort.SliceStable(row, func(i, j int) bool {
			return row[i].Box.XMin < row[j].Box.XMin
		})

		head := row[0]
		for _, block := range row[1:] {
			head.Content = normalizeText(head.Content + " " + block.Content)
			head.Box.XMin = math.Min(head.Box.XMin, block.Box.XMin)
			head.Box.YMin = math.Min(head.Box.YMin, block.Box.YMin)
			head.Box.XMax = math.Max(head.Box.XMax, block.Box.XMax)
			head.Box.YMax = math.Max(head.Box.YMax, block.Box.YMax)
		}
		if len(row) > 1 {
			head.Lines = []textLine{{Text: head.Content, Box: *head.Box}}
		}
		merged = append(merged, head)
	}

	return merged
}

func sameVisualRow(left, right *textBlock) bool {
	if left.Box == nil || right.Box == nil {
		return false
	}
	if left.LineCount != 1 || right.LineCount != 1 {
		return false
	}

	overlap := math.Min(left.Box.YMax, right.Box.YMax) - math.Max(left.Box.YMin, right.Box.YMin)
	if overlap <= 0 {
		return false
	}

	shortest := math.Min(left.Box.YMax-left.Box.YMin, right.Box.YMax-right.Box.YMin)
	if shortest <= 0 {
		return false
	}

	return overlap/shortest >= 0.6
}

func textFromPageBlocks(pageBlocks map[int][]textBlock) string {
	pageNumbers := make([]int, 0, len(pageBlocks))
	for pageNumber := range pageBlocks {
		pageNumbers = append(pageNumbers, pageNumber)
	}
	sort.Ints(pageNumbers)

	parts := make([]string, 0, len(pageNumbers))
	for _, pageNumber := range pageNumbers {
		if text := textFromBlocks(pageBlocks[pageNumber]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func (nativePDFTextExtractor) Extract(data []byte) (pdfTextExtraction, error) {
	reader := bytes.NewReader(data)
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return pdfTextExtraction{}, err
	}

	extraction := pdfTextExtraction{
		PageCount:  pdfReader.NumPage(),
		PageBlocks: make(map[int][]textBlock, pdfReader.NumPage()),
	}

	for pageIndex := 1; pageIndex <= pdfReader.NumPage(); pageIndex++ {
		page := pdfReader.Page(pageIndex)
		if page.V.IsNull() || page.V.Key("Contents").Kind() == pdf.Null {
			continue
		}

		pageNumber := pageIndex
		blocks := blocksFromPDFPage(page, &pageNumber)
		if len(blocks) > 0 {
			extraction.PageBlocks[pageNumber] = blocks
		}
	}

	return extraction, nil
}

func (e *commandPDFOCRExtractor) Name() string {
	return "local_tesseract"
}

func (e *commandPDFOCRExtractor) Extract(
	ctx context.Context,
	data []byte,
	pageNumbers []int,
) (map[int][]textBlock, error) {
	if e == nil || len(pageNumbers) == 0 {
		return nil, nil
	}

	tempDir, err := os.MkdirTemp("", "thingindex-pdf-ocr-*")
	if err != nil {
		return nil, fmt.Errorf("create ocr temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	pdfPath := filepath.Join(tempDir, "document.pdf")
	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("write temporary pdf: %w", err)
	}

	results := make(map[int][]textBlock, len(pageNumbers))
	for _, pageNumber := range pageNumbers {
		if pageNumber <= 0 {
			continue
		}

		imageBase := filepath.Join(tempDir, fmt.Sprintf("page-%d", pageNumber))
		renderCmd := exec.CommandContext(
			ctx,
			e.rendererPath,
			"-f", strconv.Itoa(pageNumber),
			"-l", strconv.Itoa(pageNumber),
			"-singlefile",
			"-png",
			pdfPath,
			imageBase,
		)
		if output, err := renderCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("render pdf page %d for ocr: %w: %s", pageNumber, err, strings.TrimSpace(string(output)))
		}

		imagePath := imageBase + ".png"
		ocrArgs := []string{imagePath, "stdout", "--psm", "1"}
		if langs := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_LANGS")); langs != "" {
			ocrArgs = append(ocrArgs, "-l", langs)
		}

		ocrCmd := exec.CommandContext(ctx, e.tesseractPath, ocrArgs...)
		output, err := ocrCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("ocr pdf page %d: %w: %s", pageNumber, err, strings.TrimSpace(string(output)))
		}

		text := normalizeText(string(output))
		if text != "" {
			// tesseract entrega texto puro: os blocos saem sem Box, entao as
			// paginas cobertas por ele nao participam do grifo posicional.
			if blocks := blocksFromOCRPageText(text, pageNumber); len(blocks) > 0 {
				results[pageNumber] = blocks
			}
		}
	}

	return results, nil
}

type outlineParagraphSpan struct {
	start int
	end   int
}

func sectionTitleForKey(headings []outlineDraft, sectionKey string) string {
	if sectionKey == "" {
		return ""
	}

	for _, heading := range headings {
		if heading.SectionKey == sectionKey {
			return heading.Title
		}
	}

	return ""
}

var numberedHeadingPattern = regexp.MustCompile(`^\d+(\.\d+)*\s+`)

var pdfPageMarkerPattern = regexp.MustCompile(`^(p[aá]gina|page)\s+\d+(\s+(de|of)\s+\d+)?$`)

var pdfProcessNumberPattern = regexp.MustCompile(`\b\d{4,7}-\d{2}\.\d{4}\.\d\.\d{2}\.\d{4}\b`)

type documentHeadingArchetype string

const (
	documentHeadingArchetypeUnknown      documentHeadingArchetype = ""
	documentHeadingArchetypeLegalRoot    documentHeadingArchetype = "legal_root"
	documentHeadingArchetypeLegalSection documentHeadingArchetype = "legal_section"
	documentHeadingArchetypeAttachment   documentHeadingArchetype = "attachment"
)

var legalDocumentRootHeadingKeywords = []string{
	"sentenca",
	"acordao",
	"decisao",
	"despacho",
	"oficio",
	"certidao",
	"mandado",
	"alvara",
}

var legalSectionHeadingKeywords = []string{
	"relatorio",
	"fundamentacao",
	"dispositivo",
	"ementa",
	"voto",
}

var attachmentHeadingKeywords = []string{
	"anexo",
	"apendice",
	"planilha",
	"memoria de calculo",
	"quadro demonstrativo",
	"demonstrativo de calculo",
}

// mostlyNumeric: linha de calendario ou tabela de datas ("22 23 24 25") casa
// com o padrao de titulo numerado e viraria secao.
func mostlyNumeric(text string) bool {
	fields := strings.Fields(text)
	numeric := 0
	for _, field := range fields {
		if isNumericToken(field) {
			numeric++
		}
	}
	// Tres ou mais numeros soltos nunca sao titulo ("15 16 17 18 19 20 21 2 a
	// 6 Rematricula..."); dois numeros soltos so quando sao pelo menos metade
	// das palavras ("9 a 13 Confirmacao"). "1 DAS DISPOSICOES PRELIMINARES"
	// (1 de 4) e "2.3 Da inscricao" continuam titulos.
	return numeric >= 3 || (numeric >= 2 && numeric*2 >= len(fields))
}

func isNumericToken(field string) bool {
	if field == "" {
		return false
	}
	for _, r := range field {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func looksLikeHeading(text string) bool {
	if text == "" || strings.Contains(text, ". ") && len(text) > 120 {
		return false
	}
	if mostlyNumeric(text) {
		return false
	}
	// Data por extenso no comeco ("29 de dezembro de 2008, que institui...")
	// e citacao de lei dentro de um paragrafo, nao titulo -- e vem antes do
	// padrao de titulo numerado, que o "29 " enganaria.
	if dateStartPattern.MatchString(strings.ToLower(text)) {
		return false
	}
	// Reject markdown table rows and separators
	if strings.Count(text, "|") >= 2 {
		return false
	}
	if strings.HasPrefix(strings.TrimSpace(text), "---") {
		return false
	}
	// Reject field-like lines (e.g., "Advogado: Dr. José Santos", "Email: test@test.com")
	if strings.Contains(text, ":") {
		parts := strings.SplitN(text, ":", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			// If left side is short (likely a label) and right side has content
			if len(left) <= 20 && len(left) >= 2 && right != "" {
				return false
			}
		}
	}
	if classifyDocumentHeadingArchetype(text) != documentHeadingArchetypeUnknown {
		return true
	}
	if numberedHeadingPattern.MatchString(text) {
		return true
	}
	if strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?") || strings.HasSuffix(text, ";") || strings.HasSuffix(text, ",") {
		return false
	}
	fields := strings.Fields(text)
	// Pedaco de titulo quebrado em duas linhas: comeca por preposicao ou
	// conjuncao ("DE ENSINO", "E CONSUMO") ou termina numa ("Programação de",
	// "Tecnologia da"). "DA", "DO", "DAS", "DOS" no INICIO ficam de fora da
	// regra porque abrem capitulo em texto legal ("DAS DISPOSIÇÕES GERAIS").
	if len(fields) > 0 {
		if isPortugueseConnective(strings.ToLower(fields[0])) || isPortugueseContractionOrConnective(strings.ToLower(fields[len(fields)-1])) {
			return false
		}
	}
	if len(fields) <= 12 && text == strings.ToUpper(text) {
		return true
	}

	// Celula de tabela ("Teorica 67", "Total 205", "Subtotal 4 80 40"): curta,
	// sem numeracao de secao e com um numero solto.
	if hasStandaloneNumber(text) {
		return false
	}
	return len(strings.Fields(text)) <= 10 && isMostlyTitleCase(text)
}

func isPortugueseConnective(word string) bool {
	switch word {
	case "de", "e", "em", "na", "no", "nas", "nos", "a", "o", "ao", "aos", "para", "por", "com", "ou":
		return true
	}
	return false
}

func isPortugueseContractionOrConnective(word string) bool {
	switch word {
	case "da", "do", "das", "dos":
		return true
	}
	return isPortugueseConnective(word)
}

var dateStartPattern = regexp.MustCompile(`^\d{1,2} de [a-zç]+ de \d{4}`)

func hasStandaloneNumber(text string) bool {
	for _, field := range strings.Fields(text) {
		if isNumericToken(strings.Trim(field, "()[],;")) {
			return true
		}
	}
	return false
}

func looksLikePDFHeading(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !looksLikeHeading(trimmed) {
		return false
	}
	if isLowSignalPDFText(trimmed) || looksLikePDFFieldLine(trimmed) || looksLikePDFBoilerplate(trimmed) {
		return false
	}
	// Em PDF, uma palavra so em caixa mista ("Reitor", "Presencial",
	// "Bacharelado") e valor de campo da capa ou da identificacao, nao secao.
	// Titulos de uma palavra que importam ("REFERENCIAS", "Anexo") ja
	// passaram pela caixa alta ou pelo arquetipo.
	if len(strings.Fields(trimmed)) == 1 && trimmed != strings.ToUpper(trimmed) && classifyDocumentHeadingArchetype(trimmed) == documentHeadingArchetypeUnknown {
		return false
	}
	return true
}

func looksLikePDFFieldLine(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if numberedHeadingPattern.MatchString(trimmed) {
		return false
	}
	if pdfPageMarkerPattern.MatchString(strings.ToLower(trimmed)) {
		return true
	}
	if pdfProcessNumberPattern.MatchString(trimmed) {
		return true
	}
	if strings.HasSuffix(trimmed, ":") {
		return true
	}
	if strings.Contains(trimmed, "@") {
		return true
	}
	if strings.Contains(strings.ToLower(trimmed), "processo nº") ||
		strings.Contains(strings.ToLower(trimmed), "processo n°") ||
		strings.Contains(strings.ToLower(trimmed), "processo no ") {
		return true
	}
	return strings.Count(trimmed, ":") >= 1
}

func looksLikePDFBoilerplate(text string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(text)), " "))
	if normalized == "" {
		return false
	}

	if pdfPageMarkerPattern.MatchString(normalized) || pdfProcessNumberPattern.MatchString(text) {
		return true
	}

	if classifyDocumentHeadingArchetype(text) != documentHeadingArchetypeUnknown {
		return false
	}

	if len(strings.Fields(text)) >= 8 && text == strings.ToUpper(text) && !numberedHeadingPattern.MatchString(text) {
		return true
	}

	return containsAnySubstring(normalized,
		"tribunal de justiça",
		"tribunal de justica",
		"poder judiciário",
		"poder judiciario",
		"diário da justiça",
		"diario da justica",
		"publicação oficial",
		"publicacao oficial",
		"documento assinado digitalmente",
		"assinado digitalmente",
		"código de autenticidade",
		"codigo de autenticidade",
		"código verificador",
		"codigo verificador",
		"para conferir o original",
		"processo digital nº",
		"processo digital n°",
		"http://",
		"https://",
		"www.",
	)
}

func isLowSignalPDFText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	if numberedHeadingPattern.MatchString(trimmed) {
		remainder := strings.TrimSpace(numberedHeadingPattern.ReplaceAllString(trimmed, ""))
		return !containsLetter(remainder)
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return true
	}

	letters := 0
	digits := 0
	meaningfulWords := 0
	shortWords := 0

	for _, word := range words {
		wordLetters := 0
		wordDigits := 0
		wordRunes := 0
		for _, r := range word {
			if unicode.IsLetter(r) {
				letters++
				wordLetters++
				wordRunes++
				continue
			}
			if unicode.IsDigit(r) {
				digits++
				wordDigits++
				wordRunes++
			}
		}

		if wordRunes == 0 {
			continue
		}
		if wordRunes <= 2 {
			shortWords++
		}
		if wordLetters >= 3 || (wordLetters >= 1 && wordDigits >= 2) {
			meaningfulWords++
		}
	}

	switch {
	case letters == 0:
		return digits <= 6
	case letters < 3 && meaningfulWords == 0:
		return true
	case meaningfulWords == 0 && shortWords >= min(2, len(words)):
		return true
	case len(words) == 1 && letters <= 2:
		return true
	case len(words) <= 4 && shortWords == len(words) && meaningfulWords == 0:
		return true
	default:
		return false
	}
}

func containsLetter(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func appendIndexingMetadataPair(result *domain.IndexingResult, documentID uint, key, value string) {
	if result == nil || documentID == 0 {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return
	}

	for idx := range result.MetadataPairs {
		if result.MetadataPairs[idx].Key == key {
			result.MetadataPairs[idx].Value = value
			return
		}
	}

	result.MetadataPairs = append(result.MetadataPairs, models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: len(result.MetadataPairs),
		Key:           key,
		Value:         value,
	})
}

func shouldSuppressRepeatedPDFHeading(text string, occurrences map[string]int) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if looksLikePDFBoilerplate(trimmed) && occurrences[trimmed] >= 2 {
		return true
	}
	return occurrences[trimmed] >= 3 && len(strings.Fields(trimmed)) <= 16
}

func inferHeadingLevel(text string) int {
	switch classifyDocumentHeadingArchetype(text) {
	case documentHeadingArchetypeLegalRoot:
		return 1
	case documentHeadingArchetypeLegalSection, documentHeadingArchetypeAttachment:
		return 2
	}

	if matches := numberedHeadingPattern.FindString(text); matches != "" {
		trimmed := strings.TrimSpace(matches)
		return strings.Count(trimmed, ".") + 1
	}

	switch {
	case text == strings.ToUpper(text):
		return 1
	case len(strings.Fields(text)) <= 4:
		return 2
	default:
		return 3
	}
}

func classifyDocumentHeadingArchetype(text string) documentHeadingArchetype {
	normalized := normalizeHeadingLexicon(text)
	if normalized == "" {
		return documentHeadingArchetypeUnknown
	}

	wordCount := len(strings.Fields(normalized))
	if wordCount == 0 || wordCount > 14 {
		return documentHeadingArchetypeUnknown
	}

	candidates := []string{normalized}
	if stripped := strings.TrimSpace(numberedHeadingPattern.ReplaceAllString(normalized, "")); stripped != "" && stripped != normalized {
		candidates = append(candidates, stripped)
	}

	for _, candidate := range candidates {
		switch {
		case matchesHeadingKeywordFamily(candidate, legalDocumentRootHeadingKeywords):
			return documentHeadingArchetypeLegalRoot
		case matchesHeadingKeywordFamily(candidate, legalSectionHeadingKeywords):
			return documentHeadingArchetypeLegalSection
		case matchesHeadingKeywordFamily(candidate, attachmentHeadingKeywords):
			return documentHeadingArchetypeAttachment
		}
	}

	return documentHeadingArchetypeUnknown
}

func matchesHeadingKeywordFamily(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if value == keyword || strings.HasPrefix(value, keyword+" ") {
			return true
		}
	}
	return false
}

func normalizeHeadingLexicon(text string) string {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	lastSpace := false
	for _, r := range trimmed {
		folded := foldLatinRune(r)
		switch {
		case unicode.IsLetter(folded) || unicode.IsDigit(folded):
			builder.WriteRune(folded)
			lastSpace = false
		default:
			if !lastSpace && builder.Len() > 0 {
				builder.WriteRune(' ')
				lastSpace = true
			}
		}
	}

	return strings.TrimSpace(builder.String())
}

func foldLatinRune(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	default:
		return r
	}
}

func isMostlyTitleCase(text string) bool {
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}

	titleCaseCount := 0
	for _, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		if unicode.IsUpper(runes[0]) {
			titleCaseCount++
		}
	}

	return titleCaseCount*2 >= len(words)
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")

	paragraphs := strings.Split(text, "\n\n")
	normalizedParagraphs := make([]string, 0, len(paragraphs))
	for _, para := range paragraphs {
		lines := strings.Split(para, "\n")
		compacted := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
			if line != "" {
				compacted = append(compacted, line)
			}
		}
		normalized := strings.TrimSpace(strings.Join(compacted, " "))
		if normalized != "" {
			normalizedParagraphs = append(normalizedParagraphs, normalized)
		}
	}
	return strings.Join(normalizedParagraphs, "\n\n")
}

func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if idx := strings.Index(text, "."); idx > 0 {
		return strings.TrimSpace(text[:idx])
	}
	words := strings.Fields(text)
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, " ")
}

func slugify(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	var builder strings.Builder
	lastDash := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "section"
	}
	return slug
}

func buildOutlineNodeID(sequence int, title string) string {
	return clampIdentifierWithHash(
		fmt.Sprintf("node-%03d-%s", sequence, slugify(title)),
		maxOutlineNodeIDLength,
	)
}

func buildOutlineSlug(title string) string {
	return clampIdentifierWithHash(slugify(title), maxOutlineSlugLength)
}

func buildOutlineNodePath(parts []string) string {
	return clampIdentifierWithHash(strings.Join(parts, "/"), maxOutlineNodePathLength)
}

// clampIdentifierWithHash corta em caracteres, nao bytes: varchar(N) conta
// caracteres e cortar por byte parte "ã" ao meio (INSERT falha com UTF8 invalido).
func clampIdentifierWithHash(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 9 {
		return string(runes[:limit])
	}

	hash := shortStableHash(value)
	keep := limit - len(hash) - 1
	if keep < 1 {
		keep = 1
	}

	return string(runes[:keep]) + "-" + hash
}

func shortStableHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intPtr(value int) *int {
	return &value
}

func estimateTextTokens(content string) int {
	return len(strings.Fields(content))
}

func sortedMapKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return []string{}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortInts(values []int) {
	sort.Ints(values)
}

func containsAnySubstring(value string, substrings ...string) bool {
	for _, substring := range substrings {
		if substring != "" && strings.Contains(value, substring) {
			return true
		}
	}
	return false
}
