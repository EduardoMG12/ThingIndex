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
//
// # De onde vem o 1,000, sem rodar a ferramenta
//
// A regra do projeto e derivar o esperado da DEFINICAO, nunca copiar da saida
// da implementacao (ver o cabecalho de internal/provenance). A derivacao:
//
//  1. C1 e texto puro, sem form feed. buildDocumentPages atribui pagina 1 a
//     todo bloco que nao traz pagina (internal/indexing/outline.go). Logo todo
//     item de escopo de pagina em C1 TEM pagina.
//  2. Item de escopo de documento e completo por definicao quando aponta o
//     documento, e todos apontam.
//  3. Portanto TPC global = 1,000 e cada RateFor = 1,000.
//
// # O que mudou em relacao ao 0,329 anterior, item a item
//
// O valor antigo era 51/155. Os 104 itens contados como incompletos eram:
//
//	85 pares de metadado  -- estatisticas do documento inteiro, que nunca
//	                         tiveram pagina e nunca teriam. Passaram a ser
//	                         medidos por escopo de documento.
//	19 chunks             -- furo REAL de auditabilidade, agora fechado: o
//	                         chunk carrega a pagina do seu paragrafo de origem.
//
// Ou seja, 82% do furo publicado era problema de classificacao da metrica e
// 18% era defeito do indice. A decomposicao precisa aparecer no artigo, porque
// so o numero final (1,000) pareceria maquiagem.
//
// # Ressalva que NAO pode sumir do artigo
//
// 1,000 em C1 prova que o mecanismo carrega a pagina, e NAO que a pagina e a
// certa: C1 tem uma pagina so, entao "pagina correta" e "constante 1" sao
// indistinguiveis aqui. A prova de atribuicao correta esta em
// internal/indexing/chunks_test.go, com paragrafos em paginas diferentes.
const (
	tpcGlobalC1 = 1.000
	tcvC1       = 1.000
	// Todos os tipos medidos em C1 tem proveniencia completa depois da
	// mudanca de 2026-09-19: paragraph, page, outline, chunk e metadata.
	tiposCompletos = 5
	// Nenhum tipo incompleto. Se este numero voltar a subir, houve regressao.
	tiposIncompletos = 0
)

// TestTPCReportaPorTipoEstruturalENaoSoAMedia nasceu exigindo o contrario do
// que exige hoje, e a inversao e o registro de um resultado.
//
// A versao anterior falhava se NENHUM tipo estivesse incompleto, com a
// mensagem "se o esquema ganhou page_number em tabela/metadado/chunk, o artigo
// precisa ser atualizado e este teste tambem". Foi exatamente o que aconteceu
// em 2026-09-19. O teste agora trava o estado novo: todo tipo completo. Se um
// tipo voltar a ficar incompleto, houve regressao de auditabilidade e o build
// tem que quebrar.
//
// O recorte por tipo continua sendo o ponto: uma media global, alta ou baixa,
// esconde qual estrutura perdeu a coordenada.
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

// TestEscopoNaoEscondeItemQueDeveriaTerPagina e o guarda-corpo da mudanca de
// criterio. A objecao obvia a "medir por escopo" e que alguem passe a
// classificar como escopo de documento qualquer item inconveniente, e o numero
// melhore sem que a auditabilidade melhore.
//
// O teste fecha essa porta pelo tamanho: em C1 o escopo de documento tem que
// conter EXATAMENTE os pares de metadado. Se um paragrafo, uma pagina, um
// chunk, uma tabela ou um no de sumario for reclassificado para escopo de
// documento, as contas param de bater e o build quebra.
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

// TestTPCGlobalEstaTravadaNoValorDoArtigo protege o numero citado no texto.
func TestTPCGlobalEstaTravadaNoValorDoArtigo(t *testing.T) {
	corpus := carregarCorpus(t)
	report, _ := measure(corpus)

	if diff := report.Rate() - tpcGlobalC1; diff > 0.001 || diff < -0.001 {
		t.Fatalf("TPC global mudou: artigo diz %.3f, medicao deu %.3f -- atualize os dois juntos",
			tpcGlobalC1, report.Rate())
	}
}

// TestCausaDaIncompletudeEhPaginaAusente provava que a limitacao tinha UMA
// causa, nao varias. Com a incompletude zerada em C1 ele passa a provar a
// ausencia de qualquer causa: 0 == 0. Mantido de proposito -- se uma
// incompletude voltar a aparecer por outro motivo (document_id ausente,
// sequence_order invalido, conteudo vazio), ele quebra e aponta qual.
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
