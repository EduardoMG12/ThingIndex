package indexing

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"thing-index/internal/models"
)

// Deteccao de tabelas em PDF a partir das caixas de palavra do pdftotext.
//
// O pipeline de PDF nunca emitia TableSection: a matriz curricular virava prosa
// embaralhada de pagina, e "qual o pre-requisito de X?" caia numa pagina do
// PPC. O experimento de geracao mostrou que o mesmo texto, lido como tabela,
// responde 90% das perguntas tabulares. A deteccao aqui e geometrica e
// deterministica (constituicao II):
//
//  1. palavras viram linhas visuais (mesmo Y) e celulas (folga horizontal);
//  2. um registro comeca numa linha com celula-ancora (numero, faixa de dias,
//     codigo de disciplina); a tabela e a sequencia de registros cujas bordas
//     esquerdas se alinham nas mesmas colunas;
//  3. linhas sem ancora sao celulas multilinha; a tabela decide por votacao
//     se elas continuam o registro anterior (celula alinhada no topo, SAGRES)
//     ou abrem o proximo (alinhada na base, tabelas do PPC);
//  4. o cabecalho sao as linhas sem ancora coladas acima do primeiro registro;
//     o titulo, a linha solta logo acima do cabecalho.
//
// ponytail: tolerancias fixas em pontos, ajustadas nos PDFs reais do demo.
// Se outro gerador quebrar, derivar as tolerancias da altura media da fonte.
const (
	pdfTableRowTolerance    = 2.5  // pt: palavras na mesma linha visual
	pdfTableMinCellGap      = 12.0 // pt: folga minima entre celulas
	pdfTableCellGapChars    = 2.5  // folga em "largura media de caractere"
	pdfTableColumnTolerance = 8.0  // pt: bordas esquerdas na mesma coluna
	pdfTableMinColumns      = 3
	pdfTableMinRecords      = 2
	pdfTableTitleMaxGap     = 30.0 // pt: titulo logo acima da tabela
	pdfTableMaxRowGapFactor = 2.5  // linhas mais afastadas que isto encerram a tabela
	pdfTableHeaderGapFactor = 1.5  // cabecalho precisa estar colado ao primeiro registro
	pdfTableMaxHeaderRows   = 3
	pdfTableHeaderLeftSlack = 40.0 // pt: rotulo de cabecalho pode comecar a esquerda dos dados
)

type pdfWord struct {
	Text                   string
	XMin, YMin, XMax, YMax float64
}

type pdfCell struct {
	Words      []pdfWord
	XMin, XMax float64
}

func (c pdfCell) text() string {
	parts := make([]string, 0, len(c.Words))
	for _, w := range c.Words {
		parts = append(parts, w.Text)
	}
	return strings.Join(parts, " ")
}

type pdfVisualRow struct {
	YMin, YMax float64
	Cells      []pdfCell
}

func (r pdfVisualRow) height() float64 { return math.Max(r.YMax-r.YMin, 1) }

// pdfAnchorCell reconhece a celula que marca o inicio de um registro: numero,
// faixa de dias ("2a6", "17, 18 e 19"), codigo de disciplina ("SIS07101",
// "SIS07OP01").
var pdfAnchorCell = regexp.MustCompile(`^(\d{1,4}([.,]\d+)?( \d{1,4}([.,]\d+)?)*|\d{1,2} ?[ae\-–] ?\d{1,2}|\d{1,2}(, ?\d{1,2})* ?e ?\d{1,2}|[A-Z]{2,}\d{2,}[A-Z0-9]*)$`)

var pdfConnectives = map[string]bool{
	"de": true, "da": true, "do": true, "das": true, "dos": true, "e": true, "em": true, "a": true, "ao": true,
	"à": true, "para": true, "com": true, "por": true, "no": true, "na": true, "nos": true, "nas": true,
	"o": true, "os": true, "um": true, "uma": true, "ou": true, "-": true, "–": true,
}

func detectPDFTables(page pdfBBoxPage, pageNumber int) []models.TableSection {
	rows := pdfVisualRows(pdfPageWords(page))
	var tables []models.TableSection
	for i := 0; i < len(rows); i++ {
		if !pdfIsDataRow(rows[i]) {
			continue
		}
		end := pdfDataRunEnd(rows, i)
		dataStart := pdfLeadingWrapStart(rows, i)
		headerStart := pdfHeaderStart(rows, dataStart)
		title := pdfTableTitle(rows, headerStart)
		if table, ok := pdfTableFromRows(rows[headerStart:dataStart], rows[dataStart:end], title, pageNumber, page); ok {
			tables = append(tables, table)
		}
		i = end - 1
	}
	return tables
}

