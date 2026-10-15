// Command generation-eval roda o experimento de geracao ponta a ponta sobre o
// gabarito de SI e emite um CSV com uma linha por (pergunta, braco, repeticao).
//
// Bracos:
//
//	T  ThingIndex como esta (BM25 + 6 fontes + LLM), medido pela API real
//	F  o mesmo modelo com o documento inteiro no contexto, sem recuperacao
//
// Nao roda em CI: depende de rede, de provedor e da API no ar.
//
// Uso (a partir de api/, com o .env carregado no ambiente):
//
//	go run ./cmd/generation-eval -arms T,F -repeats 3 -out ../docs/article/results/generation-results.csv
//	go run ./cmd/generation-eval -summary -in ../docs/article/results/generation-results.csv
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"thing-index/internal/evalcorpus"
)

func defaultGabaritoPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "internal/evalcorpus/testdata/gabarito-si.json"
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "internal", "evalcorpus", "testdata", "gabarito-si.json")
}

// variant e pair_id vao no CSV para comparar literal x parafrase sem cruzar
// com o gabarito na mao.
var csvHeader = []string{"question_id", "stratum", "variant", "pair_id", "origin", "arm", "scope", "repeat", "model", "correct", "cited_pages", "provenance_ok", "retrieval_hit", "prompt_tokens", "completion_tokens", "usage_reported", "latency_ms", "cost_usd", "truncated", "error", "answer"}

// csvColumn devolve o indice da coluna pelo nome; indice fixo desalinhava
// quando uma coluna entrava no meio.
func csvColumn(name string) int {
	for i, column := range csvHeader {
		if column == name {
			return i
		}
	}
	panic("generation-eval: coluna inexistente no csvHeader: " + name)
}

var (
	colCorrect = csvColumn("correct")
	colModel   = csvColumn("model")
)

type asker interface {
	Ask(ctx context.Context, q evalcorpus.GabaritoQuestion) answer
}

