package indexing

import "testing"

// Saida real de `pdftotext -bbox-layout`, reduzida. O parser precisa aguentar o
// XHTML com DOCTYPE e entidades, que o decoder estrito recusa.
const bboxLayoutFixture = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><html xmlns="http://www.w3.org/1999/xhtml">
<head>
</head>
<body>
<doc>
  <page width="612.000000" height="792.000000">
    <flow>
      <block xMin="72.000000" yMin="74.768000" xMax="249.408000" yMax="96.968000">
        <line xMin="72.000000" yMin="74.768000" xMax="249.408000" yMax="96.968000">
          <word xMin="72.000000" yMin="74.768000" xMax="146.712000" yMax="96.968000">Vara</word>
          <word xMin="153.384000" yMin="74.768000" xMax="249.408000" yMax="96.968000">Civel</word>
        </line>
        <line xMin="72.000000" yMin="100.000000" xMax="200.000000" yMax="120.000000">
          <word xMin="72.000000" yMin="100.000000" xMax="200.000000" yMax="120.000000">Comarca</word>
        </line>
      </block>
    </flow>
  </page>
  <page width="595.000000" height="842.000000">
    <flow>
      <block xMin="50.000000" yMin="60.000000" xMax="300.000000" yMax="80.000000">
        <line xMin="50.000000" yMin="60.000000" xMax="300.000000" yMax="80.000000">
          <word xMin="50.000000" yMin="60.000000" xMax="300.000000" yMax="80.000000">A parte autora requer a citacao da parte re para responder aos termos da presente acao, sob pena de revelia e confissao quanto a materia de fato.</word>
        </line>
      </block>
    </flow>
  </page>
  <page width="612.000000" height="792.000000">
    <flow>
      <block xMin="10.000000" yMin="20.000000" xMax="30.000000" yMax="40.000000">
        <line xMin="10.000000" yMin="20.000000" xMax="30.000000" yMax="40.000000">
          <word xMin="10.000000" yMin="20.000000" xMax="30.000000" yMax="40.000000">Anexo</word>
        </line>
      </block>
    </flow>
  </page>