func pdfIsDataRow(row pdfVisualRow) bool {
	return len(row.Cells) >= pdfTableMinColumns && pdfRowHasAnchor(row)
}

func pdfPageWords(page pdfBBoxPage) []pdfWord {
	var words []pdfWord
	for _, block := range page.Blocks {
		for _, line := range block.Lines {
			for _, w := range line.Words {
				text := strings.TrimSpace(w.Text)
				if text == "" || w.XMax <= w.XMin {
					continue
				}
				words = append(words, pdfWord{Text: text, XMin: w.XMin, YMin: w.YMin, XMax: w.XMax, YMax: w.YMax})
			}
		}
	}
	sort.SliceStable(words, func(a, b int) bool {
		if math.Abs(words[a].YMin-words[b].YMin) > pdfTableRowTolerance {
			return words[a].YMin < words[b].YMin
		}
		return words[a].XMin < words[b].XMin
	})
	return words
}

// pdfVisualRows agrupa palavras por linha visual e cada linha em celulas.
func pdfVisualRows(words []pdfWord) []pdfVisualRow {
	var rows []pdfVisualRow
	var current []pdfWord
	flush := func() {
		if len(current) == 0 {
			return
		}
		sort.SliceStable(current, func(a, b int) bool { return current[a].XMin < current[b].XMin })
		rows = append(rows, pdfRowFromWords(current))
		current = nil
	}
	for _, w := range words {
		if len(current) > 0 && math.Abs(w.YMin-current[0].YMin) > pdfTableRowTolerance {
			flush()
		}
		current = append(current, w)
	}
	flush()
	return rows
}

func pdfRowFromWords(words []pdfWord) pdfVisualRow {
	var totalWidth float64
	var totalChars int
	row := pdfVisualRow{YMin: words[0].YMin, YMax: words[0].YMax}
	for _, w := range words {
		totalWidth += w.XMax - w.XMin
		totalChars += len([]rune(w.Text))
		row.YMin = math.Min(row.YMin, w.YMin)
		row.YMax = math.Max(row.YMax, w.YMax)
	}
	gap := pdfTableMinCellGap
	if totalChars > 0 {
		gap = math.Max(gap, pdfTableCellGapChars*totalWidth/float64(totalChars))
	}
	for _, w := range words {
		if n := len(row.Cells); n > 0 && w.XMin-row.Cells[n-1].XMax <= gap {
			row.Cells[n-1].Words = append(row.Cells[n-1].Words, w)
			row.Cells[n-1].XMax = math.Max(row.Cells[n-1].XMax, w.XMax)
			continue
		}
		row.Cells = append(row.Cells, pdfCell{Words: []pdfWord{w}, XMin: w.XMin, XMax: w.XMax})
	}
	return row
}

func pdfRowHasAnchor(row pdfVisualRow) bool {
	for _, c := range row.Cells {
		if pdfAnchorCell.MatchString(strings.TrimSpace(c.text())) {
			return true
		}
	}
	return false
}

// pdfColumnsFromRows agrupa as bordas esquerdas das celulas dos registros em
// colunas (tolerancia fixa), em ordem crescente.
func pdfColumnsFromRows(rows []pdfVisualRow) []float64 {
	var lefts []float64
	for _, row := range rows {
		if !pdfRowHasAnchor(row) {
			continue
		}
		for _, c := range row.Cells {
			lefts = append(lefts, c.XMin)
		}
	}
	sort.Float64s(lefts)
	var columns []float64
	for _, x := range lefts {
		if len(columns) > 0 && x-columns[len(columns)-1] <= pdfTableColumnTolerance {
			continue
		}
		columns = append(columns, x)
	}
	return columns
}

// pdfColumnIndex mapeia uma borda esquerda a coluna cujo intervalo a contem.
func pdfColumnIndex(columns []float64, x float64) int {
	idx := 0
	for i, left := range columns {
		if x+pdfTableColumnTolerance >= left {
			idx = i
		}
	}
	return idx
}

