package retrieval

import (
	"math"
	"math/rand"
	"sort"
)

// Este arquivo implementa o BRACO C-IVF: recuperacao densa APROXIMADA, pelo
// mesmo algoritmo do indice `ivfflat` do pgvector.
//
// POR QUE UM INDICE APROXIMADO PRECISA ESTAR NO EXPERIMENTO
//
// Comparar a recuperacao relacional apenas com o denso EXATO seria montar um
// espantalho invertido -- desta vez a favor deste trabalho. A varredura exata
// e O(N) por consulta: ninguem a usa em colecao grande, e a promessa de escala
// do RAG vetorial repousa inteiramente sobre indices aproximados (IVF-Flat,
// HNSW). Se o experimento so medisse o exato, a conclusao "o denso tambem e
// deterministico" seria verdadeira no que foi medido e falsa no que se usa.
//
// Com as duas variantes, a afirmacao do artigo fica precisa e defensavel:
//
//	o denso exato e deterministico mas nao escala; o denso aproximado escala
//	mas troca determinismo por latencia. A recuperacao relacional nao paga
//	nenhum dos dois precos -- e ESSA e a contribuicao, muito mais especifica
//	que "somos deterministicos".
//
// COMO O IVF-FLAT FUNCIONA, E ONDE ELE PERDE RECALL
//
//	 1. k-means agrupa os N vetores em `lists` particoes.
//	 2. Cada documento pertence a particao do centroide mais proximo.
//	 3. Na consulta, mede-se a distancia aos `lists` centroides e visitam-se
//	    apenas as `probes` particoes mais proximas.
//
// O passo 3 e a fonte da perda: um documento relevante que caiu numa particao
// nao visitada e INVISIVEL, por mais similar que seja a consulta. Nao ha
// mensagem de erro; o documento apenas some do ranking. Com probes = lists o
// indice degenera para varredura completa e o resultado volta a ser exato --
// propriedade que os testes usam como ancora de corretude.
//
// AS DUAS FONTES DE NAO-DETERMINISMO, QUE SAO O OBJETO DE MEDIDA
//
//	SEED:            a inicializacao dos centroides e aleatoria. Seeds
//	                 diferentes convergem para particionamentos diferentes.
//	ORDEM DE INSERCAO: o k-means percorre os vetores em ordem; permutar o
//	                 corpus muda a trajetoria de convergencia.
//
// Nenhuma das duas tem relacao com o corpus, com o modelo ou com a consulta.
// Sao artefatos da ESTRUTURA DE INDICE. E exatamente esse o ponto que o
// trabalho quer demonstrar: no relacional nao ha estrutura de indice capaz de
// mudar a resposta.

// IVFParams sao os hiperparametros do indice aproximado.
//
// Os nomes seguem o pgvector deliberadamente (`lists`, `probes`), para que o
// leitor do artigo consiga mapear o que foi medido aqui no que ele configura
// em producao com `CREATE INDEX ... USING ivfflat (... ) WITH (lists = ...)` e
// `SET ivfflat.probes = ...`.
type IVFParams struct {
	Lists      int
	Probes     int
	Seed       int64
	MaxIter    int
	InsertPerm []int
}

// DefaultIVFParams devolve uma configuracao tipica.
//
// A heuristica `lists = sqrt(N)` e a recomendada pela documentacao do pgvector
// para colecoes de ate 1 milhao de linhas. `probes = 1` e o DEFAULT do
// pgvector -- e escolher o default e proposital: o experimento mede o que um
// usuario obtem sem tunar nada, que e a configuracao real da maioria das
// instalacoes, nao a configuracao ideal que so aparece em benchmark de
// fornecedor.
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

// IVFIndex e o indice aproximado.
type IVFIndex struct {
	params    IVFParams
	centroids [][]float64
	// buckets[i] guarda os documentos atribuidos ao centroide i, ja ordenados
	// por ID. A ordenacao torna a saida estavel DADO um particionamento -- o
	// nao-determinismo que o experimento mede vem da escolha das particoes,
	// nao da ordem de leitura dentro delas. Separar as duas coisas e o que
	// permite atribuir a variacao observada a causa certa.
	buckets [][]DenseVector
}

// NewIVFIndex constroi o indice aproximado sobre os vetores dados.
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

	// A permutacao de insercao e aplicada ANTES do k-means, que e onde ela tem
	// efeito. Ela existe como parametro para que o experimento possa medir
	// quanta variacao a mera ordem de chegada dos documentos induz -- uma
	// pergunta que so faz sentido fazer a um indice aproximado.
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

// kmeans agrupa os vetores em k centroides (Lloyd, com inicializacao aleatoria
// simples -- a mesma familia que o pgvector usa).
//
// A aleatoriedade vem de rand.New(rand.NewSource(seed)) e nao do gerador
// global: o gerador global e compartilhado pelo processo, entao qualquer outro
// codigo que sorteasse um numero mudaria o particionamento aqui. O indice
// ficaria nao-deterministico por uma razao acidental, e a medida de
// nao-determinismo do experimento passaria a incluir ruido do proprio harness
// em vez de so a propriedade do algoritmo.
func kmeans(vectors []DenseVector, k int, seed int64, maxIter int) [][]float64 {
	dim := len(vectors[0].Vector)
	source := rand.New(rand.NewSource(seed))

	// Inicializacao: k vetores distintos sorteados como centroides iniciais.
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
				// Centroide orfao. Mante-lo onde esta e a escolha correta:
				// re-sortea-lo introduziria uma segunda fonte de aleatoriedade
				// dentro do laco e o indice deixaria de convergir de forma
				// reproduzivel para uma mesma seed -- o que impediria isolar a
				// seed como variavel do experimento.
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

// nearestCentroid devolve o indice do centroide mais proximo.
//
// Desempate pelo MENOR indice (o `>` estrito na comparacao) e deliberado: com
// vetores normalizados, empates de similaridade sao possiveis, e um desempate
// implicito deixaria a atribuicao dependente da ordem de iteracao.
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

// assign distribui os vetores nos baldes dos centroides.
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

// Search visita as `probes` particoes mais proximas e devolve os documentos
// delas, ordenados por similaridade.
//
// O documento relevante que estiver numa particao nao visitada NAO APARECE. E
// esse o custo de recall que o experimento quantifica -- e ele e invisivel do
// ponto de vista do usuario: nao ha erro, nao ha aviso, o resultado
// simplesmente vem incompleto.
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
			// O MESMO criterio de piso do braco exato. Usar criterios
			// diferentes tornaria a diferenca entre C-exato e C-IVF
			// parcialmente atribuivel ao limiar em vez de so a aproximacao do
			// indice, e o experimento deixaria de isolar a variavel que quer
			// medir.
			if !passesFloor(score) {
				continue
			}
			results = append(results, Result{DocID: vector.ID, Score: score})
		}
	}

	sortResults(results)
	return results
}

// BucketSizes expoe a distribuicao de documentos por particao.
//
// Serve ao artigo: particoes muito desbalanceadas explicam perda de recall com
// probes baixo, e reportar a distribuicao evita que o leitor tenha de aceitar
// a perda como fato bruto sem mecanismo.
func (idx *IVFIndex) BucketSizes() []int {
	sizes := make([]int, len(idx.buckets))
	for i, bucket := range idx.buckets {
		sizes[i] = len(bucket)
	}
	return sizes
}
