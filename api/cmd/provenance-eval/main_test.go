package main

import (
	"strings"
	"testing"

	"thing-index/internal/evalcorpus"
	"thing-index/internal/provenance"
)

func carregarCorpus(t *testing.T) *evalcorpus.Corpus {
	t.Helper()
	corpus, err := evalcorpus.Load("../../internal/evalcorpus/testdata/corpus-c1.json")
	if err != nil {
		t.Fatalf("carregar corpus: %v", err)
	}
	return corpus
}

// TPC = 1,000 em C1: texto puro sem form feed, entao buildDocumentPages poe
// todo bloco na pagina 1, e itens de escopo de documento apontam o documento.
// Isso nao prova que a pagina e a certa (C1 tem uma pagina so); isso esta em
// internal/indexing/chunks_test.go.
//
// O valor antigo era 51/155: 85 pares de metadado (hoje escopo de documento)
// e 19 chunks sem pagina (hoje herdam a do paragrafo de origem).
const (
	tpcGlobalC1 = 1.000
	tcvC1       = 1.000
	// paragraph, page, outline, chunk e metadata.
	tiposCompletos = 5
)

// Todo tipo estrutural tem de estar completo; antes de 2026-09-19 este teste
// exigia o contrario.
func TestTPCReportaPorTipoEstruturalENaoSoAMedia(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	if len(report.Kinds()) != tiposCompletos {
		t.Fatalf("esperava %d tipos estruturais em C1, veio %d: %v",
			tiposCompletos, len(report.Kinds()), report.Kinds())
	}

	for _, kind := range report.Kinds() {
		if taxa := report.RateFor(kind); taxa < 0.999 {
			t.Errorf("regressao de proveniencia no tipo %q: TPC %.3f, esperava 1.000", kind, taxa)
		}
	}
}

// Em C1 o escopo de documento contem exatamente os pares de metadado; qualquer
// outro item reclassificado faz as contas nao baterem.
func TestEscopoNaoEscondeItemQueDeveriaTerPagina(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	metadados := report.ByKind["metadata"].Total
	escopoDocumento := report.ByScope[provenance.ScopeDocument].Total
	if escopoDocumento != metadados {
		t.Fatalf("escopo de documento tem %d itens mas so os %d pares de metadado deveriam estar la",
			escopoDocumento, metadados)
	}

	escopoPagina := report.ByScope[provenance.ScopePage].Total
	if escopoPagina+escopoDocumento != report.Total {
		t.Fatalf("os escopos nao cobrem o total: pagina=%d documento=%d total=%d",
			escopoPagina, escopoDocumento, report.Total)
	}
	if escopoPagina == 0 {
		t.Fatal("nenhum item de escopo de pagina: a metrica deixou de exigir pagina de qualquer coisa")
	}
}

func TestTPCGlobalEstaTravadaNoValorDoArtigo(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	if diff := report.Rate() - tpcGlobalC1; diff > 0.001 || diff < -0.001 {
		t.Fatalf("TPC global mudou: artigo diz %.3f, medicao deu %.3f -- atualize os dois juntos",
			tpcGlobalC1, report.Rate())
	}
}

// Hoje e 0 == 0, mas se voltar a haver incompletude ele aponta a causa.
func TestCausaDaIncompletudeEhPaginaAusente(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	faltando := report.Total - report.Complete
	if report.MissingPageNumber != faltando {
		t.Fatalf("esperava que TODA incompletude fosse page_number ausente; faltando=%d, page_number=%d, doc_id=%d, seq=%d, vazio=%d",
			faltando, report.MissingPageNumber, report.MissingDocumentID,
			report.InvalidSequenceOrder, report.EmptyContent)
	}
}

// TCV = 1,000: todo trecho indexado existe literalmente no documento original.
func TestTCVProvaQueAFragmentacaoNaoCorrompeConteudo(t *testing.T) {
	corpus := carregarCorpus(t)
	_, citations := measure(corpus)

	if citations.Total == 0 {
		t.Fatal("nenhuma citacao verificada: o teste nao tem poder de deteccao")
	}
	if diff := citations.Rate() - tcvC1; diff > 0.001 || diff < -0.001 {
		t.Fatalf("TCV mudou: artigo diz %.3f, medicao deu %.3f", tcvC1, citations.Rate())
	}
}

func TestTabelaGeradaEhLatexValido(t *testing.T) {
	corpus := carregarCorpus(t)
	report, citations := measure(corpus)

	tabela := buildTable(report, citations, len(corpus.Documents))

	if strings.Count(tabela, `\begin{tabular}`) != strings.Count(tabela, `\end{tabular}`) {
		t.Fatal("tabular desbalanceado")
	}
	if strings.Count(tabela, `\begin{table}`) != strings.Count(tabela, `\end{table}`) {
		t.Fatal("table desbalanceado")
	}
	if !strings.Contains(tabela, "GERADO POR") {
		t.Fatal("falta o aviso de arquivo gerado: alguem vai editar a mao")
	}
	for _, kind := range report.Kinds() {
		if !strings.Contains(tabela, traduzirTipo(kind)) {
			t.Fatalf("tipo %q ausente da tabela", kind)
		}
	}
}

func TestItemSemPaginaContaComoIncompleto(t *testing.T) {
	pagina := 3
	items := []provenance.Item{
		{Kind: "paragraph", DocumentID: 1, PageNumber: &pagina, SequenceOrder: 0, Content: "com pagina"},
		{Kind: "paragraph", DocumentID: 1, PageNumber: nil, SequenceOrder: 1, Content: "sem pagina"},
	}

	report := provenance.Evaluate(items)
	if report.Complete != 1 {
		t.Fatalf("esperava 1 item completo de 2, veio %d", report.Complete)
	}
	if report.Rate() != 0.5 {
		t.Fatalf("TPC deveria ser 0,5, veio %.3f", report.Rate())
	}
}