func main() {
	gabaritoPath := flag.String("gabarito", defaultGabaritoPath(), "gabarito de perguntas")
	pdfDir := flag.String("pdf-dir", filepath.Join("..", "examples", "faculdade"), "pasta com os PDFs do gabarito")
	arms := flag.String("arms", "T,F", "bracos a rodar: T (ThingIndex) e/ou F (documento inteiro)")
	scope := flag.String("scope", "document", "braco F: document (so os documentos-alvo) ou collection (todos)")
	repeats := flag.Int("repeats", 3, "repeticoes por (pergunta, braco)")
	model := flag.String("model", "", "modelo do braco F (padrao: THINGINDEX_TEST_LLM_MODEL)")
	api := flag.String("api", "http://localhost:8000/api/v1", "base da API para o braco T")
	email := flag.String("email", "demo-faculdade@thingindex.local", "login do braco T")
	password := flag.String("password", os.Getenv("THINGINDEX_EVAL_PASSWORD"), "senha do braco T (ou THINGINDEX_EVAL_PASSWORD)")
	collection := flag.Uint("collection", 86, "id da colecao de SI no braco T")
	outPath := flag.String("out", "", "CSV de saida (vazio = stdout)")
	inputPrice := flag.Float64("input-price", 0, "USD por milhao de tokens de entrada (preco de referencia; 0 = sem custo)")
	outputPrice := flag.Float64("output-price", 0, "USD por milhao de tokens de saida")
	only := flag.String("only", "", "ids de pergunta separados por virgula (depuracao)")
	summary := flag.Bool("summary", false, "em vez de rodar, resume os CSVs de -in")
	rescore := flag.Bool("rescore", false, "em vez de rodar, reavalia correct/provenance dos CSVs de -in com o gabarito atual (sem chamar modelo) e reescreve cada arquivo")
	in := flag.String("in", "", "CSV(s) para -summary, separados por virgula")
	prose := flag.Bool("prose", false, "em vez de rodar, mede nos CSVs de -in se a pagina escrita na prosa da resposta e pagina-alvo (mesmo criterio para T e F)")
	flag.Parse()

	if *summary {
		if err := printSummary(strings.Split(*in, ","), os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "generation-eval: %v\n", err)
			os.Exit(1)
		}
		return
	}

	g, err := evalcorpus.LoadGabarito(*gabaritoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generation-eval: %v\n", err)
		os.Exit(1)
	}
	if *prose {
		if err := printProseProvenance(*in, g); err != nil {
			fmt.Fprintf(os.Stderr, "generation-eval: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *rescore {
		for _, path := range strings.Split(*in, ",") {
			if path = strings.TrimSpace(path); path == "" {
				continue
			}
			changed, err := rescoreFile(path, g)
			if err != nil {
				fmt.Fprintf(os.Stderr, "generation-eval: %s: %v\n", path, err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "%s: %d linha(s) mudaram de veredito\n", path, changed)
		}
		return
	}

	askers := map[string]asker{}
	for _, name := range strings.Split(*arms, ",") {
		switch strings.TrimSpace(name) {
		case "T":
			arm, err := newThingIndexArm(*api, *email, *password, *collection, g)
			if err != nil {
				fmt.Fprintf(os.Stderr, "generation-eval: braco T: %v\n", err)
				os.Exit(1)
			}
			askers["T"] = arm
		case "F":
			arm, err := newFullDocumentArm(*pdfDir, *scope, *model, g)
			if err != nil {
				fmt.Fprintf(os.Stderr, "generation-eval: braco F: %v\n", err)
				os.Exit(1)
			}
			askers["F"] = arm
			if *model == "" {
				*model = arm.model
			}
		case "":
		default:
			fmt.Fprintf(os.Stderr, "generation-eval: braco desconhecido %q\n", name)
			os.Exit(1)
		}
	}

	wanted := map[string]bool{}
	for _, id := range strings.Split(*only, ",") {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}

	out := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generation-eval: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}
	w := csv.NewWriter(out)
	_ = w.Write(csvHeader)

	for _, armName := range []string{"T", "F"} {
		arm, ok := askers[armName]
		if !ok {
			continue
		}
		armScope := ""
		if armName == "F" {
			armScope = *scope
		}
		for _, q := range g.Questions {
			if len(wanted) > 0 && !wanted[q.ID] {
				continue
			}
			for r := 1; r <= *repeats; r++ {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
				a := arm.Ask(ctx, q)
				cancel()
				if a.Err != nil && strings.Contains(a.Err.Error(), "usage limit") {
					fmt.Fprintf(os.Stderr, "generation-eval: credito do provedor esgotado em %s %s r%d; lote interrompido: %v\n", armName, q.ID, r, a.Err)
					w.Flush()
					os.Exit(2)
				}
				row := buildRow(q, armName, armScope, r, *model, a, *inputPrice, *outputPrice)
				_ = w.Write(row)
				w.Flush()
				status := "certa"
				if row[colCorrect] != "true" {
					status = "errada"
				}
				if a.Err != nil {
					status = "erro: " + a.Err.Error()
				}
				fmt.Fprintf(os.Stderr, "%s %s r%d %-7s %6dms %s\n", armName, q.ID, r, status, a.Latency.Milliseconds(), row[colModel])
			}
		}
	}
}

func buildRow(q evalcorpus.GabaritoQuestion, arm, scope string, repeat int, model string, a answer, inPrice, outPrice float64) []string {
	if a.Model != "" {
		model = a.Model
	}
	correct := a.Err == nil && answerCorrect(a.Text, q)
	cited, prov, hit := "", "", ""
	if !q.IsNegative() {
		if arm == "T" {
			cited = formatCitations(a.Citations)
			prov = strconv.FormatBool(provenanceOK(q, a.Citations))
			hit = prov
		} else {
			pages := citedPages(a.Text)
			cited = formatPages(pages)
			prov = strconv.FormatBool(provenanceByPage(q, pages))
		}
	}
	cost := ""
	if a.UsageReported && (inPrice > 0 || outPrice > 0) {
		cost = strconv.FormatFloat(float64(a.PromptTokens)*inPrice/1e6+float64(a.CompletionTokens)*outPrice/1e6, 'f', 6, 64)
	}
	errText := ""
	if a.Err != nil {
		errText = a.Err.Error()
	}
	return []string{
		q.ID, q.Stratum, q.Variant, q.PairID, q.Origin, arm, scope, strconv.Itoa(repeat), model,
		strconv.FormatBool(correct), cited, prov, hit,
		strconv.Itoa(a.PromptTokens), strconv.Itoa(a.CompletionTokens), strconv.FormatBool(a.UsageReported),
		strconv.FormatInt(a.Latency.Milliseconds(), 10), cost, strconv.FormatBool(a.Truncated), errText,
		strings.ReplaceAll(a.Text, "\r", ""),
	}
}
