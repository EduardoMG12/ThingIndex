// Package retrieval implementa os bracos de recuperacao do experimento e as
// metricas de avaliacao (P@k, R@k, MRR, nDCG@k com relevancia graduada).
package retrieval

import (
	"math"
	"sort"
)

// Judgment: Relevance vai de 0 (irrelevante) a 3 (perfeitamente relevante).
type Judgment struct {
	DocID     string
	Relevance int
}

type Result struct {
	DocID string
	Score float64
}

// Qrels mapeia consulta -> julgamentos (convencao TREC).
type Qrels map[string]map[string]int

// PrecisionAtK divide sempre por k, nao por len(results): devolver menos de k
// itens conta como erro.
func PrecisionAtK(results []Result, relevant map[string]int, k int) float64 {
	if k <= 0 {
		return 0
	}

	hits := 0
	for i, result := range results {
		if i >= k {
			break
		}
		if relevant[result.DocID] >= 1 {
			hits++
		}
	}

	return float64(hits) / float64(k)
}

// RecallAtK devolve 0 quando a consulta nao tem relevantes (caso negativo);
// esses casos sao medidos por FalsePositiveRate.
func RecallAtK(results []Result, relevant map[string]int, k int) float64 {
	total := 0
	for _, grade := range relevant {
		if grade >= 1 {
			total++
		}
	}
	if total == 0 {
		return 0
	}

	hits := 0
	for i, result := range results {
		if i >= k {
			break
		}
		if relevant[result.DocID] >= 1 {
			hits++
		}
	}

	return float64(hits) / float64(total)
}

// ReciprocalRank e 1/posicao do primeiro relevante, ou 0.
func ReciprocalRank(results []Result, relevant map[string]int) float64 {
	for i, result := range results {
		if relevant[result.DocID] >= 1 {
			return 1 / float64(i+1)
		}
	}
	return 0
}

// DCGAtK usa ganho exponencial (2^rel - 1) / log2(i+2), como no TREC.
func DCGAtK(results []Result, relevant map[string]int, k int) float64 {
	dcg := 0.0
	for i, result := range results {
		if i >= k {
			break
		}
		grade := relevant[result.DocID]
		if grade <= 0 {
			continue
		}
		dcg += (math.Pow(2, float64(grade)) - 1) / math.Log2(float64(i+2))
	}
	return dcg
}

// NDCGAtK devolve 0 quando nao ha nenhum item relevante.
func NDCGAtK(results []Result, relevant map[string]int, k int) float64 {
	ideal := idealDCG(relevant, k)
	if ideal == 0 {
		return 0
	}
	return DCGAtK(results, relevant, k) / ideal
}

func idealDCG(relevant map[string]int, k int) float64 {
	grades := make([]int, 0, len(relevant))
	for _, grade := range relevant {
		if grade > 0 {
			grades = append(grades, grade)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(grades)))

	dcg := 0.0
	for i, grade := range grades {
		if i >= k {
			break
		}
		dcg += (math.Pow(2, float64(grade)) - 1) / math.Log2(float64(i+2))
	}
	return dcg
}

// FalsePositiveRate e para casos negativos: 1 se o braco devolveu algo, 0 se
// devolveu vazio.
func FalsePositiveRate(results []Result) float64 {
	if len(results) == 0 {
		return 0
	}
	return 1
}

