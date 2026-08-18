package provenance

import (
	"math"
	"testing"
)

func ptr(value int) *int { return &value }

func aprox(t *testing.T, nome string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %.6f, esperava %.6f", nome, got, want)
	}
}

func TestTPCContaApenasItensComProvenienciaCompleta(t *testing.T) {
	t.Parallel()

	items := []Item{
		{Kind: "paragraph", DocumentID: 1, PageNumber: ptr(1), SequenceOrder: 0, Content: "texto"},
		{Kind: "paragraph", DocumentID: 1, PageNumber: ptr(2), SequenceOrder: 1, Content: "outro"},
		// Sem pagina: nao da para apontar no documento original.
		{Kind: "paragraph", DocumentID: 1, PageNumber: nil, SequenceOrder: 2, Content: "sem pagina"},
		{Kind: "table", DocumentID: 1, PageNumber: ptr(3), SequenceOrder: 0, Content: "tabela"},
	}

	report := Evaluate(items)

	if report.Total != 4 || report.Complete != 3 {
		t.Fatalf("esperava 3 de 4 completos, veio %d de %d", report.Complete, report.Total)
	}
	aprox(t, "TPC", report.Rate(), 0.75)
	if report.MissingPageNumber != 1 {
		t.Errorf("esperava 1 item sem pagina, veio %d", report.MissingPageNumber)
	}
}

// O furo concreto que a metrica existe para expor: ParagraphNode.PageNumber e
// *int, portanto nulavel, e blocos vindos do tesseract entram sem caixa. Um
// item sem pagina nao pode ser apontado no documento original -- e "aponte onde
// voce leu isso" e exatamente o que auditabilidade significa.
func TestTPCReprovaItemSemNumeroDePagina(t *testing.T) {
	t.Parallel()

	report := Evaluate([]Item{
		{Kind: "paragraph", DocumentID: 1, PageNumber: nil, SequenceOrder: 0, Content: "veio do OCR sem caixa"},
	})

	aprox(t, "TPC", report.Rate(), 0)
	if report.MissingPageNumber != 1 {
		t.Fatalf("a causa deveria ser registrada como pagina ausente, veio %+v", report)
	}
}

// Criterio deliberadamente rigido: proveniencia PARCIAL nao permite apontar o
// trecho no original. Saber que algo veio do documento 7 sem saber a pagina nao
// e, na pratica, melhor que nao saber nada.
func TestTPCNaoAceitaProvenienciaParcial(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome string
		item Item
	}{
		{"sem document_id", Item{Kind: "p", DocumentID: 0, PageNumber: ptr(1), SequenceOrder: 0, Content: "x"}},
		{"sem pagina", Item{Kind: "p", DocumentID: 1, PageNumber: nil, SequenceOrder: 0, Content: "x"}},
		{"sequencia negativa", Item{Kind: "p", DocumentID: 1, PageNumber: ptr(1), SequenceOrder: -1, Content: "x"}},
		{"conteudo vazio", Item{Kind: "p", DocumentID: 1, PageNumber: ptr(1), SequenceOrder: 0, Content: "   "}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if report := Evaluate([]Item{caso.item}); report.Complete != 0 {
				t.Fatalf("item com proveniencia parcial nao pode contar como completo")
			}
		})
	}
}

// sequence_order = 0 e o valor LEGITIMO do primeiro item. Se a validacao o
// rejeitasse, todo primeiro paragrafo de todo documento seria contado como
// inauditavel -- exatamente o erro que ja apareceu nas tags de binding.
func TestTPCAceitaSequenceOrderZero(t *testing.T) {
	t.Parallel()

	report := Evaluate([]Item{
		{Kind: "paragraph", DocumentID: 1, PageNumber: ptr(1), SequenceOrder: 0, Content: "primeiro paragrafo"},
	})

	aprox(t, "TPC", report.Rate(), 1.0)
}

// Um indice vazio nao e "perfeitamente auditavel". Reportar 100% para o vazio
// seria a pior falha possivel numa metrica de auditabilidade.
func TestTPCDeIndiceVazioEZeroENaoUm(t *testing.T) {
	t.Parallel()

	if rate := Evaluate(nil).Rate(); rate != 0 {
		t.Fatalf("indice vazio deveria dar 0, veio %v", rate)
	}
}