</doc>
</body>
</html>
`

func TestParsePDFBBoxLayoutSeparatesPages(t *testing.T) {
	extraction, err := parsePDFBBoxLayout([]byte(bboxLayoutFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Regressao: o parser antigo dividia paginas por linha de tracos, que
	// pdftotext nunca emite (ele usa form feed). Todo bloco caia na pagina 1 e
	// PageCount vinha 1, o que tambem limitava o fallback de OCR a primeira pagina.
	if extraction.PageCount != 3 {
		t.Fatalf("PageCount = %d, quero 3", extraction.PageCount)
	}
	if len(extraction.PageBlocks) != 3 {
		t.Fatalf("PageBlocks = %d paginas, quero 3", len(extraction.PageBlocks))
	}

	for pageNumber := 1; pageNumber <= 3; pageNumber++ {
		blocks := extraction.PageBlocks[pageNumber]
		if len(blocks) != 1 {
			t.Fatalf("pagina %d: %d blocos, quero 1", pageNumber, len(blocks))
		}
		if blocks[0].PageNumber == nil || *blocks[0].PageNumber != pageNumber {
			t.Fatalf("pagina %d: PageNumber do bloco errado", pageNumber)
		}
	}
}

func TestParsePDFBBoxLayoutKeepsBoxAndText(t *testing.T) {
	extraction, err := parsePDFBBoxLayout([]byte(bboxLayoutFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	block := extraction.PageBlocks[1][0]
	// normalizeText compacta as linhas de um mesmo bloco com espaco; o bbox do
	// bloco continua cobrindo as duas linhas.
	if block.Content != "Vara Civel Comarca" {
		t.Fatalf("Content = %q, quero %q", block.Content, "Vara Civel Comarca")
	}

	if block.Box == nil {
		t.Fatal("Box nil: a ancora posicional foi descartada")
	}
	if block.Box.XMin != 72 || block.Box.YMin != 74.768 || block.Box.XMax != 249.408 || block.Box.YMax != 96.968 {
		t.Fatalf("Box = %+v", *block.Box)
	}
	if block.Box.PageWidth != 612 || block.Box.PageHeight != 792 {
		t.Fatalf("dimensoes da pagina = %vx%v, quero 612x792", block.Box.PageWidth, block.Box.PageHeight)
	}

	// Paginas podem ter tamanhos diferentes no mesmo documento; normalizar com a
	// dimensao errada desloca o grifo.
	if page2 := extraction.PageBlocks[2][0]; page2.Box.PageWidth != 595 || page2.Box.PageHeight != 842 {
		t.Fatalf("pagina 2 = %vx%v, quero 595x842", page2.Box.PageWidth, page2.Box.PageHeight)
	}
}

// O poppler quebra uma linha visual em varios blocos quando encontra folga
// horizontal grande. Sem remontar, o primeiro fragmento ("Critical") virava
// heading e o corpo do texto virava tres paragrafos soltos.
func TestMergeBlocksOnSameRow(t *testing.T) {
	t.Parallel()

	page := &textBox{PageWidth: 612, PageHeight: 792}
	row := func(content string, xMin, xMax, yMin, yMax float64, lines int) textBlock {
		box := *page
		box.XMin, box.XMax, box.YMin, box.YMax = xMin, xMax, yMin, yMax
		return textBlock{Content: content, LineCount: lines, Box: &box}
	}

	merged := mergeBlocksOnSameRow([]textBlock{
		row("region and complete an annual", 300, 470, 100, 118, 1),
		row("Critical", 72, 130, 100, 118, 1),
		row("services run in an alternate", 140, 290, 101, 119, 1),
		row("Audit contacts remain available.", 72, 400, 140, 158, 1),
	})

	if len(merged) != 2 {
		var got []string
		for _, block := range merged {
			got = append(got, block.Content)
		}
		t.Fatalf("blocos = %d, quero 2: %q", len(merged), got)
	}
	if merged[0].Content != "Critical services run in an alternate region and complete an annual" {
		t.Fatalf("linha remontada = %q", merged[0].Content)
	}
	// A caixa resultante precisa cobrir a linha inteira, senao o grifo cobre so
	// o primeiro fragmento.
	if merged[0].Box.XMin != 72 || merged[0].Box.XMax != 470 {
		t.Fatalf("caixa da linha = %+v", *merged[0].Box)
	}
	if merged[1].Content != "Audit contacts remain available." {
		t.Fatalf("linha seguinte foi fundida indevidamente: %q", merged[1].Content)
	}
}

// Bloco multilinha ja e um paragrafo agrupado pelo poppler: fundi-lo com um
// vizinho misturaria colunas distintas.
func TestMergeBlocksOnSameRowKeepsMultilineBlocks(t *testing.T) {
	t.Parallel()

	box := func(xMin, xMax float64) *textBox {
		return &textBox{XMin: xMin, XMax: xMax, YMin: 100, YMax: 160, PageWidth: 612, PageHeight: 792}
	}

	merged := mergeBlocksOnSameRow([]textBlock{
		{Content: "SUPPLIER MASTER ORDER FORM", LineCount: 4, Box: box(72, 260)},
		{Content: "coluna vizinha", LineCount: 1, Box: box(300, 460)},
	})

	if len(merged) != 2 {
		t.Fatalf("blocos = %d, quero 2 (multilinha nao pode fundir)", len(merged))
	}
}

func TestBuildIndexingResultPersistsBox(t *testing.T) {
	extraction, err := parsePDFBBoxLayout([]byte(bboxLayoutFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	headings, paragraphs := buildPDFStructureFromPageBlocks(extraction.PageBlocks, extraction.PageCount)
	result := buildIndexingResult(1, headings, paragraphs, nil)

	anchored := 0
	for _, node := range result.ParagraphNodes {
		if box, ok := node.Metadata["box"].(*textBox); ok && box.XMax > box.XMin {
			anchored++
		}
	}
	if anchored == 0 {
		t.Fatal("nenhum ParagraphNode chegou com box no metadata")
	}
}
