package indexing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// Celula de tabela nao e secao: a Matriz inteira e uma tabela, e o sumario
// dela tinha 16 "secoes" como "67 SIS07304 | SIS07405" e "Atividade Pedagógica".
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
		if strings.Contains(title, "sis07") || strings.Contains(title, "atividade pedag") || strings.Contains(title, "hrs coreq") {
			t.Fatalf("celula de tabela virou secao: %q", node.Title)
		}
	}
	if len(result.OutlineNodes) > 6 {
		titles := make([]string, 0, len(result.OutlineNodes))
		for _, n := range result.OutlineNodes {
			titles = append(titles, n.Title)
		}
		t.Fatalf("a matriz deveria ter poucas secoes (cabecalho do SAGRES), veio %d: %v", len(result.OutlineNodes), titles)
	}
}
