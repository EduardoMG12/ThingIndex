package retrieval

import (
	"strings"
	"testing"
)

func TestTokenizeNormalizaEPreservaAcento(t *testing.T) {
	t.Parallel()

	got := Tokenize("Pagamento, em 15 dias -- após o RECEBIMENTO!")
	want := []string{"pagamento", "em", "15", "dias", "após", "o", "recebimento"}

	if len(got) != len(want) {
		t.Fatalf("Tokenize devolveu %d tokens (%v), esperava %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token %d = %q, esperava %q", i, got[i], want[i])
		}
	}
}

// Tokenize nao remove acentos.
func TestTokenizeNaoColapsaPalavrasDiferentesPorAcento(t *testing.T) {
	t.Parallel()

	if Tokenize("é")[0] == Tokenize("e")[0] {
		t.Fatal("'é' e 'e' viraram o mesmo token; remover acento cria casamento falso")
	}
}

// O braco A compara substring, entao "pag" casa em "pagamento" e "pagina".
func TestSubstringScorerCasaDentroDePalavra(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "pagamento", Text: "As condicoes de pagamento sao mensais."},
		{ID: "pagina", Text: "Veja a pagina seguinte."},
	}

	results := SubstringScorer(docs, "pag")

	if len(results) != 2 {
		t.Fatalf("o scorer de substring deveria casar nos dois documentos, veio %d", len(results))
	}
}

// Sem IDF, "de" pesa igual a "rescisao".
func TestSubstringScorerNaoDistingueTermoComumDeTermoDiscriminativo(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "irrelevante", Text: "de de de de de de de de de de"},
		{ID: "relevante", Text: "clausula de rescisao contratual"},
	}

	results := SubstringScorer(docs, "de rescisao")

	// irrelevante casa so "de" (1); relevante casa os dois (2)
	if results[0].DocID != "relevante" {
		t.Fatalf("esperava o documento relevante em 1o, veio %q", results[0].DocID)
	}

	somenteComum := SubstringScorer(docs, "de")
	if len(somenteComum) != 2 {
		t.Fatalf("esperava empate de casamento no termo comum, veio %d", len(somenteComum))
	}
	if somenteComum[0].Score != somenteComum[1].Score {
		t.Fatal("o scorer de substring deveria empatar aqui; se nao empata, a reproducao do braco A esta errada")
	}
}

// Com IDF o termo raro decide onde a contagem de substring empata.
func TestBM25UsaIDFParaPriorizarTermoRaro(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "d1", Text: "contrato de servico com pagamento mensal"},
		{ID: "d2", Text: "contrato de servico com pagamento anual"},
		{ID: "d3", Text: "contrato de servico com rescisao antecipada"},
	}

	index := NewBM25Index(docs, DefaultBM25Params())
	results := index.Search("contrato rescisao")

	if len(results) == 0 {
		t.Fatal("o BM25 nao devolveu nenhum resultado")
	}
	// "contrato" esta nos 3 (IDF baixo); "rescisao" so em d3
	if results[0].DocID != "d3" {
		t.Fatalf("o termo raro deveria dominar o ranking: esperava d3 em 1o, veio %q", results[0].DocID)
	}
}

// Sem o `1 +` no IDF, termo em mais da metade do corpus daria IDF negativo.
func TestBM25NuncaAtribuiScoreNegativo(t *testing.T) {
	t.Parallel()

	// df/N = 0.8
	docs := []Document{
		{ID: "d1", Text: "comum comum comum"},
		{ID: "d2", Text: "comum texto"},
		{ID: "d3", Text: "comum outro"},
		{ID: "d4", Text: "comum mais"},
		{ID: "d5", Text: "isolado"},
	}

	index := NewBM25Index(docs, DefaultBM25Params())
	results := index.Search("comum")

	if len(results) == 0 {
		t.Fatal("esperava resultados para um termo presente no corpus")
	}
	for _, result := range results {
		if result.Score <= 0 {
			t.Fatalf("documento %q recebeu score %.6f; IDF negativo inverte o ranking", result.DocID, result.Score)
		}
	}
}

// Parametro b: mesmo numero de ocorrencias, um documento afogado em texto.
func TestBM25NormalizaPeloComprimentoDoDocumento(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "curto", Text: "rescisao contratual"},
		{ID: "longo", Text: "rescisao contratual " + strings.Repeat("preenchimento irrelevante ", 200)},
	}

	index := NewBM25Index(docs, DefaultBM25Params())
	results := index.Search("rescisao")

	if results[0].DocID != "curto" {
		t.Fatalf("o documento conciso deveria vencer o documento inflado, veio %q em 1o", results[0].DocID)
	}
}

// Parametro k1: o ganho por repeticao tem de ser sublinear.
func TestBM25SaturaAFrequenciaDoTermo(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "duas", Text: "rescisao rescisao"},
		{ID: "cem", Text: strings.TrimSpace(strings.Repeat("rescisao ", 100))},
	}

	index := NewBM25Index(docs, DefaultBM25Params())
	results := index.Search("rescisao")

	scores := map[string]float64{}
	for _, result := range results {
		scores[result.DocID] = result.Score
	}

	if scores["cem"] <= 0 || scores["duas"] <= 0 {
		t.Fatalf("scores inesperados: %+v", scores)
	}

	razao := scores["cem"] / scores["duas"]
	if razao > 5 {
		t.Fatalf("50x mais repeticoes rendeu %.2fx mais score; a saturacao nao esta agindo", razao)
	}
}

func TestBM25IgnoraConsultaVaziaECorpusVazio(t *testing.T) {
	t.Parallel()

	index := NewBM25Index([]Document{{ID: "d1", Text: "algum texto"}}, DefaultBM25Params())
	if got := index.Search("   "); got != nil {
		t.Fatalf("consulta so com espacos deveria devolver nil, veio %v", got)
	}

	vazio := NewBM25Index(nil, DefaultBM25Params())
	if got := vazio.Search("qualquer"); got != nil {
		t.Fatalf("corpus vazio deveria devolver nil, veio %v", got)
	}
}

func TestBM25EDeterministico(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "zeta", Text: "rescisao contratual"},
		{ID: "alfa", Text: "rescisao contratual"},
		{ID: "beta", Text: "rescisao contratual"},
	}

	index := NewBM25Index(docs, DefaultBM25Params())
	primeiro := index.Search("rescisao")

	for i := 0; i < 30; i++ {
		atual := index.Search("rescisao")
		for j := range atual {
			if atual[j].DocID != primeiro[j].DocID {
				t.Fatalf("ordem instavel na execucao %d", i)
			}
		}
	}

	if primeiro[0].DocID != "alfa" {
		t.Fatalf("no empate o desempate e por DocID crescente, veio %q", primeiro[0].DocID)
	}
}
