package retrieval

import (
	"math"
	"math/rand"
	"sort"
)

// IVF-Flat, o mesmo algoritmo do ivfflat do pgvector. Um documento relevante
// numa particao nao visitada some do ranking sem erro. Com probes == lists o
// indice vira varredura completa (os testes usam isso como referencia).

// IVFParams usa os nomes do pgvector (lists, probes).
type IVFParams struct {
	Lists      int
	Probes     int
	Seed       int64
	MaxIter    int
	InsertPerm []int
}

// DefaultIVFParams: lists = sqrt(N), como recomenda o pgvector, e probes = 1,
// que e o default dele.
func DefaultIVFParams(documentCount int) IVFParams {
	lists := int(math.Round(math.Sqrt(float64(documentCount))))
	if lists < 1 {
		lists = 1
	}
	if lists > documentCount {
		lists = documentCount
	}
	return IVFParams{Lists: lists, Probes: 1, Seed: 42, MaxIter: 25}
}

type IVFIndex struct {
	params    IVFParams
	centroids [][]float64
	// ordenados por ID para a saida ser estavel dado um particionamento
	buckets [][]DenseVector
}

func NewIVFIndex(vectors []DenseVector, params IVFParams) *IVFIndex {
	if len(vectors) == 0 {
		return &IVFIndex{params: params}
	}

	if params.Lists < 1 {
		params.Lists = 1
	}
	if params.Lists > len(vectors) {
		params.Lists = len(vectors)
	}
	if params.Probes < 1 {
		params.Probes = 1
	}
	if params.Probes > params.Lists {
		params.Probes = params.Lists
	}
	if params.MaxIter < 1 {
		params.MaxIter = 25
	}

	// a permutacao so tem efeito se aplicada antes do k-means
	ordered := make([]DenseVector, len(vectors))
	copy(ordered, vectors)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })

	if len(params.InsertPerm) == len(ordered) {
		permuted := make([]DenseVector, len(ordered))
		for position, index := range params.InsertPerm {
			if index < 0 || index >= len(ordered) {
				permuted = ordered
				break
			}
			permuted[position] = ordered[index]
		}
		ordered = permuted
	}

	index := &IVFIndex{params: params}
	index.centroids = kmeans(ordered, params.Lists, params.Seed, params.MaxIter)
	index.buckets = assign(ordered, index.centroids)
	return index
}

// kmeans (Lloyd, inicializacao aleatoria). Usa rand proprio e nao o global,
// senao qualquer outro sorteio no processo mudaria o particionamento.
func kmeans(vectors []DenseVector, k int, seed int64, maxIter int) [][]float64 {
	dim := len(vectors[0].Vector)
	source := rand.New(rand.NewSource(seed))

	perm := source.Perm(len(vectors))
	centroids := make([][]float64, k)
	for i := 0; i < k; i++ {
		centroid := make([]float64, dim)
		copy(centroid, vectors[perm[i]].Vector)
		centroids[i] = centroid
	}

	for iteration := 0; iteration < maxIter; iteration++ {
		sums := make([][]float64, k)
		counts := make([]int, k)
		for i := range sums {
			sums[i] = make([]float64, dim)
		}

		for _, vector := range vectors {
			best := nearestCentroid(vector.Vector, centroids)
			counts[best]++
			for d := range vector.Vector {
				sums[best][d] += vector.Vector[d]
			}
		}

		moved := false
		for i := 0; i < k; i++ {
			if counts[i] == 0 {
				// centroide orfao fica parado; re-sortear quebraria a reprodutibilidade por seed
				continue
			}
			for d := 0; d < dim; d++ {
				updated := sums[i][d] / float64(counts[i])
				if math.Abs(updated-centroids[i][d]) > 1e-12 {
					moved = true
				}
				centroids[i][d] = updated
			}
		}

		if !moved {
			break
		}
	}

	return centroids
}

// nearestCentroid desempata pelo menor indice.
func nearestCentroid(vector []float64, centroids [][]float64) int {
	best, bestScore := 0, math.Inf(-1)
	for i, centroid := range centroids {
		score := dotProduct(vector, centroid)
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

func assign(vectors []DenseVector, centroids [][]float64) [][]DenseVector {
	buckets := make([][]DenseVector, len(centroids))
	for _, vector := range vectors {
		best := nearestCentroid(vector.Vector, centroids)
		buckets[best] = append(buckets[best], vector)
	}
	for i := range buckets {
		sort.Slice(buckets[i], func(a, b int) bool {
			return buckets[i][a].ID < buckets[i][b].ID
		})
	}
	return buckets
}

// Search visita so as `probes` particoes mais proximas.
func (idx *IVFIndex) Search(queryVector []float64) []Result {
	if idx == nil || len(idx.centroids) == 0 {
		return nil
	}

	type probe struct {
		index int
		score float64
	}

	probes := make([]probe, 0, len(idx.centroids))
	for i, centroid := range idx.centroids {
		probes = append(probes, probe{index: i, score: dotProduct(queryVector, centroid)})
	}
	sort.Slice(probes, func(i, j int) bool {
		if probes[i].score != probes[j].score {
			return probes[i].score > probes[j].score
		}
		return probes[i].index < probes[j].index
	})

	limit := idx.params.Probes
	if limit > len(probes) {
		limit = len(probes)
	}

	results := make([]Result, 0)
	for _, current := range probes[:limit] {
		for _, vector := range idx.buckets[current.index] {
			score := dotProduct(queryVector, vector.Vector)
			// mesmo piso do indice exato, para a diferenca ser so da aproximacao
			if !passesFloor(score) {
				continue
			}
			results = append(results, Result{DocID: vector.ID, Score: score})
		}
	}

	sortResults(results)
	return results
}

// BucketSizes ajuda a explicar perda de recall por particao desbalanceada.
func (idx *IVFIndex) BucketSizes() []int {
	sizes := make([]int, len(idx.buckets))
	for i, bucket := range idx.buckets {
		sizes[i] = len(bucket)
	}
	return sizes
}
