// Command dense-determinism mede a estabilidade do ranking dos bracos densos
// variando seed do k-means, ordem de insercao e probes. O braco exato entra
// como controle (tem de dar 100%). Estabilidade = hash SHA-256 do ranking
// completo, que pega reordenacao em qualquer posicao.
//
// Uso:
//
//	go run ./cmd/dense-determinism
//	go run ./cmd/dense-determinism -tex ../docs/article/tables/dense-determinism.tex
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"thing-index/internal/evalcorpus"
	"thing-index/internal/retrieval"
)

func repoPath(parts ...string) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join(parts...)
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(append([]string{root}, parts...)...)
}

// rankingHash inclui o score, nao so a ordem.
func rankingHash(results []retrieval.Result) string {
	var builder strings.Builder
	for _, result := range results {
		fmt.Fprintf(&builder, "%s:%.12f|", result.DocID, result.Score)
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

type stability struct {
	fator string
	// rankings diferentes observados, somados sobre as consultas (ideal: 1 por consulta)
	consultasInstaveis int
	consultasTotal     int
	maxRankingsPorQ    int
	// fracao media de documentos perdidos em relacao a varredura exata
	perdaRecallMedia float64
}

func (s stability) taxaEstabilidade() float64 {
	if s.consultasTotal == 0 {
		return 0
	}
	return float64(s.consultasTotal-s.consultasInstaveis) / float64(s.consultasTotal)
}

func main() {
	corpusPath := flag.String("corpus", repoPath("internal", "evalcorpus", "testdata", "corpus-c1.json"), "corpus")
	embeddingsPath := flag.String("embeddings", repoPath("internal", "evalcorpus", "testdata", "embeddings-c1.json"), "embeddings")
	texPath := flag.String("tex", "", "arquivo LaTeX de saida (vazio = so stdout)")
	repeticoes := flag.Int("repeticoes", 30, "numero de seeds / permutacoes testadas")
	flag.Parse()

	corpus, err := evalcorpus.Load(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dense-determinism: %v\n", err)
		os.Exit(1)
	}

	dense, err := retrieval.LoadDenseArtifact(*embeddingsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dense-determinism: %v\n", err)
		os.Exit(1)
	}

	vectors := dense.DocumentVectors()
	exact := retrieval.NewDenseIndex(vectors)

	type consulta struct {
		id     string
		vector []float64
	}
	consultas := make([]consulta, 0, len(corpus.Queries))
	for _, query := range corpus.Queries {
		vector, ok := dense.QueryVector(query.ID)
		if !ok {
			fmt.Fprintf(os.Stderr, "dense-determinism: consulta %q sem embedding\n", query.ID)
			os.Exit(1)
		}
		consultas = append(consultas, consulta{id: query.ID, vector: vector})
	}

	// Referencia de recall: o que a varredura exata recupera.
	recallExato := map[string]int{}
	for _, current := range consultas {
		recallExato[current.id] = len(exact.Search(current.vector))
	}

	var medidas []stability

	// Controle: o exato repetido tem de dar 100%.
	controle := stability{fator: "C (exato), 30 execuções", consultasTotal: len(consultas)}
	for _, current := range consultas {
		hashes := map[string]bool{}
		for i := 0; i < *repeticoes; i++ {
			hashes[rankingHash(exact.Search(current.vector))] = true
		}
		if len(hashes) > 1 {
			controle.consultasInstaveis++
		}
		if len(hashes) > controle.maxRankingsPorQ {
			controle.maxRankingsPorQ = len(hashes)
		}
	}
	medidas = append(medidas, controle)

	// Fator 1: seed do k-means.
	seedFactor := stability{
		fator:          fmt.Sprintf("C$'$ (IVF), %d sementes", *repeticoes),
		consultasTotal: len(consultas),
	}
	perdaAcumulada := 0.0
	for _, current := range consultas {
		hashes := map[string]bool{}
		recuperadosTotal := 0
		for seed := 0; seed < *repeticoes; seed++ {
			params := retrieval.DefaultIVFParams(len(vectors))
			params.Seed = int64(seed)
			results := retrieval.NewIVFIndex(vectors, params).Search(current.vector)
			hashes[rankingHash(results)] = true
			recuperadosTotal += len(results)
		}
		if len(hashes) > 1 {
			seedFactor.consultasInstaveis++
		}
		if len(hashes) > seedFactor.maxRankingsPorQ {
			seedFactor.maxRankingsPorQ = len(hashes)
		}

		if recallExato[current.id] > 0 {
			recuperadoMedio := float64(recuperadosTotal) / float64(*repeticoes)
			perdaAcumulada += 1 - recuperadoMedio/float64(recallExato[current.id])
		}
	}
	seedFactor.perdaRecallMedia = perdaAcumulada / float64(len(consultas))
	medidas = append(medidas, seedFactor)

	// Fator 2: ordem de insercao, seed fixa. Mesmos documentos subidos em
	// ordem diferente geram indices diferentes.
	orderFactor := stability{
		fator:          fmt.Sprintf("C$'$ (IVF), %d ordens de inserção", *repeticoes),
		consultasTotal: len(consultas),
	}
	source := rand.New(rand.NewSource(2026))
	permutacoes := make([][]int, 0, *repeticoes)
	for i := 0; i < *repeticoes; i++ {
		permutacoes = append(permutacoes, source.Perm(len(vectors)))
	}
	for _, current := range consultas {
		hashes := map[string]bool{}
		for _, perm := range permutacoes {
			params := retrieval.DefaultIVFParams(len(vectors))
			params.Seed = 42
			params.InsertPerm = perm
			hashes[rankingHash(retrieval.NewIVFIndex(vectors, params).Search(current.vector))] = true
		}
		if len(hashes) > 1 {
			orderFactor.consultasInstaveis++
		}
		if len(hashes) > orderFactor.maxRankingsPorQ {
			orderFactor.maxRankingsPorQ = len(hashes)
		}
	}
	medidas = append(medidas, orderFactor)

	// Fator 3: probes, seed e ordem fixas.
	probeFactor := stability{fator: "C$'$ (IVF), \\texttt{probes} de 1 a \\texttt{lists}", consultasTotal: len(consultas)}
	maxProbes := retrieval.DefaultIVFParams(len(vectors)).Lists
	for _, current := range consultas {
		hashes := map[string]bool{}
		for probes := 1; probes <= maxProbes; probes++ {
			params := retrieval.DefaultIVFParams(len(vectors))
			params.Seed = 42
			params.Probes = probes
			hashes[rankingHash(retrieval.NewIVFIndex(vectors, params).Search(current.vector))] = true
		}
		if len(hashes) > 1 {
			probeFactor.consultasInstaveis++
		}
		if len(hashes) > probeFactor.maxRankingsPorQ {
			probeFactor.maxRankingsPorQ = len(hashes)
		}
	}
	medidas = append(medidas, probeFactor)

	fmt.Printf("corpus: %d documentos, %d consultas | IVF lists=%d probes=%d (default do pgvector)\n\n",
		len(vectors), len(consultas), maxProbes, retrieval.DefaultIVFParams(len(vectors)).Probes)

	fmt.Printf("%-52s %12s %14s %14s\n", "fator", "estaveis", "max rankings", "perda recall")
	for _, medida := range medidas {
		perda := "--"
		if medida.perdaRecallMedia > 0 {
			perda = fmt.Sprintf("%.1f%%", 100*medida.perdaRecallMedia)
		}
		fmt.Printf("%-52s %10.1f%% %14d %14s\n",
			medida.fator, 100*medida.taxaEstabilidade(), medida.maxRankingsPorQ, perda)
	}

	if *texPath != "" {
		if err := writeTeX(*texPath, medidas, len(vectors), len(consultas)); err != nil {
			fmt.Fprintf(os.Stderr, "dense-determinism: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\ndense-determinism: %s\n", *texPath)
	}
}

func writeTeX(path string, medidas []stability, documentos, consultas int) error {
	var builder strings.Builder

	builder.WriteString("% GERADO POR cmd/dense-determinism -- nao edite a mao.\n")
	builder.WriteString("\\begin{table}[htb]\n\\centering\n")
	fmt.Fprintf(&builder,
		"\\caption{Estabilidade do ranking do braço denso sob variação de fatores que não são propriedades da consulta. "+
			"Corpus C1 (%d documentos, %d consultas). Uma ordenação é considerada estável quando o \\emph{hash} SHA-256 da lista "+
			"completa (identificador e escore) é idêntico em todas as repetições.}\n",
		documentos, consultas)
	builder.WriteString("\\label{tab:dense-determinism}\n")
	builder.WriteString("\\small\n\\begin{tabular}{lccc}\n\\hline\n")
	builder.WriteString("Fator variado & Estáveis & Ordenações distintas & Perda de revocação \\\\\n\\hline\n")

	for _, medida := range medidas {
		perda := "--"
		if medida.perdaRecallMedia > 0 {
			perda = fmt.Sprintf("%.1f\\%%", 100*medida.perdaRecallMedia)
		}
		fmt.Fprintf(&builder, "%s & %.0f\\%% & %d & %s \\\\\n",
			escapeTeX(medida.fator), 100*medida.taxaEstabilidade(), medida.maxRankingsPorQ, perda)
	}

	builder.WriteString("\\hline\n\\end{tabular}\n\\end{table}\n")

	return os.WriteFile(path, []byte(evalcorpus.Tabela(builder.String())), 0o644)
}

func escapeTeX(text string) string {
	replacer := strings.NewReplacer("_", "\\_", "%", "\\%", "&", "\\&", "#", "\\#")
	return replacer.Replace(text)
}
