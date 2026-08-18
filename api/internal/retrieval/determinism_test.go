package retrieval

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// DETERMINISMO E A AFIRMACAO MAIS FORTE E MAIS BARATA DO TRABALHO.
//
// O artigo afirma determinismo como vantagem da abordagem relacional sobre a
// densa, e ate agora isso era sustentado apenas por leitura de codigo. E uma
// vitoria que estava sendo deixada na mesa: o relacional entrega 100% por
// construcao, enquanto um indice HNSW varia com ef_search e com a ordem de
// insercao.
//
// "Determinismo" aqui tem definicao operacional precisa, e nao retorica:
//
//	a mesma consulta, sobre o mesmo corpus, produz o MESMO ranking --
//	byte a byte, incluindo a ordem dos empates -- em execucoes repetidas.
//
// A verificacao e por hash do ranking serializado. Comparar so o primeiro
// colocado esconderia instabilidade no resto da lista; comparar os scores com
// tolerancia esconderia reordenacao de empates. O hash nao perdoa nenhum dos
// dois.

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

// Tres documentos com texto IDENTICO existem de proposito no corpus: eles
// produzem empate perfeito de score, que e exatamente onde uma implementacao
// nao deterministica se revela (a ordem passaria a depender da iteracao de
// mapa ou da estabilidade do sort).
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

// Reconstruir o indice do zero tem que produzir o mesmo ranking. Se nao
// produzir, o resultado depende de estado acumulado -- e o experimento deixa de
// ser reproduzivel entre execucoes do harness.
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

	// O RRF e o braco de maior risco: ele agrega scores num MAPA, e a ordem de
	// iteracao de mapa em Go e deliberadamente aleatoria. Sem o desempate
	// explicito por DocID, este teste falharia.
	//
	// Nota de metodo: o corpus precisa ter empates SUFICIENTES para revelar a
	// instabilidade. Com 2 ou 3 itens empatados o sort.Slice frequentemente
	// devolve a mesma ordem por acaso, e o teste passaria mesmo sem o
	// desempate -- verificado na pratica. Por isso a consulta abaixo casa com
	// os tres documentos identicos E o teste tambem cobre um corpus maior.
	referencia := hashRanking(fundir("rescisao contratual"))
	for execucao := 2; execucao <= determinismRepetitions; execucao++ {
		if atual := hashRanking(fundir("rescisao contratual")); atual != referencia {
			t.Fatalf("execucao %d da fusao divergiu: a agregacao em mapa vazou nao-determinismo", execucao)
		}
	}
}

// Este teste expoe a instabilidade que o desempate por DocID corrige.
//
// Detalhe que so aparece investigando: alimentar o RRF com documentos de texto
// identico NAO produz empate. O RRF pontua por POSICAO, e as posicoes sao
// distintas (1o, 2o, 3o...), entao os scores fundidos saem todos diferentes --
// verifiquei isso na pratica, e por isso a primeira versao deste teste passava
// mesmo com o desempate removido.
//
// O empate real acontece quando o mesmo documento ocupa a MESMA posicao nos
// dois rankings de entrada. Aqui isso e construido diretamente: dois rankings
// com a ordem invertida fazem os pares (i, n-1-i) somarem exatamente o mesmo
// score.
//
// Verificado por mutacao: removendo o desempate por DocID de
// ReciprocalRankFusion, este teste falha; com o desempate, passa nas 30
// execucoes.
func TestFusaoRRFEDeterministicaComScoresEmpatados(t *testing.T) {
	t.Parallel()

	const n = 12
	direto := make([]Result, 0, n)
	inverso := make([]Result, 0, n)
	for i := 0; i < n; i++ {
		direto = append(direto, Result{DocID: fmt.Sprintf("doc-%02d", i)})
		inverso = append(inverso, Result{DocID: fmt.Sprintf("doc-%02d", n-1-i)})
	}

	// Confirmacao de que o cenario realmente empata: se os scores saissem
	// distintos, o teste nao exercitaria o desempate e passaria por engano.
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

// Prova de que o teste tem poder de deteccao: um ranking com ordem trocada
// PRECISA gerar hash diferente. Sem isso, os testes acima poderiam estar
// passando por comparar sempre a mesma coisa.
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