func pdfRowAlignsWithColumns(row pdfVisualRow, columns []float64) bool {
	aligned := 0
	for _, c := range row.Cells {
		for _, left := range columns {
			if math.Abs(c.XMin-left) <= pdfTableColumnTolerance {
				aligned++
				break
			}
		}
	}
	return aligned*2 >= len(row.Cells)
}

func pdfAnchoredRows(rows []pdfVisualRow) int {
	n := 0
	for _, row := range rows {
		if pdfRowHasAnchor(row) {
			n++
		}
	}
	return n
}

func pdfRowInsideExtent(row pdfVisualRow, left, right float64) bool {
	for _, c := range row.Cells {
		if c.XMin < left-pdfTableColumnTolerance || c.XMin > right {
			return false
		}
	}
	return true
}

func pdfRunExtent(rows []pdfVisualRow) (float64, float64) {
	left, right := math.Inf(1), 0.0
	for _, row := range rows {
		if !pdfRowHasAnchor(row) {
			continue
		}
		for _, c := range row.Cells {
			left = math.Min(left, c.XMin)
			right = math.Max(right, c.XMax)
		}
	}
	return left, right
}

// pdfDataRunEnd devolve o fim (exclusivo) da sequencia de linhas da tabela que
// comeca no registro start: registros alinhados as colunas e linhas de
// continuacao dentro da largura da tabela, sem salto vertical grande.
func pdfDataRunEnd(rows []pdfVisualRow, start int) int {
	end := start + 1
	for end < len(rows) {
		prev, row := rows[end-1], rows[end]
		if row.YMin-prev.YMax > pdfTableMaxRowGapFactor*prev.height() {
			break
		}
		run := rows[start:end]
		columns := pdfColumnsFromRows(run)
		left, _ := pdfRunExtent(run)
		if pdfRowHasAnchor(row) {
			// As duas primeiras linhas com ancora definem as colunas juntas: a
			// grade do calendario abre o mes com uma semana de 5 dias que nao
			// alinha sozinha com a linha cheia seguinte.
			if len(row.Cells) < 2 || (pdfAnchoredRows(run) >= 2 && !pdfRowAlignsWithColumns(row, columns)) {
				break
			}
		} else if len(row.Cells) > len(columns) || !pdfRowInsideExtent(row, left-pdfTableHeaderLeftSlack, math.Inf(1)) {
			// So o lado esquerdo e checado: a descricao longa de uma celula
			// passa da borda direita dos registros curtos.
			break
		}
		end++
	}
	// Linhas sem ancora no fim so pertencem a tabela se alguem as reivindicar;
	// uma linha de prosa colada embaixo nao e celula.
	for end > start+1 && !pdfRowHasAnchor(rows[end-1]) && len(rows[end-1].Cells) == 1 && !pdfEndsWithConnective(rows[end-2]) {
		end--
	}
	return end
}

func pdfRowsGlued(above, below pdfVisualRow) bool {
	return below.YMin-above.YMax <= pdfTableHeaderGapFactor*above.height()
}

// pdfLeadingWrapStart recua sobre as linhas coladas acima do primeiro registro
// que pertencem a ele: o nome quebrado de uma celula alinhada na base
// ("Arquitetura e" / "Organização de" / "Computadores SIS05201") e rotulos
// soltos de uma celula ("2º"). Uma linha com varias celulas e cabecalho, nao
// continuacao.
func pdfLeadingWrapStart(rows []pdfVisualRow, first int) int {
	start := first
	left, _ := pdfRunExtent(rows[first:])
	for start > 0 {
		cand := rows[start-1]
		if pdfRowHasAnchor(cand) || !pdfRowsGlued(cand, rows[start]) || !pdfRowInsideExtent(cand, left-pdfTableHeaderLeftSlack, math.Inf(1)) {
			break
		}
		// Linha com varias celulas so e continuacao se ela mesma termina em
		// conectivo; a minuscula do registro abaixo se explica pela linha de
		// uma celula que esta entre os dois.
		if len(cand.Cells) != 1 && !pdfEndsWithConnective(cand) {
			break
		}
		start--
	}
	return start
}

