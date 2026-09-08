// Command retrieval-eval roda o experimento comparativo de recuperacao e emite
// um CSV com uma linha por (consulta, braco).
//
// Por que UMA LINHA POR CONSULTA e nao a media: o teste estatistico do artigo e
// pareado (Wilcoxon signed-rank, que nao assume normalidade). Ele compara os
// bracos consulta a consulta, e uma media agregada joga fora exatamente a
// informacao de que ele precisa. O CSV bruto tambem permite que qualquer pessoa
// recalcule as tabelas do artigo sem reexecutar nada.
//
// Bracos implementados aqui:
//
//	A  relacional  scorer atual do ThingIndex (contagem de substring)
//	B  bm25        Robertson & Zaragoza -- a baseline que o artigo cita e nunca usou
//	C  denso       multilingual-e5-base, varredura exata (cosseno)
//	C' denso-ivf   o mesmo modelo sob indice APROXIMADO IVF-Flat (o do pgvector)
//	D  rrf         fusao lexical (A+B) por Reciprocal Rank Fusion (Cormack et al. 2009)
//	E  rrf-denso   fusao lexical + denso -- o estado da pratica em RAG hibrido
//
// # POR QUE O BRACO C EXISTE AGORA, E COMO ELE CONTINUA HERMETICO
//
// Uma versao anterior deste arquivo declarava a ausencia do braco denso como
// deliberada, porque carregar um modelo de 1,1 GB tornaria a execucao
// nao-hermetica e nao-reproduzivel em CI. O diagnostico estava certo; a
// conclusao, nao. A solucao usada em replicabilidade de RI e separar as duas
// etapas: os embeddings sao computados UMA VEZ, fora do harness
// (scripts/embed-corpus.py), e VERSIONADOS como artefato. Aqui so acontece
// busca -- aritmetica de ponto flutuante sobre vetores lidos de um JSON.
//
// O harness continua hermetico, a medida de latencia deixa de estar
// contaminada por tempo de inferencia do modelo, e o resultado deixa de
// depender da versao de torch instalada na maquina de quem executa.
//
// # POR QUE DUAS VARIANTES DE DENSO
//
// Medir apenas o denso exato produziria uma comparacao enviesada A FAVOR da
// tese deste trabalho: "o denso tambem e deterministico" e verdade na
// varredura exata e falso no indice aproximado -- e e o aproximado que se usa
// em escala, porque a varredura exata e O(N) por consulta. Ver
// internal/retrieval/ivf.go.
//
// Uso:
//
//	go run ./cmd/retrieval-eval -corpus <caminho> -out results.csv
//	go run ./cmd/retrieval-eval -summary        # imprime o resumo por braco
//	go run ./cmd/retrieval-eval -embeddings ""  # roda so os bracos lexicais
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"thing-index/internal/evalcorpus"
	"thing-index/internal/retrieval"
)

// arm e um braco do experimento: recebe a consulta e devolve o ranking.
//
// A busca recebe a CONSULTA INTEIRA, e nao so o texto, porque os bracos densos
// precisam do ID para localizar o vetor pre-computado no artefato. Passar o
// texto e reencodar aqui exigiria o modelo dentro do harness -- exatamente o
// que o desenho evita.
type arm struct {
	name   string
	search func(query evalcorpus.Query) []retrieval.Result
}

func defaultCorpusPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "internal/evalcorpus/testdata/corpus-c1.json"
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(root, "internal", "evalcorpus", "testdata", "corpus-c1.json")
}

func defaultEmbeddingsPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "internal/evalcorpus/testdata/embeddings-c1.json"
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(root, "internal", "evalcorpus", "testdata", "embeddings-c1.json")
}

