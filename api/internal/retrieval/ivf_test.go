package retrieval

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// clusteredCorpus: tres grupos ortogonais com `perGroup` vetores cada, para
// sabermos de antemao o que deveria ser recuperado.
func clusteredCorpus(perGroup int) []DenseVector {
	centers := [][]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}

	vectors := make([]DenseVector, 0, perGroup*len(centers))
	for group, center := range centers {
		for i := 0; i < perGroup; i++ {
			// inclina dentro do grupo mantendo norma 1
			tilt := 0.02 * float64(i)
			vector := make([]float64, 3)
			copy(vector, center)
			other := (group + 1) % 3
			vector[group] = math.Sqrt(1 - tilt*tilt)
			vector[other] = tilt

			vectors = append(vectors, DenseVector{
				ID:     fmt.Sprintf("g%d-d%02d", group, i),
				Vector: vector,
			})
		}
	}
	return vectors
}

func TestIVFWithProbesEqualListsMatchesExactSearch(t *testing.T) {
	// Visitando todas as particoes o IVF tem de bater item a item com o exato.
	vectors := clusteredCorpus(4)
	query := []float64{1, 0, 0}

	exact := NewDenseIndex(vectors).Search(query)

	params := DefaultIVFParams(len(vectors))
	params.Lists = 3
	params.Probes = 3
	approximate := NewIVFIndex(vectors, params).Search(query)

	if len(exact) != len(approximate) {
		t.Fatalf("com probes = lists o IVF deveria devolver os mesmos %d itens, devolveu %d",
			len(exact), len(approximate))
	}
	for i := range exact {
		if exact[i].DocID != approximate[i].DocID {
			t.Fatalf("divergencia na posicao %d: exato=%q ivf=%q",
				i, exact[i].DocID, approximate[i].DocID)
		}
		if math.Abs(exact[i].Score-approximate[i].Score) > 1e-12 {
			t.Fatalf("score divergente em %q: exato=%.8f ivf=%.8f",
				exact[i].DocID, exact[i].Score, approximate[i].Score)
		}
	}
}

// angularCorpus: centros a 0, 25 e 50 graus no plano xy. Com grupos
// ortogonais nada de um grupo vizinho passaria do piso, e nao haveria recall
// a perder.
func angularCorpus(perGroup int) []DenseVector {
	degrees := []float64{0, 25, 50}

	vectors := make([]DenseVector, 0, perGroup*len(degrees))
	for group, degree := range degrees {
		for i := 0; i < perGroup; i++ {
			angle := (degree + float64(i)) * math.Pi / 180
			vectors = append(vectors, DenseVector{
				ID:     fmt.Sprintf("a%d-d%02d", group, i),
				Vector: []float64{math.Cos(angle), math.Sin(angle), 0},
			})
		}
	}
	return vectors
}

func TestIVFWithFewerProbesLosesRecall(t *testing.T) {
	// Precisa do corpus angular: com grupos ortogonais a consulta entre dois
	// grupos fica abaixo do piso e o teste dava "0 vs 0".
	vectors := angularCorpus(4)

	// Alinhada ao grupo 0: grupos 0 e 1 passam do piso, o 2 nao. probes=1 deve
	// perder o grupo 1.
	query := []float64{1, 0, 0}

	params := DefaultIVFParams(len(vectors))
	params.Lists = 3
	// Com outras seeds o k-means junta dois grupos numa particao e probes=1
	// recupera tudo por acaso (ver TestIVFRecallDependsOnSeed).
	params.Seed = 42

	params.Probes = 3
	full := NewIVFIndex(vectors, params).Search(query)

	params.Probes = 1
	partial := NewIVFIndex(vectors, params).Search(query)

	if len(full) < 2 {
		t.Fatalf("o cenario precisa de pelo menos 2 documentos acima do piso na varredura completa, veio %d", len(full))
	}

	if len(partial) >= len(full) {
		t.Fatalf("probes=1 deveria recuperar MENOS que probes=3; veio %d vs %d",
			len(partial), len(full))
	}

	t.Logf("perda de recall: probes=3 -> %d itens, probes=1 -> %d itens", len(full), len(partial))

	// o indice aproximado omite resultados, nunca inventa
	inFull := map[string]bool{}
	for _, result := range full {
		inFull[result.DocID] = true
	}
	for _, result := range partial {
		if !inFull[result.DocID] {
			t.Fatalf("probes=1 devolveu %q, que nao aparece na varredura completa: o indice esta inventando resultado",
				result.DocID)
		}
	}
}

func TestIVFIsDeterministicForFixedSeedAndOrder(t *testing.T) {
	vectors := clusteredCorpus(5)
	query := []float64{1, 0, 0}

	params := DefaultIVFParams(len(vectors))
	params.Lists = 3
	params.Probes = 1
	params.Seed = 99

	reference := NewIVFIndex(vectors, params).Search(query)

	for run := 0; run < 30; run++ {
		current := NewIVFIndex(vectors, params).Search(query)
		if len(current) != len(reference) {
			t.Fatalf("execucao %d devolveu %d itens, a referencia devolveu %d",
				run, len(current), len(reference))
		}
		for i := range current {
			if current[i].DocID != reference[i].DocID {
				t.Fatalf("execucao %d divergiu na posicao %d: %q vs %q",
					run, i, current[i].DocID, reference[i].DocID)
			}
		}
	}
}