// pdfHeaderStart recua sobre as linhas sem ancora, com duas ou mais celulas,
// coladas ao primeiro registro (ou as suas linhas de continuacao).
func pdfHeaderStart(rows []pdfVisualRow, dataStart int) int {
	start := dataStart
	left, right := pdfRunExtent(rows[dataStart:])
	for start > 0 && dataStart-start < pdfTableMaxHeaderRows {
		cand := rows[start-1]
		if pdfRowHasAnchor(cand) || len(cand.Cells) < 2 || !pdfRowsGlued(cand, rows[start]) || pdfIsTitleRow(cand) {
			break
		}
		// Cabecalho pode comecar um pouco a esquerda dos dados ("Periodo") e
		// rotular colunas vazias a direita; so o lado esquerdo e checado.
		if !pdfRowInsideExtent(cand, left-pdfTableHeaderLeftSlack, math.Inf(1)) {
			break
		}
		start--
	}
	_ = right
	return start
}

// pdfIsTitleRow: uma celula so, ou o mesmo texto repetido ("Janeiro 2026" nas
// duas metades do calendario).
func pdfIsTitleRow(row pdfVisualRow) bool {
	if len(row.Cells) == 0 || pdfRowHasAnchor(row) {
		return false
	}
	first := strings.TrimSpace(row.Cells[0].text())
	for _, c := range row.Cells[1:] {
		if strings.TrimSpace(c.text()) != first {
			return false
		}
	}
	return true
}

func pdfTableTitle(rows []pdfVisualRow, headerStart int) string {
	if headerStart == 0 {
		return ""
	}
	prev := rows[headerStart-1]
	if !pdfIsTitleRow(prev) || rows[headerStart].YMin-prev.YMax > pdfTableTitleMaxGap {
		return ""
	}
	return strings.TrimSpace(prev.Cells[0].text())
}

func pdfEndsWithConnective(row pdfVisualRow) bool {
	for _, c := range row.Cells {
		words := c.Words
		if len(words) == 0 || pdfAnchorCell.MatchString(c.text()) {
			continue
		}
		if pdfConnectives[strings.ToLower(words[len(words)-1].Text)] {
			return true
		}
	}
	return false
}

func pdfStartsLowercase(row pdfVisualRow) bool {
	for _, c := range row.Cells {
		text := strings.TrimSpace(c.text())
		if text == "" || pdfAnchorCell.MatchString(text) {
			continue
		}
		r := []rune(text)[0]
		return unicode.IsLower(r)
	}
	return false
}

// pdfWrapsForward diz se a linha sem ancora `wrapped` e o comeco do registro
// `next` (celula alinhada na base): termina em conectivo ou o registro
// continua em minuscula.
func pdfWrapsForward(wrapped, next pdfVisualRow) bool {
	return pdfEndsWithConnective(wrapped) || pdfStartsLowercase(next)
}

