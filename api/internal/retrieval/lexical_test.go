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

// Remover acento introduziria colisoes que favoreceriam artificialmente o
// casamento lexical -- exatamente o efeito que o experimento quer medir sem
// vies. Este teste trava a decisao.
func TestTokenizeNaoColapsaPalavrasDiferentesPorAcento(t *testing.T) {
	t.Parallel()

	if Tokenize("é")[0] == Tokenize("e")[0] {
		t.Fatal("'é' e 'e' viraram o mesmo token; remover acento cria casamento falso")
	}
}

// O braco A e uma reproducao FIEL do scorer de producao, defeitos inclusive.
// Este teste documenta o defeito mais visivel: por comparar substring e nao
// token, "pag" casa dentro de "pagamento" e de "pagina".
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

// O defeito que mais compromete a metrica: sem IDF, um termo comum pesa igual
// a um termo discriminativo. Aqui a consulta tem um termo inutil ("de") e um
// util ("rescisao"); o scorer de substring da 2 pontos ao documento errado
// so porque ele contem "de".
func TestSubstringScorerNaoDistingueTermoComumDeTermoDiscriminativo(t *testing.T) {
	t.Parallel()

	docs := []Document{
		{ID: "irrelevante", Text: "de de de de de de de de de de"},
		{ID: "relevante", Text: "clausula de rescisao contratual"},
	}

	results := SubstringScorer(docs, "de rescisao")

	// O documento irrelevante casa apenas "de" (score 1); o relevante casa os
	// dois termos (score 2). Ate aqui o scorer acerta -- o problema aparece
	// quando o termo comum e o UNICO em comum, o que o BM25 resolve com IDF.
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

// A diferenca central entre os bracos A e B: o BM25 usa IDF, entao um termo
// raro discrimina muito mais que um termo comum. Este teste mostra o BM25
// acertando onde a contagem de substring empata.
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
	// "contrato" aparece nos 3 documentos (IDF baixo); "rescisao" so em d3
	// (IDF alto). O d3 tem que vencer.
	if results[0].DocID != "d3" {
		t.Fatalf("o termo raro deveria dominar o ranking: esperava d3 em 1o, veio %q", results[0].DocID)
	}
}

// Sem o `1 +` na formula do IDF, um termo presente em mais da metade do corpus
// recebe IDF negativo e passa a PUNIR quem o contem: um documento que casa com
// todos os termos da consulta pode ficar atras de um que nao casa com nenhum.
func TestBM25NuncaAtribuiScoreNegativo(t *testing.T) {
	t.Parallel()

	// "comum" aparece em 4 de 5 documentos -- df/N = 0.8, a faixa em que o
	// IDF sem suavizacao fica negativo.
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

// Normalizacao por comprimento (parametro b): sem ela, documento longo vence
// por ser longo. Os dois documentos abaixo casam o termo o mesmo numero de
// vezes, mas um esta afogado em texto irrelevante.
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

// Saturacao de frequencia (parametro k1): repetir o termo 100 vezes nao pode
// valer 100 vezes mais que repetir 2 vezes, senao keyword stuffing domina o
// ranking. O ganho tem que ser sublinear.
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

// Reprodutibilidade: mesma entrada, mesma saida, sempre. Sem desempate
// explicito a ordem de itens com score igual varia entre execucoes.
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
