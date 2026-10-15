package indexing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"thing-index/internal/models"
)

const diretorioDoDemo = "../../../examples/faculdade"

func tabelasDaPagina(t *testing.T, arquivo string, pagina int) []tabelaLida {
	t.Helper()
	exigirFerramentasDePDF(t)
	caminho := filepath.Join(diretorioDoDemo, arquivo)
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Skipf("PDF do demo ausente: %v", err)
	}
	extraction, err := newCommandPDFTextExtractor().Extract(dados)
	if err != nil {
		t.Fatal(err)
	}
	var saida []tabelaLida
	for _, tabela := range extraction.PageTables[pagina] {
		lida := tabelaLida{titulo: tabela.Title, cabecalho: tabela.Headers}
		for _, linha := range tabela.Rows {
			lida.linhas = append(lida.linhas, RenderTableRow(tabela.Headers, linha))
		}
		saida = append(saida, lida)
	}
	return saida
}

type tabelaLida struct {
	titulo    string
	cabecalho []string
	linhas    []string
}

func linhaContendo(tabelas []tabelaLida, trechos ...string) (string, bool) {
	for _, tabela := range tabelas {
		for _, linha := range tabela.linhas {
			ok := true
			for _, trecho := range trechos {
				if !strings.Contains(linha, trecho) {
					ok = false
					break
				}
			}
			if ok {
				return linha, true
			}
		}
	}
	return "", false
}

// A matriz curricular do SAGRES: celula "Disciplina" quebrada em duas linhas,
// pre-requisito numa coluna a direita. A linha de Estrutura de Dados I precisa
// sair inteira, com o codigo do pre-requisito e o cabecalho colado.
func TestMatrizCurricularViraTabelaComPreRequisitoNaLinha(t *testing.T) {
	tabelas := tabelasDaPagina(t, "matriz-curricular-si-palmas-2023.pdf", 1)
	if len(tabelas) == 0 {
		t.Fatal("nenhuma tabela detectada na pagina 1 da matriz")
	}
	linha, ok := linhaContendo(tabelas, "SIS07302", "SIS07206")
	if !ok {
		for _, tb := range tabelas {
			t.Logf("cabecalho: %v", tb.cabecalho)
			for _, l := range tb.linhas {
				t.Logf("  %s", l)
			}
		}
		t.Fatal("a linha de Estrutura de Dados I nao saiu com o pre-requisito SIS07206")
	}
	if !strings.Contains(strings.ToLower(linha), "estrutura de dados i") {
		t.Fatalf("nome da disciplina nao foi remontado das duas linhas: %s", linha)
	}
	if !strings.Contains(strings.ToLower(linha), "requisito") {
		t.Fatalf("cabecalho do pre-requisito nao foi colado na celula: %s", linha)
	}
	if _, ok := linhaContendo(tabelas, "SIS07405", "SIS07303", "SIS07304"); !ok {
		t.Fatal("Programacao de Computadores III precisa dos dois pre-requisitos na mesma linha")
	}
}

func TestCalendarioViraTabelaComDiaEDescricao(t *testing.T) {
	tabelas := tabelasDaPagina(t, "calendario-academico-palmas-2026.pdf", 1)
	linha, ok := linhaContendo(tabelas, "Rematrícula dos estudantes veteranos")
	if !ok {
		t.Fatal("a linha da rematricula nao saiu como linha de tabela")
	}
	if !strings.Contains(linha, "2a6") && !strings.Contains(linha, "2 a 6") {
		t.Fatalf("a faixa de dias nao ficou na mesma linha: %s", linha)
	}
	titulo := false
	for _, tb := range tabelas {
		if strings.Contains(tb.titulo, "Fevereiro") {
			titulo = true
		}
	}
	if !titulo {
		t.Fatal("o mes ('Fevereiro 2026', repetido nas duas metades da pagina) precisa virar titulo da tabela: sem ele a linha nao diz de que mes e")
	}
}

func TestTabela4DoPPCViraTabela(t *testing.T) {
	tabelas := tabelasDaPagina(t, "ppc-sistemas-de-informacao-palmas-2019.pdf", 56)
	if _, ok := linhaContendo(tabelas, "SIS05101", "120"); !ok {
		for _, tb := range tabelas {
			t.Logf("cabecalho: %v", tb.cabecalho)
			for _, l := range tb.linhas {
				t.Logf("  %s", l)
			}
		}
		t.Fatal("a linha de Algoritmos e Logica (SIS05101, 120 h/a) nao saiu da Tabela 4")
	}
}

func TestPaginaDeProsaNaoGeraTabela(t *testing.T) {
	tabelas := tabelasDaPagina(t, "ppc-sistemas-de-informacao-palmas-2019.pdf", 31)
	if len(tabelas) != 0 {
		t.Fatalf("pagina 31 e prosa (objetivos) e gerou %d tabela(s): %+v", len(tabelas), tabelas)
	}
}