func main() {
	corpusPath := flag.String("corpus", defaultCorpusPath(), "caminho do corpus de avaliacao")
	embeddingsPath := flag.String("embeddings", defaultEmbeddingsPath(),
		"caminho do artefato de embeddings (vazio desativa os bracos densos)")
	outPath := flag.String("out", "", "arquivo CSV de saida (vazio = stdout)")
	summary := flag.Bool("summary", false, "imprime o resumo agregado por braco no stderr")
	flag.Parse()

	corpus, err := evalcorpus.Load(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "retrieval-eval: %v\n", err)
		os.Exit(1)
	}

	// A checagem de circularidade roda ANTES da medicao e aborta a execucao.
	// Um corpus circular produz numeros altos e sem significado; deixar o
	// experimento rodar assim seria produzir evidencia falsa.
	if warnings := corpus.CheckCircularity(retrieval.Tokenize); len(warnings) > 0 {
		fmt.Fprintln(os.Stderr, "retrieval-eval: corpus circular, medicao abortada:")
		for _, warning := range warnings {
			fmt.Fprintf(os.Stderr, "  - %s\n", warning)
		}
		os.Exit(1)
	}

	// O artefato de embeddings e opcional, mas quando presente ele e
	// CONFERIDO CONTRA O CORPUS. Ver checkDenseSync: um artefato obsoleto nao
	// falha, produz numero errado -- por isso a checagem aborta em vez de
	// avisar.
	var dense *retrieval.DenseArtifact
	if *embeddingsPath != "" {
		loaded, err := retrieval.LoadDenseArtifact(*embeddingsPath)
		switch {
		case os.IsNotExist(err):
			// Ausencia ANUNCIADA, nunca silenciosa: o experimento roda so com
			// os bracos lexicais e quem le o stderr sabe disso.
			fmt.Fprintf(os.Stderr,
				"retrieval-eval: %s nao existe; bracos densos DESATIVADOS.\n"+
					"  gere com: python3 scripts/embed-corpus.py --corpus %s --out %s\n",
				*embeddingsPath, *corpusPath, *embeddingsPath)
		case err != nil:
			fmt.Fprintf(os.Stderr, "retrieval-eval: %v\n", err)
			os.Exit(1)
		default:
			if problems := checkDenseSync(corpus, loaded); len(problems) > 0 {
				fmt.Fprintln(os.Stderr,
					"retrieval-eval: embeddings dessincronizados do corpus, medicao abortada:")
				for _, problem := range problems {
					fmt.Fprintf(os.Stderr, "  - %s\n", problem)
				}
				fmt.Fprintf(os.Stderr,
					"  regenere com: python3 scripts/embed-corpus.py --corpus %s --out %s\n",
					*corpusPath, *embeddingsPath)
				os.Exit(1)
			}
			dense = loaded
			fmt.Fprintf(os.Stderr,
				"retrieval-eval: embeddings de %s (dim %d) conferidos contra o corpus\n",
				dense.Manifest.Model, dense.Manifest.Dimension)
		}
	}

	// O experimento roda nas DUAS granularidades na mesma execucao.
	//
	// Ranquear documento inteiro descasa da tese (o trabalho afirma
	// proveniencia de paragrafo) e satura as metricas: com 5 documentos e um
	// relevante por consulta, R@5 e 1,000 por aritmetica. Ranquear secao
	// corrige isso -- o universo passa a 19 unidades e a baseline aleatoria de
	// P@1 cai de 0,200 para 0,053.
	//
	// As duas ficam lado a lado, e nao so a nova, por duas razoes: a
	// comparacao entre granularidades e ela propria um resultado do artigo, e
	// os bracos densos so tem vetor POR DOCUMENTO (o artefato traz 5 vetores).
	// Re-embutir por secao exigiria chamada de rede, que o principio II da
	// constituicao proibe no caminho principal.
	var allRows []row

	docUnits := evalcorpus.RetrievalUnits(documentosComoUnidadeUnica(corpus))
	docArms := buildArms(docUnits, dense)
	allRows = append(allRows, evaluate(corpus, docArms, "document", func(q evalcorpus.Query) map[string]int {
		return q.Relevance
	})...)

	sectionUnits := evalcorpus.RetrievalUnits(corpus.Documents)
	if len(sectionUnits) > len(corpus.Documents) {
		// Os bracos densos ficam de fora desta granularidade: sem vetor por
		// secao, eles devolveriam vazio e apareceriam como falha do metodo, e
		// nao como ausencia de artefato. Declarar a ausencia e melhor que
		// medir errado.
		sectionArms := buildArms(sectionUnits, nil)
		allRows = append(allRows, evaluate(corpus, sectionArms, "section", func(q evalcorpus.Query) map[string]int {
			return q.RelevanceSections
		})...)

		fmt.Fprintf(os.Stderr,
			"retrieval-eval: granularidade de secao ativa (%d unidades); bracos densos ausentes nela por falta de vetor por secao\n",
			len(sectionUnits))
	} else {
		fmt.Fprintln(os.Stderr,
			"retrieval-eval: corpus sem secoes declaradas; medindo so em nivel de documento")
	}

	rows := allRows

	writer := os.Stdout
	if *outPath != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "retrieval-eval: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	if err := writeCSV(writer, rows); err != nil {
		fmt.Fprintf(os.Stderr, "retrieval-eval: %v\n", err)
		os.Exit(1)
	}

	if *outPath != "" {
		fmt.Fprintf(os.Stderr, "retrieval-eval: %d linhas escritas em %s\n", len(rows), *outPath)
	}

	if *summary {
		printSummary(os.Stderr, corpus, rows)
	}
}

