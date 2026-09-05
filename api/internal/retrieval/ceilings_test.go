package retrieval

import "testing"

// O teto de uma metrica e o valor maximo ALCANCAVEL dado o gabarito, nao 1,000.
//
// P@3 com um unico item relevante tem teto 1/3: mesmo um sistema perfeito, que
// coloca o unico alvo em primeiro lugar, so pode acertar 1 dos 3 primeiros. Sem
// declarar isso, a tabela do artigo reporta "precisao de 31%" quando o real e
// "83% do maximo possivel" -- e o leitor conclui que o sistema e ruim.
//
// Este e o mesmo defeito de construto que o trabalho denuncia no benchmark
// antigo: metrica que nao discrimina, reportada como se discriminasse.

func TestTetoDePrecisaoRefleteOsRelevantesDisponiveis(t *testing.T) {
	casos := []struct {
		nome     string
		relevant map[string]int
		k        int
		esperado float64
	}{
		{"um relevante em P@3", map[string]int{"a": 3}, 3, 1.0 / 3.0},
		{"dois relevantes em P@3", map[string]int{"a": 3, "b": 2}, 3, 2.0 / 3.0},
		{"tres relevantes em P@3", map[string]int{"a": 3, "b": 2, "c": 1}, 3, 1.0},
		{"quatro relevantes em P@3", map[string]int{"a": 3, "b": 3, "c": 2, "d": 1}, 3, 1.0},
		{"um relevante em P@1", map[string]int{"a": 3}, 1, 1.0},
		{"sem relevantes", map[string]int{}, 3, 0.0},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			got := PrecisionCeiling(caso.relevant, caso.k)
			if abs(got-caso.esperado) > 1e-9 {
				t.Fatalf("teto: esperava %.4f, veio %.4f", caso.esperado, got)
			}
		})
	}
}

// Grau 0 significa "julgado e considerado irrelevante" -- diferente de ausente.
// Contar como relevante inflaria o teto e faria o sistema parecer pior do que e.
func TestTetoIgnoraGrauZero(t *testing.T) {
	relevant := map[string]int{"a": 3, "b": 0, "c": 0}
	if got := PrecisionCeiling(relevant, 3); abs(got-1.0/3.0) > 1e-9 {
		t.Fatalf("grau 0 nao deveria contar como relevante; teto veio %.4f", got)
	}
}

// O teto de recall e 1,000 sempre que o corte alcanca todos os relevantes.
// Quando ha mais relevantes que o corte, o teto cai proporcionalmente.
func TestTetoDeRecall(t *testing.T) {
	casos := []struct {
		nome     string
		relevant map[string]int
		k        int
		esperado float64
	}{
		{"relevantes cabem no corte", map[string]int{"a": 3, "b": 2}, 5, 1.0},
		{"mais relevantes que o corte", map[string]int{"a": 3, "b": 2, "c": 1}, 2, 2.0 / 3.0},
		{"sem relevantes", map[string]int{}, 5, 0.0},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			got := RecallCeiling(caso.relevant, caso.k)
			if abs(got-caso.esperado) > 1e-9 {
				t.Fatalf("teto: esperava %.4f, veio %.4f", caso.esperado, got)
			}
		})
	}
}

// O sistema NUNCA pode superar o proprio teto. Se isso acontecer, ou a metrica
// esta errada ou o teto esta -- e os dois casos invalidam a tabela.
func TestMetricaNuncaSuperaOTeto(t *testing.T) {
	relevant := map[string]int{"a": 3}
	// Ranking perfeito: o unico relevante em primeiro.
	results := []Result{{DocID: "a", Score: 1}, {DocID: "b", Score: 0.5}, {DocID: "c", Score: 0.1}}

	metrics := Evaluate("q", "braco", results, relevant)

	if metrics.PrecisionAt3 > PrecisionCeiling(relevant, 3)+1e-9 {
		t.Fatalf("P@3 %.4f superou o teto %.4f", metrics.PrecisionAt3, PrecisionCeiling(relevant, 3))
	}
	if metrics.RecallAt5 > RecallCeiling(relevant, 5)+1e-9 {
		t.Fatalf("R@5 %.4f superou o teto %.4f", metrics.RecallAt5, RecallCeiling(relevant, 5))
	}

	// E o ranking perfeito precisa ATINGIR o teto: se nao atingir, o teto esta
	// alto demais e a tabela subestimaria o desempenho.
	if abs(metrics.PrecisionAt3-PrecisionCeiling(relevant, 3)) > 1e-9 {
		t.Fatalf("ranking perfeito nao atingiu o teto de P@3: %.4f vs %.4f",
			metrics.PrecisionAt3, PrecisionCeiling(relevant, 3))
	}
}

// Evaluate precisa carregar os tetos junto das metricas, senao a analise
// precisaria reabrir o corpus para calcula-los -- e um CSV que nao se explica
// sozinho e um CSV que alguem vai interpretar errado.
func TestEvaluateCarregaOsTetos(t *testing.T) {
	relevant := map[string]int{"a": 3}
	metrics := Evaluate("q", "braco", []Result{{DocID: "a", Score: 1}}, relevant)

	if abs(metrics.PrecisionAt3Ceiling-1.0/3.0) > 1e-9 {
		t.Fatalf("teto de P@3 nao veio nas metricas: %.4f", metrics.PrecisionAt3Ceiling)
	}
	if abs(metrics.RecallAt5Ceiling-1.0) > 1e-9 {
		t.Fatalf("teto de R@5 nao veio nas metricas: %.4f", metrics.RecallAt5Ceiling)
	}
	if metrics.RelevantCount != 1 {
		t.Fatalf("contagem de relevantes: esperava 1, veio %d", metrics.RelevantCount)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