// Celula de tabela nao e secao ("67 SIS07304 | SIS07405", "Atividade Pedagógica").
func TestCelulaDeTabelaNaoViraSecao(t *testing.T) {
	exigirFerramentasDePDF(t)
	dados, err := os.ReadFile(filepath.Join(diretorioDoDemo, "matriz-curricular-si-palmas-2023.pdf"))
	if err != nil {
		t.Skip(err)
	}
	result, err := NewDeterministicIndexer().indexPDF(context.Background(), 1, dados)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TableSections) < 3 {
		t.Fatalf("esperava as tabelas da matriz, veio %d", len(result.TableSections))
	}
	for _, node := range result.OutlineNodes {
		title := strings.ToLower(node.Title)
		if strings.Contains(title, "sis07") || strings.Contains(title, "atividade pedag") || strings.Contains(title, "hrs coreq") || strings.Contains(title, "trilha de") {
			t.Fatalf("celula de tabela virou secao: %q", node.Title)
		}
	}
	if len(result.OutlineNodes) > 3 {
		titles := make([]string, 0, len(result.OutlineNodes))
		for _, n := range result.OutlineNodes {
			titles = append(titles, n.Title)
		}
		t.Fatalf("a matriz deveria ter poucas secoes (cabecalho do SAGRES), veio %d: %v", len(result.OutlineNodes), titles)
	}
}

// O texto da pagina inclui os titulos (rotulos curtos promovidos a titulo).
func TestTextoDaPaginaIncluiOsTitulos(t *testing.T) {
	exigirFerramentasDePDF(t)
	dados, err := os.ReadFile(filepath.Join(diretorioDoDemo, "ppc-tecnico-informatica-palmas.pdf"))
	if err != nil {
		t.Skip(err)
	}
	result, err := NewDeterministicIndexer().indexPDF(context.Background(), 1, dados)
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range result.DocumentPages {
		if page.PageNumber == 7 {
			for _, trecho := range []string{"Eixo Tecnológico", "Informação e Comunicação", "Forma de Oferta", "Integrado"} {
				if !strings.Contains(page.Content, trecho) {
					t.Fatalf("pagina 7 sem %q:\n%s", trecho, page.Content)
				}
			}
			return
		}
	}
	t.Fatal("pagina 7 nao veio")
}

func TestCabecalhoSinteticoNaoApareceNaLinha(t *testing.T) {
	got := RenderTableRow([]string{"coluna 1", "Disciplina", "coluna 3"}, []string{"Web", "SIS05105", "33"})
	if got != "Web; Disciplina: SIS05105; 33" {
		t.Fatalf("cabecalho sintetico vazou: %q", got)
	}
}

// Paragrafo com mais de uma linha guarda a caixa de cada linha, dentro da
// caixa do paragrafo, para o grifo fino do chat.
func TestParagrafoGuardaAsLinhasComCaixa(t *testing.T) {
	exigirFerramentasDePDF(t)
	dados, err := os.ReadFile(filepath.Join(diretorioDoDemo, "ppc-sistemas-de-informacao-palmas-2019.pdf"))
	if err != nil {
		t.Skip(err)
	}
	result, err := NewDeterministicIndexer().indexPDF(context.Background(), 1, dados)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range result.ParagraphNodes {
		lines, ok := p.Metadata["lines"].([]map[string]any)
		if !ok || len(lines) < 3 {
			continue
		}
		box := p.Metadata["box"].(*textBox)
		for _, line := range lines {
			b := line["b"].([]float64)
			if b[0] < box.XMin-1 || b[2] > box.XMax+1 || b[1] < box.YMin-1 || b[3] > box.YMax+1 {
				t.Fatalf("linha fora da caixa do paragrafo: %v vs %+v", b, box)
			}
			if line["t"].(string) == "" {
				t.Fatal("linha sem texto")
			}
		}
		return
	}
	t.Fatal("nenhum paragrafo com 3+ linhas trouxe metadata.lines")
}