// documentosComoUnidadeUnica remove as secoes declaradas para que
// RetrievalUnits produza uma unidade por documento.
//
// Existe para a granularidade de documento continuar sendo medida EXATAMENTE
// como antes da issue #16, garantindo que os numeros historicos do artigo
// permanecam comparaveis. Sem isso, a mesma funcao devolveria secoes e a
// comparacao entre granularidades perderia o ponto de referencia.
func documentosComoUnidadeUnica(corpus *evalcorpus.Corpus) []evalcorpus.Document {
	docs := make([]evalcorpus.Document, 0, len(corpus.Documents))
	for _, doc := range corpus.Documents {
		doc.Sections = nil
		docs = append(docs, doc)
	}
	return docs
}

// buildArms monta os bracos do experimento.
//
// O artefato de embeddings e OPCIONAL: quando ausente, os bracos densos
// simplesmente nao entram e o experimento roda so com os lexicais. Isso e o
// que mantem o harness executavel em CI sem depender do artefato -- mas a
// ausencia e ANUNCIADA no stderr, nunca silenciosa. Um braco que some sem
// aviso viraria uma tabela incompleta no artigo sem ninguem perceber.
//
// `units` define a granularidade: documentos inteiros ou secoes. Ver
// buildUnits e o comentario de granularity em main.
func buildArms(units []evalcorpus.RetrievalUnit, dense *retrieval.DenseArtifact) []arm {
	docs := make([]retrieval.Document, 0, len(units))
	for _, unit := range units {
		docs = append(docs, retrieval.Document{ID: unit.ID, Text: unit.Title + "\n" + unit.Text})
	}

	bm25 := retrieval.NewBM25Index(docs, retrieval.DefaultBM25Params())

	substring := func(query evalcorpus.Query) []retrieval.Result {
		return retrieval.SubstringScorer(docs, query.Text)
	}
	lexical := func(query evalcorpus.Query) []retrieval.Result {
		return bm25.Search(query.Text)
	}

	arms := []arm{
		{name: "A-relacional", search: substring},
		{name: "B-bm25", search: lexical},
	}

	if dense == nil {
		arms = append(arms, arm{
			name: "D-rrf",
			search: func(query evalcorpus.Query) []retrieval.Result {
				return retrieval.ReciprocalRankFusion([][]retrieval.Result{
					substring(query),
					lexical(query),
				}, 60)
			},
		})
		return arms
	}

	vectors := dense.DocumentVectors()
	exact := retrieval.NewDenseIndex(vectors)

	// Os parametros do IVF seguem o pgvector: lists = sqrt(N), probes = 1
	// (o default). Ver DefaultIVFParams -- medir o default e proposital,
	// porque e a configuracao que a maioria das instalacoes roda.
	ivf := retrieval.NewIVFIndex(vectors, retrieval.DefaultIVFParams(len(vectors)))

	// denseSearch resolve o vetor da consulta pelo ID. Uma consulta sem vetor
	// e um artefato DESSINCRONIZADO do corpus, e o tratamento e devolver vazio
	// -- que aparece como falha do braco na tabela em vez de passar batido.
	// A verificacao forte fica em checkDenseSync, que aborta antes de medir.
	denseSearch := func(query evalcorpus.Query) []retrieval.Result {
		vector, ok := dense.QueryVector(query.ID)
		if !ok {
			return nil
		}
		return exact.Search(vector)
	}
	ivfSearch := func(query evalcorpus.Query) []retrieval.Result {
		vector, ok := dense.QueryVector(query.ID)
		if !ok {
			return nil
		}
		return ivf.Search(vector)
	}

	arms = append(arms,
		arm{name: "C-denso", search: denseSearch},
		arm{name: "C-denso-ivf", search: ivfSearch},
		arm{
			name: "D-rrf",
			search: func(query evalcorpus.Query) []retrieval.Result {
				return retrieval.ReciprocalRankFusion([][]retrieval.Result{
					substring(query),
					lexical(query),
				}, 60)
			},
		},
		arm{
			// Braco E: o hibrido do estado da pratica, fundindo sinal lexical
			// e denso. Ele existe porque o braco D funde duas variantes
			// LEXICAIS -- que erram junto, pelo mesmo motivo. Fundir sinais de
			// natureza diferente e o que a literatura de RAG hibrido faz, e
			// sem ele a comparacao do artigo estaria datada.
			name: "E-rrf-denso",
			search: func(query evalcorpus.Query) []retrieval.Result {
				return retrieval.ReciprocalRankFusion([][]retrieval.Result{
					lexical(query),
					denseSearch(query),
				}, 60)
			},
		},
	)

	return arms
}

