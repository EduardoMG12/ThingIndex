// Package retrieval implementa as metricas de avaliacao de recuperacao usadas
// no experimento comparativo do TCC II.
//
// Por que este pacote existe: o benchmark atual reporta `exact_match` agregado,
// que soma unidades incomparaveis (acerto de no com acerto de documento) e
// devolve 100/100/100 -- numero que nao discrimina nada e, por isso, nao
// sustenta comparacao entre abordagens. Para afirmar que a recuperacao
// relacional e nao-inferior a densa e preciso um painel padrao da area:
// P@k, R@k, MRR e nDCG@k com relevancia graduada.
//
// As definicoes seguem a literatura de RI (Manning et al., "Introduction to
// Information Retrieval", cap. 8; Jarvelin & Kekalainen 2002 para o nDCG), e
// nao uma convencao interna -- e isso que torna os numeros comparaveis com
// trabalhos publicados.
package retrieval

import (
	"math"
	"sort"
)

// Judgment e o julgamento de relevancia de um item para uma consulta.
//
// Relevance usa escala graduada 0-3 (0 = irrelevante, 3 = perfeitamente
// relevante), e nao binaria. A escala graduada e o que permite o nDCG
// distinguir "trouxe o documento certo em 1o" de "trouxe um documento
// tangencialmente util em 1o" -- distincao que se perde no acerto binario.
type Judgment struct {
	DocID     string
	Relevance int
}

// Result e um item devolvido por um braco de recuperacao, na ordem em que foi
// devolvido (posicao 0 = primeiro colocado).
type Result struct {
	DocID string
	Score float64
}

// Qrels mapeia consulta -> julgamentos. O nome segue a convencao TREC, para
// quem for ler o codigo depois de ler a literatura.
type Qrels map[string]map[string]int

// PrecisionAtK e a fracao dos k primeiros resultados que sao relevantes.
//
// Um item conta como relevante quando tem grau >= 1. O denominador e sempre k,
// e nao len(results): recuperar menos de k itens e uma falha do braco, e
// mascarar isso dividindo pelo que foi devolvido premiaria um sistema que
// devolve um unico resultado certo e nada mais.
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

// RecallAtK e a fracao dos itens relevantes que aparecem entre os k primeiros.
//
// Quando a consulta nao tem nenhum item relevante (caso negativo, em que a
// resposta correta e "nenhum documento serve"), o recall e indefinido e
// devolvemos 0. Casos negativos devem ser avaliados por outra metrica -- ver
// FalsePositiveRate.
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

// ReciprocalRank e 1/posicao do primeiro resultado relevante, ou 0 se nenhum
// dos resultados for relevante.
//
// A media disso sobre as consultas e o MRR. E a metrica que melhor responde
// "quao rapido o usuario encontra algo util", que e a pergunta real de um
// sistema de chat documental: o usuario le o primeiro resultado, nao os dez.
func ReciprocalRank(results []Result, relevant map[string]int) float64 {
	for i, result := range results {
		if relevant[result.DocID] >= 1 {
			return 1 / float64(i+1)
		}
	}
	return 0
}

// DCGAtK e o ganho cumulativo descontado ate a posicao k.
//
// Usa a formulacao com ganho exponencial, (2^rel - 1) / log2(i+2), que e a
// adotada em TREC e nos trabalhos que o artigo vai citar. A variante linear
// (rel / log2(i+2)) daria numeros diferentes e nao comparaveis com a
// literatura -- por isso a escolha esta documentada aqui em vez de implicita.
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

// NDCGAtK e o DCG normalizado pelo DCG do ranking ideal.
//
// A normalizacao e o que torna consultas comparaveis entre si: uma consulta com
// 10 itens perfeitamente relevantes tem teto de DCG muito maior que uma com um
// unico item, e sem dividir pelo ideal a media entre consultas seria dominada
// pelas consultas ricas.
//
// Sem nenhum item relevante o valor e indefinido; devolvemos 0.
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

// FalsePositiveRate mede o comportamento em CASOS NEGATIVOS: consultas cuja
// resposta correta e "nenhum documento e relevante".
//
// Devolve 1 quando o braco entregou algum resultado (falso positivo) e 0 quando
// devolveu vazio (comportamento correto). O benchmark atual nao tem nenhum caso
// negativo, entao nunca mediu esta dimensao -- um sistema que sempre responde
// alguma coisa pontua igual a um que sabe dizer "nao encontrei".
func FalsePositiveRate(results []Result) float64 {
	if len(results) == 0 {
		return 0
	}
	return 1
}

