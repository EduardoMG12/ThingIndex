package retrieval

import (
	"math"
	"testing"
)

// Uma metrica errada nao quebra nada: ela devolve um numero plausivel e
// invalida o experimento inteiro em silencio. Por isso os testes abaixo usam
// valores calculados A MAO a partir da definicao publicada, e nao a saida da
// propria implementacao.

const tolerance = 1e-9

func aproximadamente(t *testing.T, nome string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("%s = %.10f, esperava %.10f", nome, got, want)
	}
}

func TestPrecisionAtKUsaKNoDenominador(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "a"}, {DocID: "b"}, {DocID: "c"}}
	relevant := map[string]int{"a": 3, "c": 1}

	aproximadamente(t, "P@1", PrecisionAtK(results, relevant, 1), 1.0)
	aproximadamente(t, "P@2", PrecisionAtK(results, relevant, 2), 0.5)
	aproximadamente(t, "P@3", PrecisionAtK(results, relevant, 3), 2.0/3.0)
}

// Recuperar menos de k itens e falha do braco. Dividir pelo que foi devolvido
// premiaria um sistema que entrega um unico resultado certo e nada mais --
// P@5 = 1.0 com um resultado seria uma mentira conveniente.
func TestPrecisionAtKNaoPremiaBracoQueDevolvePoucosResultados(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "a"}}
	relevant := map[string]int{"a": 3}

	aproximadamente(t, "P@5 com 1 resultado", PrecisionAtK(results, relevant, 5), 0.2)
}

func TestRecallAtK(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "x"}, {DocID: "a"}, {DocID: "y"}, {DocID: "b"}}
	relevant := map[string]int{"a": 2, "b": 1, "c": 3}

	aproximadamente(t, "R@2", RecallAtK(results, relevant, 2), 1.0/3.0)
	aproximadamente(t, "R@4", RecallAtK(results, relevant, 4), 2.0/3.0)
}

func TestRecallSemItensRelevantesEZero(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "a"}}
	aproximadamente(t, "R@5 sem gabarito", RecallAtK(results, map[string]int{}, 5), 0)
}

func TestReciprocalRank(t *testing.T) {
	t.Parallel()

	relevant := map[string]int{"alvo": 3}

	casos := []struct {
		nome     string
		results  []Result
		esperado float64
	}{
		{"relevante em 1o", []Result{{DocID: "alvo"}, {DocID: "x"}}, 1.0},
		{"relevante em 2o", []Result{{DocID: "x"}, {DocID: "alvo"}}, 0.5},
		{"relevante em 3o", []Result{{DocID: "x"}, {DocID: "y"}, {DocID: "alvo"}}, 1.0 / 3.0},
		{"nenhum relevante", []Result{{DocID: "x"}, {DocID: "y"}}, 0},
		{"lista vazia", nil, 0},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			aproximadamente(t, "RR", ReciprocalRank(caso.results, relevant), caso.esperado)
		})
	}
}

// Valor conferido a mao pela formula (2^rel - 1)/log2(i+2):
//
//	posicao 0, rel 3 -> (8-1)/log2(2) = 7/1     = 7.0
//	posicao 1, rel 0 -> 0
//	posicao 2, rel 1 -> (2-1)/log2(4) = 1/2     = 0.5
//	                                     total  = 7.5
func TestDCGUsaGanhoExponencial(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "a"}, {DocID: "irrelevante"}, {DocID: "c"}}
	relevant := map[string]int{"a": 3, "c": 1}

	aproximadamente(t, "DCG@3", DCGAtK(results, relevant, 3), 7.5)
}

// O ranking ideal coloca rel=3 antes de rel=1:
//
//	7/log2(2) + 1/log2(3) = 7 + 0.6309297535714575 = 7.6309297535714575
//
// Com o ranking real valendo 7.5, nDCG = 7.5 / 7.6309... = 0.98284...
func TestNDCGNormalizaPeloRankingIdeal(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "a"}, {DocID: "irrelevante"}, {DocID: "c"}}
	relevant := map[string]int{"a": 3, "c": 1}

	ideal := 7 + 1/math.Log2(3)
	aproximadamente(t, "nDCG@3", NDCGAtK(results, relevant, 3), 7.5/ideal)
}

func TestNDCGDoRankingPerfeitoEUm(t *testing.T) {
	t.Parallel()

	relevant := map[string]int{"a": 3, "b": 2, "c": 1}
	perfeito := []Result{{DocID: "a"}, {DocID: "b"}, {DocID: "c"}}

	aproximadamente(t, "nDCG@10 do ranking perfeito", NDCGAtK(perfeito, relevant, 10), 1.0)
}

