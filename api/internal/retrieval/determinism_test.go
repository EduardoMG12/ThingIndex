package retrieval

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// Determinismo = mesmo ranking byte a byte, incluindo a ordem dos empates.
// Compara por hash do ranking serializado, que pega reordenacao de empates.

const determinismRepetitions = 30

func hashRanking(results []Result) string {
	var builder strings.Builder
	for i, result := range results {
		fmt.Fprintf(&builder, "%d|%s|%.12f\n", i, result.DocID, result.Score)
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func corpusDeterminismo() []Document {
	return []Document{
		{ID: "contrato-a", Text: "clausula de rescisao contratual com aviso previo de trinta dias"},
		{ID: "contrato-b", Text: "clausula de rescisao contratual com aviso previo de trinta dias"},
		{ID: "contrato-c", Text: "clausula de rescisao contratual com aviso previo de trinta dias"},
		{ID: "edital", Text: "prazo de pagamento em trinta dias apos o atesto da nota fiscal"},
		{ID: "politica", Text: "retencao de registros de acesso por seis meses"},
	}
}

// Os tres documentos identicos forcam empate de score.
func TestBracoRelacionalEDeterministicoEm30Execucoes(t *testing.T) {
	t.Parallel()

	docs := corpusDeterminismo()
	consultas := []string{
		"rescisao contratual",
		"prazo pagamento",
		"registros de acesso",
		"trinta dias",
	}

	for _, consulta := range consultas {
		t.Run(consulta, func(t *testing.T) {
			referencia := hashRanking(SubstringScorer(docs, consulta))

			for execucao := 2; execucao <= determinismRepetitions; execucao++ {
				atual := hashRanking(SubstringScorer(docs, consulta))
				if atual != referencia {
					t.Fatalf("execucao %d divergiu: hash %s != %s (o determinismo afirmado no artigo nao se sustenta)",
						execucao, atual[:16], referencia[:16])
				}
			}
		})
	}
}

func TestBracoBM25EDeterministicoEm30Execucoes(t *testing.T) {
	t.Parallel()

	docs := corpusDeterminismo()
	index := NewBM25Index(docs, DefaultBM25Params())

	for _, consulta := range []string{"rescisao contratual", "trinta dias", "pagamento"} {
		t.Run(consulta, func(t *testing.T) {
			referencia := hashRanking(index.Search(consulta))

			for execucao := 2; execucao <= determinismRepetitions; execucao++ {
				if atual := hashRanking(index.Search(consulta)); atual != referencia {
					t.Fatalf("execucao %d divergiu: hash %s != %s", execucao, atual[:16], referencia[:16])
				}
			}
		})
	}
}

func TestBM25EDeterministicoEntreReconstrucoesDoIndice(t *testing.T) {
	t.Parallel()

	docs := corpusDeterminismo()
	referencia := hashRanking(NewBM25Index(docs, DefaultBM25Params()).Search("rescisao contratual"))

	for execucao := 2; execucao <= determinismRepetitions; execucao++ {
		atual := hashRanking(NewBM25Index(docs, DefaultBM25Params()).Search("rescisao contratual"))
		if atual != referencia {
			t.Fatalf("reconstrucao %d do indice produziu ranking diferente", execucao)
		}
	}
}

func TestFusaoRRFEDeterministicaEm30Execucoes(t *testing.T) {
	t.Parallel()

	docs := corpusDeterminismo()
	index := NewBM25Index(docs, DefaultBM25Params())

	fundir := func(consulta string) []Result {
		return ReciprocalRankFusion([][]Result{
			SubstringScorer(docs, consulta),
			index.Search(consulta),
		}, 60)
	}

	// O RRF agrega num map; sem o desempate por DocID este teste falharia.
	referencia := hashRanking(fundir("rescisao contratual"))
	for execucao := 2; execucao <= determinismRepetitions; execucao++ {
		if atual := hashRanking(fundir("rescisao contratual")); atual != referencia {
			t.Fatalf("execucao %d da fusao divergiu: a agregacao em mapa vazou nao-determinismo", execucao)
		}
	}
}

// Texto identico nao empata no RRF (pontua por posicao). O empate real vem do
// mesmo documento na mesma posicao: com rankings invertidos, os pares
// (i, n-1-i) somam o mesmo score. Sem o desempate por DocID, falha.
func TestFusaoRRFEDeterministicaComScoresEmpatados(t *testing.T) {
	t.Parallel()

	const n = 12
	direto := make([]Result, 0, n)
	inverso := make([]Result, 0, n)
	for i := 0; i < n; i++ {
		direto = append(direto, Result{DocID: fmt.Sprintf("doc-%02d", i)})
		inverso = append(inverso, Result{DocID: fmt.Sprintf("doc-%02d", n-1-i)})
	}

	// garante que o cenario empata de fato
	fundido := ReciprocalRankFusion([][]Result{direto, inverso}, 60)
	distintos := map[float64]bool{}
	for _, result := range fundido {
		distintos[result.Score] = true
	}
	if len(distintos) == len(fundido) {
		t.Fatalf("o cenario nao produziu empate (%d scores distintos em %d itens); o teste nao teria poder de deteccao",
			len(distintos), len(fundido))
	}

	referencia := hashRanking(fundido)
	for execucao := 2; execucao <= determinismRepetitions; execucao++ {
		atual := hashRanking(ReciprocalRankFusion([][]Result{direto, inverso}, 60))
		if atual != referencia {
			t.Fatalf("execucao %d divergiu com scores empatados: o desempate por DocID nao esta agindo", execucao)
		}
	}
}

func TestHashDeRankingDetectaReordenacao(t *testing.T) {
	t.Parallel()

	a := []Result{{DocID: "x", Score: 1}, {DocID: "y", Score: 1}}
	b := []Result{{DocID: "y", Score: 1}, {DocID: "x", Score: 1}}

	if hashRanking(a) == hashRanking(b) {
		t.Fatal("o hash nao distingue ordem; os testes de determinismo nao teriam poder de deteccao")
	}
}

func TestHashDeRankingDetectaMudancaDeScore(t *testing.T) {
	t.Parallel()

	a := []Result{{DocID: "x", Score: 1.0}}
	b := []Result{{DocID: "x", Score: 1.0000000001}}

	if hashRanking(a) == hashRanking(b) {
		t.Fatal("o hash nao distingue score; mudanca silenciosa de pontuacao passaria despercebida")
	}
}