// Num PPC com sumario numerado, capa e fragmento em caixa alta nao sao
// capitulos: ficam no nivel 3, e os capitulos numerados no nivel 1.
func TestSumarioNumeradoRebaixaCapaEFragmentos(t *testing.T) {
	exigirFerramentasDePDF(t)
	dados, err := os.ReadFile(filepath.Join(diretorioDoDemo, "ppc-sistemas-de-informacao-palmas-2019.pdf"))
	if err != nil {
		t.Skip(err)
	}
	result, err := NewDeterministicIndexer().indexPDF(context.Background(), 1, dados)
	if err != nil {
		t.Fatal(err)
	}
	nivel1 := 0
	temApresentacao := false
	for _, node := range result.OutlineNodes {
		if node.Level != 1 {
			continue
		}
		nivel1++
		title := strings.ToUpper(node.Title)
		if strings.Contains(title, "MINISTÉRIO DA EDUCAÇÃO") || title == "EDUCAÇÃO" || strings.Contains(title, "40 VAGAS") {
			t.Fatalf("capa ou fragmento virou capitulo: %q", node.Title)
		}
		if strings.Contains(title, "APRESENTAÇÃO DO PROJETO") {
			temApresentacao = true
		}
	}
	if !temApresentacao {
		t.Fatal("'1. APRESENTAÇÃO DO PROJETO' deveria ser capitulo de nivel 1")
	}
	if nivel1 > 30 {
		t.Fatalf("PPC com %d secoes de nivel 1; capitulos de verdade sao uma dezena", nivel1)
	}
}

func TestTabelaContinuadaHerdaOCabecalhoDaPaginaAnterior(t *testing.T) {
	p1, p2, p3 := 1, 2, 3
	tables := map[int][]models.TableSection{
		1: {{PageNumber: &p1, Title: "Matriz", Headers: []string{"Disciplina", "Hrs"}, Rows: [][]string{{"A", "1"}}}},
		2: {{PageNumber: &p2, Headers: []string{"coluna 1", "coluna 2"}, Rows: [][]string{{"B", "2"}}}},
		3: {{PageNumber: &p3, Headers: []string{"coluna 1", "coluna 2", "coluna 3"}, Rows: [][]string{{"C", "3", "x"}}}},
	}
	out := flattenPageTables(7, tables, 3)
	if out[1].Headers[0] != "Disciplina" || out[1].Title != "Matriz" || out[1].Metadata["headers_inherited_from_page"] != 1 {
		t.Fatalf("a tabela da p. 2 deveria herdar o cabecalho da p. 1: %+v", out[1])
	}
	if out[2].Headers[0] != "coluna 1" {
		t.Fatalf("colunas diferentes nao herdam: %+v", out[2])
	}
}

func TestTituloDeTabelaRecusaFragmento(t *testing.T) {
	casos := map[string]string{"ARTICULA": "", "NÍVEL DE": "", "DE ENSINO": "", "Janeiro 2026": "Janeiro 2026", "Tabela 4 - Periodização": "Tabela 4 - Periodização", "Optativas": ""}
	for in, want := range casos {
		if got := cleanTableTitle(in); got != want {
			t.Errorf("%q -> %q, esperava %q", in, got, want)
		}
	}
}

func TestPaginaDeSumarioComBlocoColado(t *testing.T) {
	bloco := textBlock{Content: "1 APRESENTAÇÃO DO PROJETO 7 1.1 IDENTIFICAÇÃO 7 1.2 JUSTIFICATIVA 9 2 OBJETIVOS 12 3 PERFIL 15 4 ORGANIZAÇÃO 20",
		Lines: []textLine{{Text: "1 APRESENTAÇÃO DO PROJETO 7"}, {Text: "1.1 IDENTIFICAÇÃO 7"}, {Text: "1.2 JUSTIFICATIVA 9"}, {Text: "2 OBJETIVOS 12"}, {Text: "3 PERFIL 15"}, {Text: "4 ORGANIZAÇÃO 20"}}}
	if !isPDFTableOfContentsPage([]textBlock{bloco}) {
		t.Fatal("seis entradas de sumario coladas num bloco deveriam marcar a pagina como sumario")
	}
}

func TestPaginaDeSumarioEmDuasColunas(t *testing.T) {
	titulos := textBlock{Content: "x", Lines: []textLine{{Text: "1 APRESENTAÇÃO DO PROJETO"}, {Text: "1.1 IDENTIFICAÇÃO"}, {Text: "1.1.1 Processo"}, {Text: "2 JUSTIFICATIVA"}, {Text: "3 OBJETIVOS DO CURSO"}}}
	numeros := textBlock{Content: "7 7 7 20 23", Lines: []textLine{{Text: "7"}, {Text: "7"}, {Text: "7"}, {Text: "20"}, {Text: "23"}}}
	if !isPDFTableOfContentsPage([]textBlock{titulos, numeros}) {
		t.Fatal("bloco com cinco linhas numeradas e pagina de sumario, mesmo sem o numero da pagina na linha")
	}
	prosa := textBlock{Content: "y", Lines: []textLine{{Text: "1.1 IDENTIFICAÇÃO"}, {Text: "O curso funciona no turno noturno."}, {Text: "2 JUSTIFICATIVA"}}}
	if isPDFTableOfContentsPage([]textBlock{prosa}) {
		t.Fatal("dois titulos num bloco nao sao sumario")
	}
}