func TestIVFSeedChangesPartitioning(t *testing.T) {
	// So a seed muda; com probes < lists isso pode mudar o resultado.
	vectors := clusteredCorpus(6)

	params := DefaultIVFParams(len(vectors))
	params.Lists = 4
	params.Probes = 1

	distinctPartitionings := map[string]bool{}
	for seed := int64(1); seed <= 12; seed++ {
		params.Seed = seed
		index := NewIVFIndex(vectors, params)

		signature := fmt.Sprint(index.BucketSizes())
		distinctPartitionings[signature] = true
	}

	if len(distinctPartitionings) < 2 {
		t.Fatalf("12 seeds deveriam produzir mais de um particionamento; produziram %d. "+
			"Se o k-means fosse insensivel a seed, a afirmacao do artigo sobre nao-determinismo "+
			"do indice aproximado nao teria base",
			len(distinctPartitionings))
	}

	t.Logf("12 seeds produziram %d particionamentos distintos", len(distinctPartitionings))
}

func TestIVFInsertionOrderChangesPartitioning(t *testing.T) {
	// Mesma seed, so a ordem de insercao muda.
	vectors := clusteredCorpus(6)

	params := DefaultIVFParams(len(vectors))
	params.Lists = 4
	params.Probes = 1
	params.Seed = 21

	baseline := fmt.Sprint(NewIVFIndex(vectors, params).BucketSizes())

	source := rand.New(rand.NewSource(2026))
	changed := false
	for attempt := 0; attempt < 20; attempt++ {
		params.InsertPerm = source.Perm(len(vectors))
		signature := fmt.Sprint(NewIVFIndex(vectors, params).BucketSizes())
		if signature != baseline {
			changed = true
			t.Logf("permutacao %d mudou o particionamento: %s -> %s", attempt, baseline, signature)
			break
		}
	}

	if !changed {
		t.Fatal("20 permutacoes de ordem de insercao nao mudaram o particionamento; " +
			"a afirmacao do artigo sobre sensibilidade a ordem precisa ser revista ou removida")
	}
}

func TestIVFRecallDependsOnSeed(t *testing.T) {
	// Aqui a seed muda o conjunto devolvido, nao so a estrutura interna.
	vectors := angularCorpus(4)
	query := []float64{1, 0, 0}

	params := DefaultIVFParams(len(vectors))
	params.Lists = 3
	params.Probes = 1

	exact := NewDenseIndex(vectors).Search(query)

	recallBySeed := map[int64]int{}
	distinctResultSets := map[string]bool{}
	for seed := int64(1); seed <= 8; seed++ {
		params.Seed = seed
		results := NewIVFIndex(vectors, params).Search(query)

		recallBySeed[seed] = len(results)

		signature := ""
		for _, result := range results {
			signature += result.DocID + "|"
		}
		distinctResultSets[signature] = true
	}

	t.Logf("varredura exata recupera %d documentos", len(exact))
	for seed := int64(1); seed <= 8; seed++ {
		t.Logf("  seed=%d -> %d documentos (%.0f%% do exato)",
			seed, recallBySeed[seed],
			100*float64(recallBySeed[seed])/float64(len(exact)))
	}

	if len(distinctResultSets) < 2 {
		t.Fatal("8 seeds deveriam produzir mais de um CONJUNTO DE RESULTADOS distinto. " +
			"Se nao produzirem, a afirmacao do artigo de que o indice aproximado torna a " +
			"resposta dependente da seed nao se sustenta e precisa ser removida")
	}

	perdeuAlguma := false
	for _, recovered := range recallBySeed {
		if recovered < len(exact) {
			perdeuAlguma = true
			break
		}
	}
	if !perdeuAlguma {
		t.Fatal("nenhuma seed perdeu documentos: o indice nao esta aproximando nada")
	}

	t.Logf("8 seeds produziram %d conjuntos de resultados distintos para a MESMA consulta",
		len(distinctResultSets))
}

func TestDefaultIVFParamsFollowsPgvectorHeuristic(t *testing.T) {
	cases := []struct {
		documents int
		lists     int
	}{
		{documents: 100, lists: 10},
		{documents: 10000, lists: 100},
		{documents: 1, lists: 1},
	}

	for _, current := range cases {
		params := DefaultIVFParams(current.documents)
		if params.Lists != current.lists {
			t.Fatalf("N=%d deveria dar lists=%d, deu %d",
				current.documents, current.lists, params.Lists)
		}
	}

	if got := DefaultIVFParams(100).Probes; got != 1 {
		t.Fatalf("probes default deveria ser 1 (o do pgvector), veio %d", got)
	}
}

func TestIVFNeverExceedsCorpusSize(t *testing.T) {
	// lists > N geraria centroides repetidos e particoes vazias.
	vectors := clusteredCorpus(1)

	params := IVFParams{Lists: 99, Probes: 99, Seed: 1, MaxIter: 10}
	index := NewIVFIndex(vectors, params)

	if sizes := index.BucketSizes(); len(sizes) > len(vectors) {
		t.Fatalf("lists (%d) nao deveria exceder o numero de documentos (%d)",
			len(sizes), len(vectors))
	}
}