// checkDenseSync garante que o artefato de embeddings corresponde AO CORPUS
// QUE ESTA SENDO MEDIDO.
//
// Este e o guard mais importante do braco denso. Um artefato obsoleto nao
// causa erro: causa numeros plausiveis calculados sobre o corpus ERRADO --
// exatamente a classe de defeito que o experimento inteiro foi desenhado para
// evitar. A conferencia e por hash SHA-256 do texto que foi enviado ao modelo,
// gravado no artefato pelo scripts/embed-corpus.py.
func checkDenseSync(corpus *evalcorpus.Corpus, dense *retrieval.DenseArtifact) []string {
	var problems []string

	documentHash := map[string]string{}
	for _, entry := range dense.Documents {
		documentHash[entry.ID] = entry.TextSHA256
	}
	queryHash := map[string]string{}
	for _, entry := range dense.Queries {
		queryHash[entry.ID] = entry.TextSHA256
	}

	hashOf := func(text string) string {
		sum := sha256.Sum256([]byte(text))
		return hex.EncodeToString(sum[:])
	}

	for _, doc := range corpus.Documents {
		// A montagem tem de ser IDENTICA a do embed-corpus.py e a do
		// buildArms. Se divergir, os bracos passariam a indexar campos
		// diferentes e a comparacao mediria isso em vez do modelo.
		want := hashOf(doc.Title + "\n" + doc.Content)
		got, ok := documentHash[doc.ID]
		if !ok {
			problems = append(problems, fmt.Sprintf("documento %q nao tem embedding", doc.ID))
			continue
		}
		if got != want {
			problems = append(problems, fmt.Sprintf(
				"documento %q mudou desde a geracao dos embeddings (hash %s.. esperado %s..)",
				doc.ID, got[:8], want[:8]))
		}
	}

	for _, query := range corpus.Queries {
		want := hashOf(query.Text)
		got, ok := queryHash[query.ID]
		if !ok {
			problems = append(problems, fmt.Sprintf("consulta %q nao tem embedding", query.ID))
			continue
		}
		if got != want {
			problems = append(problems, fmt.Sprintf(
				"consulta %q mudou desde a geracao dos embeddings (hash %s.. esperado %s..)",
				query.ID, got[:8], want[:8]))
		}
	}

	sort.Strings(problems)
	return problems
}

// row e uma linha do CSV: um par (consulta, braco) numa granularidade.
type row struct {
	metrics  retrieval.QueryMetrics
	variant  string
	negative bool
	falsePos float64
	// granularity distingue as duas unidades de recuperacao medidas na mesma
	// execucao ("document" ou "section"). Sem essa coluna as duas medicoes
	// ficariam indistinguiveis no CSV e a analise somaria maca com laranja.
	granularity string
}

// evaluate mede cada braco contra o gabarito da granularidade em uso.
//
// `judgments` escolhe o gabarito: por documento ou por secao. Passar o gabarito
// errado para a granularidade em uso nao falharia em lugar nenhum -- produziria
// zeros plausiveis, porque nenhum id casaria. Por isso quem chama e obrigado a
// dizer explicitamente qual usa, e main aborta antes de medir quando o corpus
// nao tem o gabarito pedido.
func evaluate(corpus *evalcorpus.Corpus, arms []arm, granularity string, judgments func(evalcorpus.Query) map[string]int) []row {
	rows := make([]row, 0, len(corpus.Queries)*len(arms))

	for _, query := range corpus.Queries {
		for _, current := range arms {
			start := time.Now()
			results := current.search(query)
			elapsed := time.Since(start)

			metrics := retrieval.Evaluate(query.ID, current.name, results, judgments(query))
			metrics.LatencyMillis = float64(elapsed.Nanoseconds()) / 1e6

			item := row{
				metrics:     metrics,
				variant:     query.Variant,
				negative:    query.Negative,
				granularity: granularity,
			}
			if query.Negative {
				item.falsePos = retrieval.FalsePositiveRate(results)
			}

			rows = append(rows, item)
		}
	}

	return rows
}

