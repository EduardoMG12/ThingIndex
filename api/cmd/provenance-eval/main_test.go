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

// Constantes travadas: o artigo cita estes numeros, entao altera-los obriga a
// revisar o texto (constituicao, principio III).
const (
	tpcGlobalC1 = 0.329
	tcvC1       = 1.000
	// Tipos com proveniencia completa por construcao do esquema.
	tiposCompletos = 3 // paragraph, page, outline
	// Tipos sem page_number no esquema.
	tiposIncompletos = 2 // chunk, metadata
)

// TestTPCReportaPorTipoEstruturalENaoSoAMedia e o teste central da issue #17.
//
// A media global de 0,329 sozinha sugere que o indice inteiro e mal auditavel.
// A verdade medida e outra: a proveniencia e PERFEITA onde o esquema tem
// page_number e INEXISTENTE onde nao tem. Publicar so a media esconderia
// exatamente o achado.
func TestTPCReportaPorTipoEstruturalENaoSoAMedia(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	if len(report.Kinds()) < tiposCompletos+tiposIncompletos {
		t.Fatalf("esperava ao menos %d tipos estruturais, veio %d: %v",
			tiposCompletos+tiposIncompletos, len(report.Kinds()), report.Kinds())
	}

	completos := 0
	incompletos := 0
	for _, kind := range report.Kinds() {
		taxa := report.RateFor(kind)
		switch {
		case taxa >= 0.999:
			completos++
		case taxa <= 0.001:
			incompletos++
		}
	}

	if completos == 0 {
		t.Fatal("nenhum tipo com proveniencia completa: o indice regrediu")
	}
	if incompletos == 0 {
		t.Fatal("nenhum tipo incompleto -- se o esquema ganhou page_number em tabela/metadado/chunk, o artigo precisa ser atualizado e este teste tambem")
	}
}

// TestTPCGlobalEstaTravadaNoValorDoArtigo protege o numero citado no texto.
func TestTPCGlobalEstaTravadaNoValorDoArtigo(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	if diff := report.Rate() - tpcGlobalC1; diff > 0.001 || diff < -0.001 {
		t.Fatalf("TPC global mudou: artigo diz %.3f, medicao deu %.3f -- atualize os dois juntos",
			tpcGlobalC1, report.Rate())
	}
}

// TestCausaDaIncompletudeEhPaginaAusente prova que a limitacao tem UMA causa,
// nao varias. Isso importa para a Discussao: um defeito unico e enderecavel;
// varios difusos sugerem problema estrutural.
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

// TestTCVProvaQueAFragmentacaoNaoCorrompeConteudo trava a segunda metrica.
//
// TCV = 1,000 significa que todo trecho indexado existe LITERALMENTE no
// documento original. E o que separa "citou" de "citou corretamente", e e a
// evidencia de que a fragmentacao em tabelas tipadas preserva o texto.
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

// TestTabelaGeradaEhLatexValido evita quebrar a compilacao do artigo.
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
	// Toda linha de tipo precisa aparecer na tabela.
	for _, kind := range report.Kinds() {
		if !strings.Contains(tabela, traduzirTipo(kind)) {
			t.Fatalf("tipo %q ausente da tabela", kind)
		}
	}
}

// TestItemSemPaginaContaComoIncompleto e o teste de poder de deteccao da
// propria metrica: se a TPC aceitasse pagina nula, ela nao mediria nada.
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
