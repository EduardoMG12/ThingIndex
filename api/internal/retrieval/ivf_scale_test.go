package retrieval

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// O corpus C1 tem so 5 documentos (2 particoes). Estes testes conferem se o
// comportamento do IVF se mantem com N maior: a perda de recall varia com N,
// mas a instabilidade entre seeds aparece em todas as escalas testadas.

// syntheticCorpus: 768 dimensoes (a do multilingual-e5-base), 8 clusters
// gaussianos, vetores com norma 1.
func syntheticCorpus(n, dim int, seed int64) []DenseVector {
	source := rand.New(rand.NewSource(seed))

	const clusters = 8
	centers := make([][]float64, clusters)
	for c := range centers {
		center := make([]float64, dim)
		for d := range center {
			center[d] = source.NormFloat64()
		}
		centers[c] = normalize(center)
	}

	vectors := make([]DenseVector, 0, n)
	for i := 0; i < n; i++ {
		center := centers[i%clusters]
		vector := make([]float64, dim)
		for d := range vector {
			// divide por sqrt(dim); senao em 768d tudo fica ortogonal ao centro
			vector[d] = center[d] + 0.35*source.NormFloat64()/math.Sqrt(float64(dim))
		}
		vectors = append(vectors, DenseVector{
			ID:     fmt.Sprintf("d%04d", i),
			Vector: normalize(vector),
		})
	}
	return vectors
}

func normalize(vector []float64) []float64 {
	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vector
	}
	for i := range vector {
		vector[i] /= norm
	}
	return vector
}

func TestIVFInstabilidadePersisteEmEscala(t *testing.T) {
	if testing.Short() {
		t.Skip("gera corpus de ate 1000 vetores em 768d; pulado em -short")
	}

	const (
		dim         = 768
		consultas   = 20
		seedsPorQ   = 10
		topK        = 10
		perturbacao = 0.15
	)

	escalas := []int{5, 50, 200, 1000}

	for _, n := range escalas {
		vectors := syntheticCorpus(n, dim, 7)
		exact := NewDenseIndex(vectors)
		params := DefaultIVFParams(n)

		querySource := rand.New(rand.NewSource(99))
		recallAcumulado := 0.0
		consultasInstaveis := 0

		for q := 0; q < consultas; q++ {
			// consulta = documento existente perturbado; a resposta certa existe
			base := vectors[querySource.Intn(n)].Vector
			query := make([]float64, dim)
			for d := range query {
				query[d] = base[d] + perturbacao*querySource.NormFloat64()/math.Sqrt(float64(dim))
			}
			query = normalize(query)

			esperadoTopK := map[string]bool{}
			for i, result := range exact.Search(query) {
				if i >= topK {
					break
				}
				esperadoTopK[result.DocID] = true
			}

			rankingsDistintos := map[string]bool{}
			sobreposicao := 0.0

			for seed := 0; seed < seedsPorQ; seed++ {
				current := params
				current.Seed = int64(seed)
				results := NewIVFIndex(vectors, current).Search(query)

				assinatura := ""
				acertos := 0
				for i, result := range results {
					if i >= topK {
						break
					}
					assinatura += result.DocID + "|"
					if esperadoTopK[result.DocID] {
						acertos++
					}
				}
				rankingsDistintos[assinatura] = true
				sobreposicao += float64(acertos) / float64(topK)
			}

			recallAcumulado += sobreposicao / float64(seedsPorQ)
			if len(rankingsDistintos) > 1 {
				consultasInstaveis++
			}
		}

		recallMedio := 100 * recallAcumulado / float64(consultas)
		t.Logf("N=%4d lists=%3d probes=%d | recall@%d do IVF vs exato: %5.1f%% | consultas instaveis entre seeds: %d/%d",
			n, params.Lists, params.Probes, topK, recallMedio, consultasInstaveis, consultas)

		// a instabilidade por seed tem de aparecer em toda escala
		if consultasInstaveis == 0 {
			t.Errorf("N=%d: nenhuma consulta variou entre seeds. A afirmacao do artigo de que "+
				"o indice aproximado torna a resposta dependente da seed nao vale nesta escala "+
				"e precisa ser restringida", n)
		}
	}
}

func TestIVFPerdaDeRecallEMaiorEmCorpusMinusculo(t *testing.T) {
	if testing.Short() {
		t.Skip("gera corpus de 200 vetores em 768d; pulado em -short")
	}

	// A perda de recall no corpus minusculo e maior que em N=200.
	const (
		dim       = 768
		consultas = 20
		topK      = 10
	)

	medirRecall := func(n int) float64 {
		vectors := syntheticCorpus(n, dim, 7)
		exact := NewDenseIndex(vectors)
		params := DefaultIVFParams(n)
		params.Seed = 42

		source := rand.New(rand.NewSource(99))
		total := 0.0

		for q := 0; q < consultas; q++ {
			base := vectors[source.Intn(n)].Vector
			query := make([]float64, dim)
			for d := range query {
				query[d] = base[d] + 0.15*source.NormFloat64()/math.Sqrt(float64(dim))
			}
			query = normalize(query)

			esperado := map[string]bool{}
			for i, result := range exact.Search(query) {
				if i >= topK {
					break
				}
				esperado[result.DocID] = true
			}

			acertos := 0
			for i, result := range NewIVFIndex(vectors, params).Search(query) {
				if i >= topK {
					break
				}
				if esperado[result.DocID] {
					acertos++
				}
			}
			total += float64(acertos) / float64(topK)
		}

		return total / float64(consultas)
	}

	minusculo := medirRecall(5)
	realista := medirRecall(200)

	t.Logf("recall@%d do IVF vs exato: N=5 -> %.1f%%, N=200 -> %.1f%%",
		topK, 100*minusculo, 100*realista)

	if minusculo >= realista {
		t.Errorf("esperava perda de recall MAIOR no corpus minusculo (N=5: %.1f%%, N=200: %.1f%%). "+
			"Se a relacao se inverteu, a ressalva de escala registrada em Ameacas a Validade "+
			"precisa ser reescrita", 100*minusculo, 100*realista)
	}
}