// QueryMetrics fica por consulta porque o Wilcoxon pareado precisa dos pares.
type QueryMetrics struct {
	QueryID        string  `json:"query_id"`
	Arm            string  `json:"arm"`
	PrecisionAt1   float64 `json:"p_at_1"`
	PrecisionAt3   float64 `json:"p_at_3"`
	PrecisionAt5   float64 `json:"p_at_5"`
	RecallAt5      float64 `json:"r_at_5"`
	RecallAt10     float64 `json:"r_at_10"`
	ReciprocalRank float64 `json:"mrr"`
	NDCGAt10       float64 `json:"ndcg_at_10"`
	LatencyMillis  float64 `json:"latency_ms"`
	ResultCount    int     `json:"result_count"`

	// Tetos: maximo alcancavel de cada metrica dado o gabarito da consulta.
	PrecisionAt1Ceiling float64 `json:"p_at_1_ceiling"`
	PrecisionAt3Ceiling float64 `json:"p_at_3_ceiling"`
	PrecisionAt5Ceiling float64 `json:"p_at_5_ceiling"`
	RecallAt5Ceiling    float64 `json:"r_at_5_ceiling"`
	RecallAt10Ceiling   float64 `json:"r_at_10_ceiling"`
	// RelevantCount: itens com grau > 0.
	RelevantCount int `json:"relevant_count"`
}

// PrecisionCeiling e o maximo de P@k dado o gabarito: com um unico relevante
// o teto de P@3 e 1/3.
func PrecisionCeiling(relevant map[string]int, k int) float64 {
	if k <= 0 {
		return 0
	}

	disponiveis := countRelevant(relevant)
	if disponiveis == 0 {
		return 0
	}
	if disponiveis >= k {
		return 1
	}
	return float64(disponiveis) / float64(k)
}

// RecallCeiling e o maximo de R@k; vale 1 quando k >= numero de relevantes.
func RecallCeiling(relevant map[string]int, k int) float64 {
	if k <= 0 {
		return 0
	}

	disponiveis := countRelevant(relevant)
	if disponiveis == 0 {
		return 0
	}
	if k >= disponiveis {
		return 1
	}
	return float64(k) / float64(disponiveis)
}

// countRelevant ignora grau 0 (julgado irrelevante), senao infla o teto.
func countRelevant(relevant map[string]int) int {
	total := 0
	for _, grade := range relevant {
		if grade > 0 {
			total++
		}
	}
	return total
}

func Evaluate(queryID string, arm string, results []Result, relevant map[string]int) QueryMetrics {
	return QueryMetrics{
		QueryID:        queryID,
		Arm:            arm,
		PrecisionAt1:   PrecisionAtK(results, relevant, 1),
		PrecisionAt3:   PrecisionAtK(results, relevant, 3),
		PrecisionAt5:   PrecisionAtK(results, relevant, 5),
		RecallAt5:      RecallAtK(results, relevant, 5),
		RecallAt10:     RecallAtK(results, relevant, 10),
		ReciprocalRank: ReciprocalRank(results, relevant),
		NDCGAt10:       NDCGAtK(results, relevant, 10),
		ResultCount:    len(results),

		// tetos no CSV para nao precisar reabrir o corpus na analise
		PrecisionAt1Ceiling: PrecisionCeiling(relevant, 1),
		PrecisionAt3Ceiling: PrecisionCeiling(relevant, 3),
		PrecisionAt5Ceiling: PrecisionCeiling(relevant, 5),
		RecallAt5Ceiling:    RecallCeiling(relevant, 5),
		RecallAt10Ceiling:   RecallCeiling(relevant, 10),
		RelevantCount:       countRelevant(relevant),
	}
}

// ReciprocalRankFusion: score(d) = sum_i 1/(k + rank_i(d)), k = 60 fixo.
// Opera sobre posicoes, entao funde BM25 e cosseno sem normalizar.
func ReciprocalRankFusion(rankings [][]Result, k float64) []Result {
	if k <= 0 {
		k = 60
	}

	scores := map[string]float64{}
	for _, ranking := range rankings {
		for position, result := range ranking {
			scores[result.DocID] += 1 / (k + float64(position+1))
		}
	}

	fused := make([]Result, 0, len(scores))
	for docID, score := range scores {
		fused = append(fused, Result{DocID: docID, Score: score})
	}

	// desempate por DocID; a ordem de iteracao de map e aleatoria
	sort.Slice(fused, func(i, j int) bool {
		if fused[i].Score != fused[j].Score {
			return fused[i].Score > fused[j].Score
		}
		return fused[i].DocID < fused[j].DocID
	})

	return fused
}
