// Command threshold-eval mede se existe um limiar de score capaz de separar
// resultado relevante de irrelevante, ja que todos os bracos devolvem algo
// mesmo nas consultas negativas. "Nao existe limiar" tambem e resultado.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"thing-index/internal/evalcorpus"
	"thing-index/internal/retrieval"
)

func main() {
	corpusPath := flag.String("corpus", filepath.Join("internal", "evalcorpus", "testdata", "corpus-c1.json"),
		"corpus de avaliacao")
	texPath := flag.String("tex", "", "arquivo .tex de saida (vazio = stdout)")
	flag.Parse()

	corpus, err := evalcorpus.Load(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "threshold-eval: %v\n", err)
		os.Exit(1)
	}

	analises := analisarBracos(corpus)
	tabela := buildTable(analises)

	if *texPath == "" {
		fmt.Print(tabela)
	} else {
		if err := os.WriteFile(*texPath, []byte(evalcorpus.Tabela(tabela)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "threshold-eval: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "threshold-eval: %s\n", *texPath)
	}

	printSummary(os.Stderr, analises)
}

// rotuloBraco traduz o nome interno do braco ("A-relacional" era o pontuador
// de contagem de termos).
func rotuloBraco(arm string) string {
	switch arm {
	case "A-relacional":
		return "A, contagem"
	case "B-bm25":
		return "B, BM25"
	case "C-denso":
		return "C, denso exato"
	}
	return arm
}

type Separabilidade struct {
	Arm          string
	Relevantes   []float64
	Irrelevantes []float64
	// Corte que maximiza o J de Youden. So vale com Separavel.
	LimiarOtimo float64
	// sensibilidade + especificidade - 1 no limiar otimo.
	J         float64
	Separavel bool
	// medido sobre as consultas negativas
	FalsoPositivoComLimiar float64
	// sempre 1,000; vai ao lado para comparacao
	FalsoPositivoSemLimiar float64
	// fracao de irrelevantes acima do menor score relevante
	Sobreposicao float64
}

func analisarBracos(corpus *evalcorpus.Corpus) []Separabilidade {
	units := evalcorpus.RetrievalUnits(corpus.Documents)
	docs := make([]retrieval.Document, 0, len(units))
	for _, unit := range units {
		docs = append(docs, retrieval.Document{ID: unit.ID, Text: unit.Title + "\n" + unit.Text})
	}

	bm25 := retrieval.NewBM25Index(docs, retrieval.DefaultBM25Params())

	bracos := []struct {
		nome   string
		buscar func(string) []retrieval.Result
	}{
		{"A-relacional", func(q string) []retrieval.Result { return retrieval.SubstringScorer(docs, q) }},
		{"B-bm25", func(q string) []retrieval.Result { return bm25.Search(q) }},
	}

	var analises []Separabilidade
	for _, braco := range bracos {
		analises = append(analises, analisarBraco(corpus, braco.nome, braco.buscar))
	}
	return analises
}

func analisarBraco(corpus *evalcorpus.Corpus, nome string, buscar func(string) []retrieval.Result) Separabilidade {
	sep := Separabilidade{Arm: nome, FalsoPositivoSemLimiar: 1.0}

	// Irrelevantes: nao-julgados das positivas + tudo que as negativas devolvem.
	for _, query := range corpus.Queries {
		results := buscar(query.Text)

		for _, result := range results {
			grau, julgado := query.RelevanceSections[result.DocID]
			if !query.Negative && julgado && grau > 0 {
				sep.Relevantes = append(sep.Relevantes, result.Score)
			} else {
				sep.Irrelevantes = append(sep.Irrelevantes, result.Score)
			}
		}
	}

	sep.LimiarOtimo, sep.J, sep.Separavel = melhorLimiar(sep.Relevantes, sep.Irrelevantes)
	sep.Sobreposicao = calcularSobreposicao(sep.Relevantes, sep.Irrelevantes)
	sep.FalsoPositivoComLimiar = falsoPositivoComLimiar(corpus, buscar, sep.LimiarOtimo)

	return sep
}

// melhorLimiar procura o corte que maximiza o J de Youden. J e nao acuracia
// porque as classes sao desbalanceadas: rejeitar tudo daria acuracia alta.
// separavel=false quando nenhum corte alcanca J > 0.
func melhorLimiar(relevantes, irrelevantes []float64) (float64, float64, bool) {
	if len(relevantes) == 0 || len(irrelevantes) == 0 {
		return 0, 0, false
	}

	candidatos := append(append([]float64{}, relevantes...), irrelevantes...)
	sort.Float64s(candidatos)

	melhorJ := 0.0
	melhorCorte := 0.0
	encontrou := false

	for _, corte := range candidatos {
		verdadeiroPositivo := 0
		for _, score := range relevantes {
			if score >= corte {
				verdadeiroPositivo++
			}
		}
		verdadeiroNegativo := 0
		for _, score := range irrelevantes {
			if score < corte {
				verdadeiroNegativo++
			}
		}

		sensibilidade := float64(verdadeiroPositivo) / float64(len(relevantes))
		especificidade := float64(verdadeiroNegativo) / float64(len(irrelevantes))
		j := sensibilidade + especificidade - 1

		if j > melhorJ {
			melhorJ = j
			melhorCorte = corte
			encontrou = true
		}
	}

	return melhorCorte, melhorJ, encontrou
}

// calcularSobreposicao: fracao de irrelevantes que pontuam acima do menor
// relevante.
func calcularSobreposicao(relevantes, irrelevantes []float64) float64 {
	if len(relevantes) == 0 || len(irrelevantes) == 0 {
		return 0
	}

	menorRelevante := math.Inf(1)
	for _, score := range relevantes {
		if score < menorRelevante {
			menorRelevante = score
		}
	}

	acima := 0
	for _, score := range irrelevantes {
		if score >= menorRelevante {
			acima++
		}
	}
	return float64(acima) / float64(len(irrelevantes))
}

// falsoPositivoComLimiar mede o falso positivo depois do corte, sobre as
// consultas negativas.
func falsoPositivoComLimiar(corpus *evalcorpus.Corpus, buscar func(string) []retrieval.Result, limiar float64) float64 {
	negativas := 0
	comResposta := 0

	for _, query := range corpus.Queries {
		if !query.Negative {
			continue
		}
		negativas++

		for _, result := range buscar(query.Text) {
			if result.Score >= limiar {
				comResposta++
				break
			}
		}
	}

	if negativas == 0 {
		return 0
	}
	return float64(comResposta) / float64(negativas)
}

func buildTable(analises []Separabilidade) string {
	var b strings.Builder

	b.WriteString("% GERADO POR cmd/threshold-eval -- nao edite a mao.\n")
	b.WriteString("\\begin{table}[htb]\n\\centering\n")
	b.WriteString("\\caption{Separabilidade dos escores e efeito de um limiar de corte. ")
	b.WriteString("A sobreposição é a fração de itens irrelevantes que pontuam acima do menor item relevante; ")
	b.WriteString("$J$ é o índice de Youden no limiar ótimo, valendo 1 para separação perfeita e 0 quando o corte não supera o acaso. ")
	b.WriteString("A taxa de falso positivo é medida sobre as consultas negativas, cuja resposta correta é não devolver nada.}\n")
	b.WriteString("\\label{tab:threshold}\n")
	b.WriteString("\\begin{tabular}{lrrrrr}\n\\hline\n")
	b.WriteString("Braço & Sobrep. & Limiar & $J$ & FP sem limiar & FP com limiar \\\\\n\\hline\n")

	for _, a := range analises {
		limiar := "---"
		j := "---"
		if a.Separavel {
			limiar = fmt.Sprintf("%.3f", a.LimiarOtimo)
			j = fmt.Sprintf("%.3f", a.J)
		}
		fmt.Fprintf(&b, "%s & %.3f & %s & %s & %.3f & %.3f \\\\\n",
			rotuloBraco(a.Arm), a.Sobreposicao, limiar, j, a.FalsoPositivoSemLimiar, a.FalsoPositivoComLimiar)
	}

	b.WriteString("\\hline\n\\end{tabular}\n\\end{table}\n")
	return b.String()
}

func printSummary(w *os.File, analises []Separabilidade) {
	fmt.Fprintln(w)
	for _, a := range analises {
		fmt.Fprintf(w, "%s:\n", a.Arm)
		fmt.Fprintf(w, "  scores relevantes:   %d\n", len(a.Relevantes))
		fmt.Fprintf(w, "  scores irrelevantes: %d\n", len(a.Irrelevantes))
		fmt.Fprintf(w, "  sobreposicao:        %.3f\n", a.Sobreposicao)
		if a.Separavel {
			fmt.Fprintf(w, "  limiar otimo:        %.4f (J = %.3f)\n", a.LimiarOtimo, a.J)
		} else {
			fmt.Fprintf(w, "  limiar otimo:        NAO EXISTE (distribuicoes sobrepostas)\n")
		}
		fmt.Fprintf(w, "  falso positivo:      %.3f -> %.3f\n\n",
			a.FalsoPositivoSemLimiar, a.FalsoPositivoComLimiar)
	}
}