// QueryMetrics e o painel de uma unica consulta.
//
// Guardar por consulta, e nao so a media, e requisito do teste estatistico
// pareado (Wilcoxon): o teste compara os bracos consulta a consulta, e medias
// agregadas jogam fora exatamente a informacao de que ele precisa.
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

	// Tetos: o valor MAXIMO ALCANCAVEL de cada metrica dado o gabarito desta
	// consulta. Viajam junto do valor medido porque uma tabela que publica
	// "P@3 = 0,312" sem dizer que o teto e 0,375 induz o leitor a concluir que
	// o sistema acerta 31%, quando o real e 83% do maximo possivel.
	PrecisionAt1Ceiling float64 `json:"p_at_1_ceiling"`
	PrecisionAt3Ceiling float64 `json:"p_at_3_ceiling"`
	PrecisionAt5Ceiling float64 `json:"p_at_5_ceiling"`
	RecallAt5Ceiling    float64 `json:"r_at_5_ceiling"`
	RecallAt10Ceiling   float64 `json:"r_at_10_ceiling"`
	// RelevantCount e o numero de itens com grau > 0. E ele que explica o teto
	// e permite auditar a tabela sem reabrir o corpus.
	RelevantCount int `json:"relevant_count"`
}

// PrecisionCeiling e o valor MAXIMO ALCANCAVEL de P@k dado o gabarito.
//
// P@k divide por k, nao pelo numero de resultados devolvidos. Entao uma
// consulta com um unico item relevante tem teto 1/3 em P@3: nem um sistema
// perfeito, que coloca o alvo em primeiro lugar, consegue acertar mais de 1 dos
// 3 primeiros.
//
// Publicar o valor sem o teto ao lado leva o leitor a conclusao errada: uma
// tabela que reporta "P@3 = 0,312" parece dizer que o sistema acerta 31%,
// quando o real e "83% do maximo possivel". E o mesmo defeito de construto que
// este trabalho denuncia no benchmark antigo -- metrica que nao discrimina,
// apresentada como se discriminasse.
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

// RecallCeiling e o valor MAXIMO ALCANCAVEL de R@k.
//
// Vale 1,000 sempre que o corte alcanca todos os relevantes -- e e justamente
// por isso que R@5 saturava em 1,000 quando o universo tinha 5 documentos e
// cada consulta um unico alvo: a metrica nao tinha como distinguir os bracos.
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

// countRelevant conta os itens com grau > 0.
//
// Grau 0 significa "julgado e considerado irrelevante", que e diferente de
// ausente do gabarito. Conta-lo inflaria o teto e faria o sistema parecer pior
// do que e.
func countRelevant(relevant map[string]int) int {
	total := 0
	for _, grade := range relevant {
		if grade > 0 {
			total++
		}
	}
	return total
}

// Evaluate calcula o painel completo de uma consulta.
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

		// Os tetos viajam junto das metricas para o CSV se explicar sozinho.
		// Deixa-los para a analise recalcular obrigaria a reabrir o corpus, e
		// um arquivo de resultados que depende de outro arquivo para ser
		// interpretado e um arquivo que alguem vai interpretar errado.
		PrecisionAt1Ceiling: PrecisionCeiling(relevant, 1),
		PrecisionAt3Ceiling: PrecisionCeiling(relevant, 3),
		PrecisionAt5Ceiling: PrecisionCeiling(relevant, 5),
		RecallAt5Ceiling:    RecallCeiling(relevant, 5),
		RecallAt10Ceiling:   RecallCeiling(relevant, 10),
		RelevantCount:       countRelevant(relevant),
	}
}

// ReciprocalRankFusion funde varios rankings pela formula RRF de Cormack et al.
// (2009): score(d) = sum_i 1/(k + rank_i(d)).
//
// E a base do braco hibrido. A constante k=60 e a do artigo original e virou
// padrao de facto; deixa-la fixa e uma escolha deliberada -- ajustar k por
// corpus transformaria o hibrido num modelo com hiperparametro tunado contra o
// proprio conjunto de teste, o que invalidaria a comparacao.
//
// O RRF opera sobre POSICOES, nao sobre scores. E por isso que ele funde
// rankings de escalas incomparaveis (BM25 em [0, inf) e similaridade de cosseno
// em [-1, 1]) sem precisar normalizar nada.
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

	// Desempate por DocID mantem a saida deterministica. Sem isso, duas
	// execucoes identicas poderiam produzir ordens diferentes por causa da
	// iteracao aleatoria de mapa em Go -- e determinismo e justamente uma das
	// propriedades que o trabalho afirma ter.
	sort.Slice(fused, func(i, j int) bool {
		if fused[i].Score != fused[j].Score {
			return fused[i].Score > fused[j].Score
		}
		return fused[i].DocID < fused[j].DocID
	})

	return fused
}