func writeCSV(file *os.File, rows []row) error {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"query_id", "arm", "granularity", "variant", "negative",
		"p_at_1", "p_at_3", "p_at_5",
		"r_at_5", "r_at_10",
		"reciprocal_rank", "ndcg_at_10",
		"false_positive", "result_count", "latency_ms",
		// Tetos: o maximo alcancavel dado o gabarito. Sem eles a tabela nao se
		// explica -- "P@3 = 0,312" parece 31% de acerto quando o teto e 0,375.
		"p_at_1_ceiling", "p_at_3_ceiling", "p_at_5_ceiling",
		"r_at_5_ceiling", "r_at_10_ceiling", "relevant_count",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, item := range rows {
		record := []string{
			item.metrics.QueryID,
			item.metrics.Arm,
			item.granularity,
			item.variant,
			strconv.FormatBool(item.negative),
			formatFloat(item.metrics.PrecisionAt1),
			formatFloat(item.metrics.PrecisionAt3),
			formatFloat(item.metrics.PrecisionAt5),
			formatFloat(item.metrics.RecallAt5),
			formatFloat(item.metrics.RecallAt10),
			formatFloat(item.metrics.ReciprocalRank),
			formatFloat(item.metrics.NDCGAt10),
			formatFloat(item.falsePos),
			strconv.Itoa(item.metrics.ResultCount),
			formatFloat(item.metrics.LatencyMillis),
			formatFloat(item.metrics.PrecisionAt1Ceiling),
			formatFloat(item.metrics.PrecisionAt3Ceiling),
			formatFloat(item.metrics.PrecisionAt5Ceiling),
			formatFloat(item.metrics.RecallAt5Ceiling),
			formatFloat(item.metrics.RecallAt10Ceiling),
			strconv.Itoa(item.metrics.RelevantCount),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return writer.Error()
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func printSummary(file *os.File, corpus *evalcorpus.Corpus, rows []row) {
	stats := corpus.Describe(retrieval.Tokenize)
	fmt.Fprintf(file, "\ncorpus: %d documentos, %d consultas (%d negativas), %d pares julgados\n",
		stats.Documents, stats.Queries, stats.NegativeCases, stats.JudgedPairs)

	type acc struct {
		p1, mrr, ndcg float64
		positivos     int
		falsosPos     float64
		negativos     int
	}

	byArm := map[string]*acc{}
	var armOrder []string
	for _, item := range rows {
		current, ok := byArm[item.metrics.Arm]
		if !ok {
			current = &acc{}
			byArm[item.metrics.Arm] = current
			armOrder = append(armOrder, item.metrics.Arm)
		}

		if item.negative {
			current.negativos++
			current.falsosPos += item.falsePos
			continue
		}

		current.positivos++
		current.p1 += item.metrics.PrecisionAt1
		current.mrr += item.metrics.ReciprocalRank
		current.ndcg += item.metrics.NDCGAt10
	}
	sort.Strings(armOrder)

	fmt.Fprintf(file, "\n%-14s %8s %8s %10s %14s\n", "braco", "P@1", "MRR", "nDCG@10", "falso+ (neg)")
	for _, name := range armOrder {
		current := byArm[name]
		n := float64(current.positivos)
		if n == 0 {
			continue
		}
		fp := 0.0
		if current.negativos > 0 {
			fp = current.falsosPos / float64(current.negativos)
		}
		fmt.Fprintf(file, "%-14s %8.4f %8.4f %10.4f %14.4f\n",
			name, current.p1/n, current.mrr/n, current.ndcg/n, fp)
	}

	// A queda de P@1 de literal para paraphrase e noisy e o grafico mais
	// informativo do experimento: mede a dependencia de casamento lexical, que
	// e exatamente o que um benchmark com consultas copiadas do alvo esconde.
	fmt.Fprintf(file, "\nP@1 por variante de consulta (queda = dependencia de casamento lexical)\n")
	fmt.Fprintf(file, "%-14s %10s %12s %8s\n", "braco", "literal", "paraphrase", "noisy")
	for _, name := range armOrder {
		sums := map[string]float64{}
		counts := map[string]int{}
		for _, item := range rows {
			if item.metrics.Arm != name || item.negative {
				continue
			}
			sums[item.variant] += item.metrics.PrecisionAt1
			counts[item.variant]++
		}
		mean := func(variant string) float64 {
			if counts[variant] == 0 {
				return 0
			}
			return sums[variant] / float64(counts[variant])
		}
		fmt.Fprintf(file, "%-14s %10.4f %12.4f %8.4f\n",
			name, mean("literal"), mean("paraphrase"), mean("noisy"))
	}
}
