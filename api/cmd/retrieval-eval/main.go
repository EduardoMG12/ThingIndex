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
//	D  rrf         fusao dos dois por Reciprocal Rank Fusion (Cormack et al. 2009)
//
// O braco C (denso, pgvector + embeddings) NAO esta aqui, e a ausencia e
// deliberada e declarada: ele exige um modelo de embedding rodando, o que muda
// a natureza da execucao (deixa de ser hermetica e reproduzivel em CI). A
// estrutura ja o acomoda -- basta implementar a interface arm e registra-lo.
// Prometer um braco que nao roda seria repetir o erro do Apendice A.
//
// Uso:
//
//	go run ./cmd/retrieval-eval -corpus <caminho> -out results.csv
//	go run ./cmd/retrieval-eval -summary        # imprime o resumo por braco
package main

import (
	"encoding/csv"
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
type arm struct {
	name   string
	search func(query string) []retrieval.Result
}

func defaultCorpusPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "internal/evalcorpus/testdata/corpus-c1.json"
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(root, "internal", "evalcorpus", "testdata", "corpus-c1.json")
}

func main() {
	corpusPath := flag.String("corpus", defaultCorpusPath(), "caminho do corpus de avaliacao")
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

	arms := buildArms(corpus)
	rows := evaluate(corpus, arms)

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

func buildArms(corpus *evalcorpus.Corpus) []arm {
	docs := make([]retrieval.Document, 0, len(corpus.Documents))
	for _, doc := range corpus.Documents {
		docs = append(docs, retrieval.Document{ID: doc.ID, Text: doc.Title + "\n" + doc.Content})
	}

	bm25 := retrieval.NewBM25Index(docs, retrieval.DefaultBM25Params())

	substring := func(query string) []retrieval.Result {
		return retrieval.SubstringScorer(docs, query)
	}
	lexical := func(query string) []retrieval.Result {
		return bm25.Search(query)
	}

	return []arm{
		{name: "A-relacional", search: substring},
		{name: "B-bm25", search: lexical},
		{
			name: "D-rrf",
			search: func(query string) []retrieval.Result {
				return retrieval.ReciprocalRankFusion([][]retrieval.Result{
					substring(query),
					lexical(query),
				}, 60)
			},
		},
	}
}

// row e uma linha do CSV: um par (consulta, braco).
type row struct {
	metrics  retrieval.QueryMetrics
	variant  string
	negative bool
	falsePos float64
}

func evaluate(corpus *evalcorpus.Corpus, arms []arm) []row {
	rows := make([]row, 0, len(corpus.Queries)*len(arms))

	for _, query := range corpus.Queries {
		for _, current := range arms {
			start := time.Now()
			results := current.search(query.Text)
			elapsed := time.Since(start)

			metrics := retrieval.Evaluate(query.ID, current.name, results, query.Relevance)
			metrics.LatencyMillis = float64(elapsed.Nanoseconds()) / 1e6

			item := row{
				metrics:  metrics,
				variant:  query.Variant,
				negative: query.Negative,
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
		"query_id", "arm", "variant", "negative",
		"p_at_1", "p_at_3", "p_at_5",
		"r_at_5", "r_at_10",
		"reciprocal_rank", "ndcg_at_10",
		"false_positive", "result_count", "latency_ms",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, item := range rows {
		record := []string{
			item.metrics.QueryID,
			item.metrics.Arm,
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