// pdfTableFromRows monta cabecalho e registros. As linhas sem ancora entre
// registros sao celulas multilinha; a direcao (continua o anterior ou abre o
// proximo) e decidida por votacao para a tabela inteira, porque cada gerador
// alinha as celulas de um jeito so.
func pdfTableFromRows(headerRows, dataRows []pdfVisualRow, title string, pageNumber int, page pdfBBoxPage) (models.TableSection, bool) {
	columns := pdfColumnsFromRows(dataRows)
	if len(columns) < pdfTableMinColumns {
		return models.TableSection{}, false
	}

	forward, backward := 0, 0
	var lastAnchored *pdfVisualRow
	for i := range dataRows {
		row := dataRows[i]
		if pdfRowHasAnchor(row) {
			lastAnchored = &dataRows[i]
			continue
		}
		if i+1 < len(dataRows) && pdfRowHasAnchor(dataRows[i+1]) && pdfWrapsForward(row, dataRows[i+1]) {
			forward++
		}
		if lastAnchored != nil && pdfEndsWithConnective(*lastAnchored) {
			backward++
		}
	}
	wrapsForward := forward > backward

	place := func(target []string, row pdfVisualRow) {
		for _, c := range row.Cells {
			col := pdfColumnIndex(columns, c.XMin)
			for k, w := range c.Words {
				// Palavra que comeca exatamente numa borda de coluna mais a
				// direita abre celula nova, mesmo sem folga: "Lógica de Natureza"
				// e o nome da disciplina encostado na categoria.
				if next := pdfColumnIndex(columns, w.XMin); k > 0 && next > col && math.Abs(w.XMin-columns[next]) <= pdfTableColumnTolerance {
					col = next
				}
				if target[col] == "" {
					target[col] = w.Text
				} else {
					target[col] += " " + w.Text
				}
			}
		}
	}

	headers := make([]string, len(columns))
	for _, row := range headerRows {
		place(headers, row)
	}
	var records [][]string
	var pending []pdfVisualRow
	for _, row := range dataRows {
		if !pdfRowHasAnchor(row) {
			pending = append(pending, row)
			continue
		}
		record := make([]string, len(columns))
		if wrapsForward || len(records) == 0 {
			for _, p := range pending {
				place(record, p)
			}
		} else {
			for _, p := range pending {
				place(records[len(records)-1], p)
			}
		}
		pending = nil
		place(record, row)
		records = append(records, record)
	}
	for _, p := range pending {
		place(records[len(records)-1], p)
	}
	if len(records) < pdfTableMinRecords {
		return models.TableSection{}, false
	}
	for c := range headers {
		if headers[c] == "" {
			headers[c] = "coluna " + strconv.Itoa(c+1)
		}
	}
	left, right := pdfRunExtent(dataRows)
	top := dataRows[0].YMin
	if len(headerRows) > 0 {
		top = headerRows[0].YMin
		// O cabecalho rotula colunas que podem estar vazias nesta pagina
		// ("Trilha de optativas"); a caixa cobre ate ele, senao o rotulo
		// sobra fora da tabela e vira titulo de secao.
		for _, row := range headerRows {
			for _, c := range row.Cells {
				right = math.Max(right, c.XMax)
			}
		}
	}
	pageRef := pageNumber
	return models.TableSection{
		Title:      title,
		PageNumber: &pageRef,
		Headers:    headers,
		Rows:       records,
		Metadata: map[string]any{
			"source": "pdf-bbox",
			"box": textBox{
				XMin: left, YMin: top, XMax: right, YMax: dataRows[len(dataRows)-1].YMax,
				PageWidth: page.Width, PageHeight: page.Height,
			},
		},
	}, true
}

// pdfTableBoxes devolve as caixas das tabelas de uma pagina.
func pdfTableBoxes(tables []models.TableSection) []textBox {
	boxes := make([]textBox, 0, len(tables))
	for _, table := range tables {
		if box, ok := table.Metadata["box"].(textBox); ok {
			boxes = append(boxes, box)
		}
	}
	return boxes
}

// blockInsideTable diz se o centro do bloco cai dentro de alguma tabela da
// pagina: celula de tabela nao e titulo de secao, por mais que pareca
// ("Atividade Pedagógica", "67 SIS07304 | SIS07405").
func blockInsideTable(block textBlock, boxes []textBox) bool {
	if block.Box == nil {
		return false
	}
	cx := (block.Box.XMin + block.Box.XMax) / 2
	cy := (block.Box.YMin + block.Box.YMax) / 2
	for _, b := range boxes {
		// Folga maior em cima: o cabecalho de duas linhas do SAGRES comeca
		// alguns pontos acima da primeira linha visual que o detector guardou.
		if cx >= b.XMin-pdfTableColumnTolerance && cx <= b.XMax+pdfTableColumnTolerance && cy >= b.YMin-pdfTableMinCellGap && cy <= b.YMax+pdfTableRowTolerance {
			return true
		}
	}
	return false
}

// flattenPageTables devolve as tabelas na ordem das paginas com SequenceOrder
// unico por documento.
func flattenPageTables(documentID uint, pageTables map[int][]models.TableSection, pageCount int) []models.TableSection {
	var out []models.TableSection
	for page := 1; page <= pageCount; page++ {
		for _, table := range pageTables[page] {
			table.DocumentID = documentID
			table.SequenceOrder = len(out)
			out = append(out, table)
		}
	}
	return out
}

// RenderTableRow devolve "cabecalho: celula; ..." so com as celulas cheias: o
// texto que vira unidade de recuperacao e fonte de chat.
func RenderTableRow(headers []string, row []string) string {
	parts := make([]string, 0, len(row))
	for i, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		header := ""
		if i < len(headers) {
			header = strings.TrimSpace(headers[i])
		}
		// "coluna 7" e cabecalho sintetico (tabela sem linha de rotulo):
		// nao diz nada ao leitor nem ao modelo.
		if strings.HasPrefix(header, "coluna ") {
			header = ""
		}
		if header != "" {
			parts = append(parts, header+": "+cell)
		} else {
			parts = append(parts, cell)
		}
	}
	return strings.Join(parts, "; ")
}
