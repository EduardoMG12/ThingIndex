package retrieval

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// clusteredCorpus monta um corpus com estrutura de cluster CONHECIDA: tres
// grupos bem separados no espaco, `perGroup` vetores cada.
//
// A estrutura ser conhecida e o que torna os testes falseaveis: sabemos de
// antemao qual documento deveria ser recuperado, entao "o IVF perdeu" e uma
// afirmacao verificavel e nao uma leitura da saida do proprio codigo.
func clusteredCorpus(perGroup int) []DenseVector {
	// Tres eixos ortogonais como centros. Vetores do mesmo grupo ficam a
	// cosseno >= 0,99 entre si e ~0 dos outros grupos.
	centers := [][]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}

	vectors := make([]DenseVector, 0, perGroup*len(centers))
	for group, center := range centers {
		for i := 0; i < perGroup; i++ {
			// Inclina levemente o vetor dentro do proprio grupo, mantendo
			// norma 1. O desvio cresce com i, entao a ordem dentro do grupo
			// tambem e conhecida.
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
	// ANCORA DE CORRETUDE. Visitando TODAS as particoes, o IVF deixa de ser
	// aproximado e precisa concordar item a item com a varredura exata.
	//
	// Sem esta ancora, uma diferenca entre C-exato e C-IVF seria ambigua:
	// poderia ser a aproximacao (o efeito que queremos medir) ou um bug na
	// implementacao (que invalidaria a medida inteira).
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

// angularCorpus monta clusters ANGULARMENTE PROXIMOS: os centros ficam a 0, 25
// e 50 graus no plano xy.
//
// Existe porque clusteredCorpus separa os grupos por 90 graus, e a 90 graus
// nenhum documento de um grupo vizinho passa o piso de similaridade de 0,80 --
// o que impede construir o cenario de perda de recall (nao ha o que perder se
// nada era recuperavel). Com 25 graus de separacao:
//
//	cos(0)   = 1,000  -> acima do piso
//	cos(25)  = 0,906  -> acima do piso
//	cos(50)  = 0,643  -> abaixo do piso
//
// Assim uma consulta alinhada ao grupo 0 tem relevantes legitimos em DUAS
// particoes, e visitar apenas uma perde documentos que deveriam ter vindo.
func angularCorpus(perGroup int) []DenseVector {
	degrees := []float64{0, 25, 50}

	vectors := make([]DenseVector, 0, perGroup*len(degrees))
	for group, degree := range degrees {
		for i := 0; i < perGroup; i++ {
			// Espalha os membros do grupo em ate 4 graus em torno do centro,
			// o que mantem a coesao interna (cos(4) = 0,998) sem que fiquem
			// identicos.
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
	// O ACHADO CENTRAL DO BRACO C-IVF, em forma de teste.
	//
	// Um documento relevante que cai numa particao nao visitada e invisivel.
	// Nao ha erro, nao ha aviso -- o resultado simplesmente vem incompleto.
	// Este teste constroi o cenario de proposito e verifica que a perda
	// acontece, para que o numero reportado no artigo tenha mecanismo
	// demonstrado e nao seja uma afirmacao de fe.
	//
	// O corpus ANGULAR e necessario aqui, e a razao vale registro: com
	// clusters ortogonais (o de clusteredCorpus) a consulta que fica entre
	// dois grupos tem cosseno 0,707 com AMBOS -- abaixo do piso de 0,80 --
	// entao nenhum documento passa e nao ha recall a perder. A primeira versao
	// deste teste falhava exatamente assim, com "0 vs 0". Para demonstrar
	// perda de recall e preciso que documentos de MAIS DE UMA particao
	// estejam legitimamente acima do piso.
	vectors := angularCorpus(4)

	// Consulta alinhada ao grupo 0. Pela construcao de angularCorpus:
	//   grupo 0 -> cosseno ~1,00  (acima do piso)
	//   grupo 1 -> cosseno ~0,91  (acima do piso)
	//   grupo 2 -> cosseno ~0,64  (abaixo do piso)
	// Logo a varredura completa deve trazer os grupos 0 e 1, e probes=1 --
	// visitando so a particao do grupo 0 -- deve perder o grupo 1.
	query := []float64{1, 0, 0}

	params := DefaultIVFParams(len(vectors))
	params.Lists = 3
	// Seed 42 e a default de DefaultIVFParams e produz o particionamento
	// "correto" (um centroide por grupo angular). A escolha e deliberada e a
	// razao esta em TestIVFRecallDependsOnSeed: com outras seeds o k-means
	// COLAPSA dois grupos numa particao so, e ai probes=1 recupera tudo por
	// acidente. Ou seja, a mesma configuracao perde ou nao perde recall
	// dependendo de um numero aleatorio -- e esse e o achado, nao um defeito
	// do teste. Aqui fixamos a seed para isolar o efeito de `probes`.
	params.Seed = 42

	params.Probes = 3
	full := NewIVFIndex(vectors, params).Search(query)

	params.Probes = 1
	partial := NewIVFIndex(vectors, params).Search(query)

	// Pre-condicao explicita: sem documentos de dois grupos acima do piso, o
	// teste nao teria como distinguir "o IVF perdeu" de "o piso cortou tudo".
	if len(full) < 2 {
		t.Fatalf("o cenario precisa de pelo menos 2 documentos acima do piso na varredura completa, veio %d", len(full))
	}

	if len(partial) >= len(full) {
		t.Fatalf("probes=1 deveria recuperar MENOS que probes=3; veio %d vs %d",
			len(partial), len(full))
	}

	t.Logf("perda de recall: probes=3 -> %d itens, probes=1 -> %d itens", len(full), len(partial))

	// Todo item devolvido com probes=1 tambem tem de aparecer com probes=3: o
	// indice aproximado OMITE resultados, nunca inventa. Se inventasse, o
	// problema seria de corretude, nao de recall.
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
	// Fixados seed E ordem de insercao, o IVF tem de ser reproduzivel. Sem
	// isso nao seria possivel atribuir a variacao observada entre execucoes a
	// seed ou a ordem -- ela poderia vir de mapa iterado em ordem aleatoria
	// dentro do proprio harness.
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
	// A EVIDENCIA QUE SUSTENTA A AFIRMACAO DO ARTIGO.
	//
	// Trocar apenas a seed -- sem tocar no corpus, no modelo ou na consulta --
	// muda o particionamento. Com probes < lists isso pode mudar o resultado
	// devolvido ao usuario.
	//
	// E esta a diferenca qualitativa que o trabalho reivindica: na recuperacao
	// relacional nao existe estrutura de indice capaz de fazer a resposta
	// depender de um numero aleatorio escolhido no momento da criacao do
	// indice.
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
	// A SEGUNDA fonte de nao-determinismo, e a mais contra-intuitiva: com a
	// MESMA seed, apenas mudar a ordem em que os documentos foram inseridos
	// pode alterar o particionamento.
	//
	// Isso importa para o argumento porque a ordem de insercao nao e uma
	// escolha de configuracao -- e o historico de uso do sistema. Dois
	// usuarios que subiram os mesmos documentos em ordem diferente podem obter
	// respostas diferentes para a mesma pergunta.
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
	// O RESULTADO MAIS FORTE DO BRACO C-IVF, e o mais desconfortavel para
	// quem defende o indice aproximado.
	//
	// TestIVFSeedChangesPartitioning mostra que a seed muda a estrutura
	// INTERNA do indice. Isso, sozinho, um cetico pode descartar: "e detalhe
	// de implementacao, o usuario nao ve".
	//
	// Este teste fecha o argumento mostrando que a seed muda O QUE O USUARIO
	// RECEBE. Mesmo corpus, mesma consulta, mesma configuracao de `lists` e
	// `probes` -- e conjuntos de documentos diferentes, porque o k-means
	// convergiu para outro particionamento.
	//
	// A traducao para o artigo: num indice aproximado, a resposta a uma
	// pergunta depende de um numero aleatorio sorteado quando o indice foi
	// criado. Nao ha consulta, corpus ou modelo que explique a diferenca, e
	// nao ha como o usuario perceber que ela ocorreu. Na recuperacao
	// relacional essa classe de variacao nao existe -- nao ha estrutura de
	// indice entre a pergunta e a resposta.
	vectors := angularCorpus(4)
	query := []float64{1, 0, 0}

	params := DefaultIVFParams(len(vectors))
	params.Lists = 3
	params.Probes = 1

	// O teto: o que a varredura exata recuperaria.
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

	// Pelo menos uma seed tem de perder documentos em relacao ao exato -- e o
	// que caracteriza a aproximacao. Se nenhuma perdesse, o indice nao estaria
	// aproximando nada e a comparacao com o exato seria vazia.
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
	// lists = sqrt(N) e a recomendacao do pgvector para colecoes de ate 1M.
	// O teste trava a heuristica porque ela e reportada no artigo como a
	// configuracao usada -- se o codigo mudasse, o artigo passaria a descrever
	// um experimento que nao foi o executado.
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

	// probes=1 e o DEFAULT do pgvector. Medir o default e proposital: e a
	// configuracao que a maioria das instalacoes efetivamente roda.
	if got := DefaultIVFParams(100).Probes; got != 1 {
		t.Fatalf("probes default deveria ser 1 (o do pgvector), veio %d", got)
	}
}

func TestIVFNeverExceedsCorpusSize(t *testing.T) {
	// Guard de robustez: lists > N faria o k-means sortear centroides
	// repetidos e produzir particoes vazias.
	vectors := clusteredCorpus(1)

	params := IVFParams{Lists: 99, Probes: 99, Seed: 1, MaxIter: 10}
	index := NewIVFIndex(vectors, params)

	if sizes := index.BucketSizes(); len(sizes) > len(vectors) {
		t.Fatalf("lists (%d) nao deveria exceder o numero de documentos (%d)",
			len(sizes), len(vectors))
	}
}
