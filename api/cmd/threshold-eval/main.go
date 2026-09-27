// Command threshold-eval mede se existe um limiar de score capaz de separar
// resultado relevante de irrelevante.
//
// Existe por causa do achado mais grave do trabalho: a taxa de falso positivo e
// 1,000 nos SEIS bracos. Nenhum deles sabe responder "nao encontrei". Todos
// sempre devolvem alguma coisa, com confianca, inclusive para consultas cuja
// resposta correta e o silencio.
//
// Ate aqui isso era uma constatacao. Este comando transforma em pergunta de
// pesquisa: A PARTIR DE QUE SCORE UMA CORRESPONDENCIA E REAL?
//
// A resposta pode ser "nao existe limiar". Isso tambem e resultado, e e
// reportado como tal -- inventar um corte plausivel quando as distribuicoes se
// sobrepoem seria fabricar evidencia. O piso de similaridade do braco denso ja
// foi refutado por medicao uma vez neste projeto; nao se repete o erro.
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
		if err := os.WriteFile(*texPath, []byte(evalcorpus.VirgulaDecimal(tabela)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "threshold-eval: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "threshold-eval: %s\n", *texPath)
	}

	printSummary(os.Stderr, analises)
}

// Separabilidade e o diagnostico de um braco.
type Separabilidade struct {
	Arm string
	// Relevantes e Irrelevantes sao as distribuicoes de score.
	Relevantes   []float64
	Irrelevantes []float64
	// LimiarOtimo e o corte que maximiza a separacao (Youden's J). So tem
	// sentido quando Separavel e verdadeiro.
	LimiarOtimo float64
	// J e o indice de Youden no limiar otimo: sensibilidade + especificidade - 1.
	// Vale 1 para separacao perfeita e 0 quando o corte nao e melhor que o acaso.
	J float64
	// Separavel indica se ha algum corte que classifica melhor que o acaso.
	Separavel bool
	// FalsoPositivoComLimiar e a taxa de falso positivo APOS aplicar o limiar,
	// medida sobre as consultas negativas. E o numero que responde se o limiar
	// resolve o problema na pratica.
	FalsoPositivoComLimiar float64
	// FalsoPositivoSemLimiar e sempre 1,000 -- o estado atual, reportado ao
	// lado para a comparacao ser direta.
	FalsoPositivoSemLimiar float64
	// Sobreposicao e a fracao de scores irrelevantes acima do menor score
	// relevante. Zero significa separacao limpa; 1,000 significa que os dois
	// conjuntos ocupam a mesma faixa.
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

	// Scores de itens RELEVANTES vem das consultas positivas; scores de itens
	// IRRELEVANTES vem tanto dos nao-julgados das positivas quanto de TUDO que
	// as consultas negativas devolvem -- nelas, por definicao, nada e relevante.
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

// melhorLimiar procura o corte que maximiza o indice de Youden (J).
//
// J = sensibilidade + especificidade - 1. Vale 1 na separacao perfeita e 0
// quando o corte nao classifica melhor que o acaso. Usa-se J, e nao acuracia,
// porque as classes sao desbalanceadas: com muito mais irrelevantes que
// relevantes, um classificador que rejeita tudo teria acuracia alta e utilidade
// zero.
//
// Devolve separavel=false quando nenhum corte alcanca J > 0 -- ou seja, quando
// as distribuicoes se sobrepoem a ponto de o limiar nao ajudar.
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

// calcularSobreposicao mede a fracao de irrelevantes que pontuam acima do menor
// relevante. E a leitura mais intuitiva do problema: se vale 1,000, todo
// irrelevante pontua tao alto quanto o pior relevante, e nenhum corte separa.
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

// falsoPositivoComLimiar mede a taxa de falso positivo DEPOIS de aplicar o
// corte, sobre as consultas negativas -- aquelas cuja resposta correta e nao
// devolver nada.
//
// E o numero que responde a pergunta pratica: o limiar resolve o problema?
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
	b.WriteString("\\caption{Separabilidade dos scores e efeito de um limiar de corte. ")
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
			a.Arm, a.Sobreposicao, limiar, j, a.FalsoPositivoSemLimiar, a.FalsoPositivoComLimiar)
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