// Media global esconde desigualdade entre tipos: se so os paragrafos tem pagina
// e as tabelas nao, metade da evidencia e inauditavel e a media nao mostra.
func TestTPCPorTipoRevelaDesigualdadeQueAMediaEsconde(t *testing.T) {
	t.Parallel()

	items := []Item{
		{Kind: "paragraph", DocumentID: 1, PageNumber: ptr(1), SequenceOrder: 0, Content: "a"},
		{Kind: "paragraph", DocumentID: 1, PageNumber: ptr(1), SequenceOrder: 1, Content: "b"},
		{Kind: "table", DocumentID: 1, PageNumber: nil, SequenceOrder: 0, Content: "c"},
		{Kind: "table", DocumentID: 1, PageNumber: nil, SequenceOrder: 1, Content: "d"},
	}

	report := Evaluate(items)

	aprox(t, "TPC global", report.Rate(), 0.5)
	aprox(t, "TPC de paragrafos", report.RateFor("paragraph"), 1.0)
	aprox(t, "TPC de tabelas", report.RateFor("table"), 0.0)

	kinds := report.Kinds()
	if len(kinds) != 2 || kinds[0] != "paragraph" || kinds[1] != "table" {
		t.Fatalf("os tipos precisam sair ordenados para reporte estavel, veio %v", kinds)
	}
}

func TestTCVAceitaCitacaoPresenteNaFonte(t *testing.T) {
	t.Parallel()

	report := VerifyCitations([]Citation{{
		Snippet:       "prazo de pagamento de 30 dias",
		SourceContent: "Clausula 4. O prazo de pagamento de 30 dias corre do atesto da nota fiscal.",
	}})

	aprox(t, "TCV", report.Rate(), 1.0)
}

// O modo de falha mais perigoso: apontar o documento certo e exibir um trecho
// que nao existe ali. Alucinacao com aparencia de fundamentacao.
func TestTCVReprovaCitacaoQueNaoExisteNaFonte(t *testing.T) {
	t.Parallel()

	report := VerifyCitations([]Citation{{
		Snippet:       "prazo de pagamento de 90 dias",
		SourceContent: "Clausula 4. O prazo de pagamento de 30 dias corre do atesto da nota fiscal.",
	}})

	aprox(t, "TCV", report.Rate(), 0)
}

// Quebra de linha e espaco duplo sao introduzidos na renderizacao, nao pelo
// modelo. Ignora-los evita falso negativo.
func TestTCVToleraDiferencaDeEspacoEmBrancoEDeCaixa(t *testing.T) {
	t.Parallel()

	report := VerifyCitations([]Citation{{
		Snippet:       "PRAZO   DE\n  PAGAMENTO",
		SourceContent: "O prazo de pagamento corre do atesto.",
	}})

	aprox(t, "TCV", report.Rate(), 1.0)
}

// Relaxar a comparacao ate ignorar acento produziria falso POSITIVO, que numa
// metrica de auditabilidade e o erro mais grave: daria por verificada uma
// citacao que o leitor humano nao encontraria no documento.
func TestTCVNaoIgnoraAcento(t *testing.T) {
	t.Parallel()

	report := VerifyCitations([]Citation{{
		Snippet:       "rescisao contratual",
		SourceContent: "A rescisão contratual exige aviso previo.",
	}})

	if report.Rate() != 0 {
		t.Fatal("ignorar acento daria por verificada uma citacao que nao consta no texto")
	}
}

func TestTCVIgnoraCitacaoOuFonteVazia(t *testing.T) {
	t.Parallel()

	report := VerifyCitations([]Citation{
		{Snippet: "", SourceContent: "algum texto"},
		{Snippet: "algo", SourceContent: ""},
	})

	if report.Verified != 0 {
		t.Fatalf("citacao ou fonte vazia nao pode contar como verificada, veio %d", report.Verified)
	}
	aprox(t, "TCV", report.Rate(), 0)
}
