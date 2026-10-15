package main

import (
	"strings"
	"testing"

	"thing-index/internal/evalcorpus"
)

func carregarCorpus(t *testing.T) *evalcorpus.Corpus {
	t.Helper()
	corpus, err := evalcorpus.Load("../../internal/evalcorpus/testdata/corpus-c1.json")
	if err != nil {
		t.Fatalf("carregar corpus: %v", err)
	}
	return corpus
}

// O diagnostico tem de achar o corte quando ele existe.
func TestDistribuicoesSeparaveisProduzemLimiar(t *testing.T) {
	relevantes := []float64{8, 9, 10, 11, 12}
	irrelevantes := []float64{1, 2, 3, 4, 5}

	limiar, j, separavel := melhorLimiar(relevantes, irrelevantes)

	if !separavel {
		t.Fatal("distribuicoes claramente separadas foram classificadas como inseparaveis")
	}
	if j < 0.999 {
		t.Fatalf("separacao perfeita deveria dar J = 1,000; veio %.3f", j)
	}
	if limiar <= 5 || limiar > 8 {
		t.Fatalf("limiar fora da faixa que separa os conjuntos: %.3f", limiar)
	}
}

// Distribuicoes identicas: nao existe corte util e o diagnostico nao pode
// inventar um.
func TestDistribuicoesIdenticasNaoProduzemLimiar(t *testing.T) {
	valores := []float64{1, 2, 3, 4, 5}

	_, j, separavel := melhorLimiar(valores, valores)

	if separavel {
		t.Fatalf("distribuicoes identicas nao podem produzir limiar util; J = %.3f", j)
	}
}

func TestSobreposicaoTotalEhDetectada(t *testing.T) {
	// Todo irrelevante pontua acima do menor relevante.
	relevantes := []float64{1, 10}
	irrelevantes := []float64{5, 6, 7}

	if got := calcularSobreposicao(relevantes, irrelevantes); got < 0.999 {
		t.Fatalf("sobreposicao total deveria ser 1,000; veio %.3f", got)
	}

	// Nenhum irrelevante alcanca o menor relevante.
	if got := calcularSobreposicao([]float64{10, 11}, []float64{1, 2}); got > 0.001 {
		t.Fatalf("sem sobreposicao deveria dar 0,000; veio %.3f", got)
	}
}

// Sem limiar o falso positivo e 1,000; com o corte tem de cair (issue #18).
func TestLimiarReduzOFalsoPositivoNoCorpusReal(t *testing.T) {
	corpus := carregarCorpus(t)
	analises := analisarBracos(corpus)

	if len(analises) == 0 {
		t.Fatal("nenhum braco analisado")
	}

	for _, a := range analises {
		if a.FalsoPositivoSemLimiar < 0.999 {
			t.Fatalf("%s: o estado atual deveria ser falso positivo 1,000; veio %.3f -- se mudou, o artigo precisa ser revisto",
				a.Arm, a.FalsoPositivoSemLimiar)
		}
		if a.FalsoPositivoComLimiar >= a.FalsoPositivoSemLimiar {
			t.Fatalf("%s: o limiar nao reduziu o falso positivo (%.3f -> %.3f)",
				a.Arm, a.FalsoPositivoSemLimiar, a.FalsoPositivoComLimiar)
		}
	}
}

// Nenhuma das distribuicoes pode vir vazia.
func TestDiagnosticoUsaScoresDeVerdadeENaoAmostraVazia(t *testing.T) {
	corpus := carregarCorpus(t)

	for _, a := range analisarBracos(corpus) {
		if len(a.Relevantes) == 0 {
			t.Fatalf("%s: nenhum score relevante coletado", a.Arm)
		}
		if len(a.Irrelevantes) == 0 {
			t.Fatalf("%s: nenhum score irrelevante coletado", a.Arm)
		}
		// O desbalanceamento e o que justifica Youden em vez de acuracia.
		if len(a.Irrelevantes) <= len(a.Relevantes) {
			t.Fatalf("%s: esperava mais irrelevantes que relevantes (%d vs %d)",
				a.Arm, len(a.Irrelevantes), len(a.Relevantes))
		}
	}
}

// Braco inseparavel aparece como "---" na tabela.
func TestTabelaDeclaraAusenciaDeLimiarComTraco(t *testing.T) {
	tabela := buildTable([]Separabilidade{{
		Arm:                    "X-inseparavel",
		Separavel:              false,
		Sobreposicao:           1.0,
		FalsoPositivoSemLimiar: 1.0,
		FalsoPositivoComLimiar: 1.0,
	}})

	if !strings.Contains(tabela, "---") {
		t.Fatal("braco sem limiar precisa aparecer com '---', nunca com numero inventado")
	}
	if strings.Count(tabela, `\begin{tabular}`) != strings.Count(tabela, `\end{tabular}`) {
		t.Fatal("tabular desbalanceado")
	}
	if !strings.Contains(tabela, "GERADO POR") {
		t.Fatal("falta o aviso de arquivo gerado")
	}
}