// A propriedade que justifica usar nDCG em vez de acerto binario: trazer o
// item MAIS relevante primeiro tem que pontuar mais que trazer o menos
// relevante primeiro. Com metrica binaria os dois casos empatam.
func TestNDCGDistingueOrdemEntreItensRelevantes(t *testing.T) {
	t.Parallel()

	relevant := map[string]int{"otimo": 3, "razoavel": 1}

	melhor := NDCGAtK([]Result{{DocID: "otimo"}, {DocID: "razoavel"}}, relevant, 10)
	pior := NDCGAtK([]Result{{DocID: "razoavel"}, {DocID: "otimo"}}, relevant, 10)

	if melhor <= pior {
		t.Fatalf("nDCG nao distinguiu a ordem: melhor=%.6f, pior=%.6f", melhor, pior)
	}
	aproximadamente(t, "nDCG do ranking otimo", melhor, 1.0)
}

func TestFalsePositiveRateMedeCasoNegativo(t *testing.T) {
	t.Parallel()

	if got := FalsePositiveRate(nil); got != 0 {
		t.Errorf("responder 'nao encontrei' deveria dar 0, veio %v", got)
	}
	if got := FalsePositiveRate([]Result{{DocID: "qualquer"}}); got != 1 {
		t.Errorf("responder algo quando nada e relevante deveria dar 1, veio %v", got)
	}
}

// O RRF opera sobre POSICOES, nao sobre scores -- e por isso que funde
// rankings de escalas incomparaveis (BM25 em [0, inf) e cosseno em [-1, 1])
// sem normalizacao. Este teste fixa essa propriedade: os scores de entrada sao
// absurdamente diferentes de proposito.
func TestReciprocalRankFusionIgnoraAEscalaDosScores(t *testing.T) {
	t.Parallel()

	bm25 := []Result{{DocID: "a", Score: 42.7}, {DocID: "b", Score: 12.1}}
	denso := []Result{{DocID: "b", Score: 0.91}, {DocID: "a", Score: 0.88}}

	fundido := ReciprocalRankFusion([][]Result{bm25, denso}, 60)

	if len(fundido) != 2 {
		t.Fatalf("esperava 2 itens fundidos, veio %d", len(fundido))
	}

	// a: 1/61 + 1/62 ; b: 1/62 + 1/61 -- identicos por construcao.
	esperado := 1/61.0 + 1/62.0
	aproximadamente(t, "score fundido de a", fundido[0].Score, esperado)
	aproximadamente(t, "score fundido de b", fundido[1].Score, esperado)
}

func TestReciprocalRankFusionPremiaConsensoEntreBracos(t *testing.T) {
	t.Parallel()

	// "consenso" aparece em 1o nos dois rankings; "so_um" aparece em 1o em
	// apenas um deles. O consenso tem que vencer.
	rankingA := []Result{{DocID: "consenso"}, {DocID: "so_um"}}
	rankingB := []Result{{DocID: "consenso"}, {DocID: "outro"}}

	fundido := ReciprocalRankFusion([][]Result{rankingA, rankingB}, 60)

	if fundido[0].DocID != "consenso" {
		t.Fatalf("o item com consenso entre bracos deveria vencer, veio %q", fundido[0].DocID)
	}
	aproximadamente(t, "score do consenso", fundido[0].Score, 2*(1/61.0))
}

// Determinismo e uma das propriedades que o trabalho afirma ter. A iteracao de
// mapa em Go e aleatoria, entao sem desempate explicito duas execucoes
// identicas poderiam devolver ordens diferentes -- e o experimento nao seria
// reproduzivel.
func TestReciprocalRankFusionEDeterministicoNoEmpate(t *testing.T) {
	t.Parallel()

	rankings := [][]Result{
		{{DocID: "zeta"}, {DocID: "alfa"}},
		{{DocID: "alfa"}, {DocID: "zeta"}},
	}

	primeiro := ReciprocalRankFusion(rankings, 60)
	for i := 0; i < 50; i++ {
		atual := ReciprocalRankFusion(rankings, 60)
		for j := range atual {
			if atual[j].DocID != primeiro[j].DocID {
				t.Fatalf("ordem instavel na execucao %d: %q vs %q", i, atual[j].DocID, primeiro[j].DocID)
			}
		}
	}

	if primeiro[0].DocID != "alfa" {
		t.Fatalf("no empate o desempate e por DocID crescente, veio %q", primeiro[0].DocID)
	}
}

func TestEvaluatePreencheOPainelCompleto(t *testing.T) {
	t.Parallel()

	results := []Result{{DocID: "alvo"}, {DocID: "x"}, {DocID: "y"}}
	relevant := map[string]int{"alvo": 3}

	metrics := Evaluate("q1", "relacional", results, relevant)

	if metrics.QueryID != "q1" || metrics.Arm != "relacional" {
		t.Fatalf("identificacao errada: %+v", metrics)
	}
	aproximadamente(t, "P@1", metrics.PrecisionAt1, 1.0)
	aproximadamente(t, "P@3", metrics.PrecisionAt3, 1.0/3.0)
	aproximadamente(t, "MRR", metrics.ReciprocalRank, 1.0)
	aproximadamente(t, "nDCG@10", metrics.NDCGAt10, 1.0)
	if metrics.ResultCount != 3 {
		t.Fatalf("ResultCount = %d, esperava 3", metrics.ResultCount)
	}
}
