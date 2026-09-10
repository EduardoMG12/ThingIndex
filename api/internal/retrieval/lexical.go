package retrieval

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Este arquivo implementa os bracos LEXICAIS do experimento comparativo.
//
// Braco A (relacional): o scorer atual do ThingIndex, contagem de substring.
// Braco B (BM25):       Robertson & Zaragoza (2009), a baseline obrigatoria.
//
// Por que o BM25 e obrigatorio: o artigo CITA Robertson & Zaragoza e depois nao
// usa BM25 em lugar nenhum. Comparar a abordagem relacional apenas com
// variacoes dela mesma nao sustenta nenhuma afirmacao de superioridade.
//
// Por que reimplementar o braco A aqui em vez de chamar o codigo de producao:
// o scorer de producao esta acoplado ao repositorio e ao modelo de dados. Para
// medir os quatro bracos sobre o MESMO corpus com a MESMA tokenizacao,
// precisamos deles isolados. A formula do braco A e copia fiel de
// document_usecase.go -- e justamente por ser tao simples que ela e o objeto de
// estudo.

// Document e uma unidade indexavel do experimento.
type Document struct {
	ID   string
	Text string
}

// Tokenize normaliza e quebra o texto em termos.
//
// A MESMA tokenizacao e usada por todos os bracos lexicais de proposito: se
// cada braco normalizasse de um jeito, a comparacao mediria diferenca de
// pre-processamento em vez de diferenca de modelo de recuperacao.
//
// Nao remove acentos: em portugues "e" e "é" sao palavras diferentes, e
// remover acento introduz colisoes que favoreceriam artificialmente o casamento
// lexical -- justamente o que o experimento quer medir sem viés.
func Tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			tokens = append(tokens, field)
		}
	}
	return tokens
}

// SubstringScorer reproduz o braco A: o scorer atual do ThingIndex.
//
// A formula e a de document_usecase.go: para cada termo da consulta, +1 se o
// termo aparece como substring no texto normalizado. Sem IDF, sem normalizacao
// por comprimento, sem saturacao de frequencia.
//
// As tres ausencias importam e sao o achado que o experimento documenta:
//
//   - Sem IDF, um termo comum ("de", "o") pesa igual a um termo discriminativo.
//   - Sem normalizacao por comprimento, documento longo vence por ser longo.
//   - Por ser substring e nao token, "pag" casa dentro de "pagamento" e dentro
//     de "pagina" -- casamentos que o usuario nao pediu.
//
// Reproduzi-lo fielmente e o ponto: o experimento precisa medir o sistema como
// ele e, nao uma versao idealizada dele.
func SubstringScorer(docs []Document, query string) []Result {
	terms := Tokenize(query)

	results := make([]Result, 0, len(docs))
	for _, doc := range docs {
		normalized := strings.ToLower(doc.Text)
		score := 0.0
		for _, term := range terms {
			if strings.Contains(normalized, term) {
				score++
			}
		}
		if score > 0 {
			results = append(results, Result{DocID: doc.ID, Score: score})
		}
	}

	sortResults(results)
	return results
}

// BM25Params agrupa os hiperparametros do BM25.
//
// Os defaults k1=1.2 e b=0.75 sao os do artigo original e os usados em TREC.
// Mante-los fixos e deliberado: ajusta-los contra o proprio conjunto de teste
// transformaria a baseline num modelo tunado e invalidaria a comparacao -- seria
// comparar o ThingIndex sem tuning contra um BM25 com tuning.
type BM25Params struct {
	K1 float64
	B  float64
}

// DefaultBM25Params devolve os parametros canonicos.
func DefaultBM25Params() BM25Params {
	return BM25Params{K1: 1.2, B: 0.75}
}

// BM25Index e um indice invertido com as estatisticas necessarias ao BM25.
type BM25Index struct {
	params        BM25Params
	docIDs        []string
	termFreqs     []map[string]int
	docLengths    []float64
	avgDocLength  float64
	docFreq       map[string]int
	totalDocCount int
}

// NewBM25Index constroi o indice a partir do corpus.
func NewBM25Index(docs []Document, params BM25Params) *BM25Index {
	if params.K1 <= 0 {
		params.K1 = 1.2
	}
	if params.B < 0 || params.B > 1 {
		params.B = 0.75
	}

	index := &BM25Index{
		params:        params,
		docIDs:        make([]string, 0, len(docs)),
		termFreqs:     make([]map[string]int, 0, len(docs)),
		docLengths:    make([]float64, 0, len(docs)),
		docFreq:       map[string]int{},
		totalDocCount: len(docs),
	}

	totalLength := 0.0
	for _, doc := range docs {
		tokens := Tokenize(doc.Text)

		freqs := map[string]int{}
		for _, token := range tokens {
			freqs[token]++
		}
		for term := range freqs {
			index.docFreq[term]++
		}

		index.docIDs = append(index.docIDs, doc.ID)
		index.termFreqs = append(index.termFreqs, freqs)
		index.docLengths = append(index.docLengths, float64(len(tokens)))
		totalLength += float64(len(tokens))
	}

	if len(docs) > 0 {
		index.avgDocLength = totalLength / float64(len(docs))
	}

	return index
}

// Search devolve os documentos ordenados por score BM25.
//
// Usa o IDF com suavizacao de Robertson:
//
//	idf(t) = ln(1 + (N - df + 0.5) / (df + 0.5))
//
// O `1 +` externo e o que garante IDF sempre positivo. Sem ele, um termo
// presente em mais da metade do corpus recebe IDF NEGATIVO e passa a PUNIR os
// documentos que o contem -- um documento que casa com todos os termos da
// consulta poderia ficar atras de um que nao casa com nenhum.
func (idx *BM25Index) Search(query string) []Result {
	if idx == nil || idx.totalDocCount == 0 {
		return nil
	}

	terms := Tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	results := make([]Result, 0, idx.totalDocCount)
	for i, docID := range idx.docIDs {
		score := 0.0
		docLength := idx.docLengths[i]

		for _, term := range terms {
			freq := float64(idx.termFreqs[i][term])
			if freq == 0 {
				continue
			}

			df := float64(idx.docFreq[term])
			idf := math.Log(1 + (float64(idx.totalDocCount)-df+0.5)/(df+0.5))

			norm := 1 - idx.params.B + idx.params.B*(docLength/idx.avgDocLength)
			score += idf * (freq * (idx.params.K1 + 1)) / (freq + idx.params.K1*norm)
		}

		if score > 0 {
			results = append(results, Result{DocID: docID, Score: score})
		}
	}

	sortResults(results)
	return results
}

// sortResults ordena por score decrescente, desempatando por DocID.
//
// O desempate explicito nao e cosmetico: sem ele a ordem de itens com score
// igual depende da ordem de iteracao, e o experimento deixa de ser
// reproduzivel -- justamente a propriedade que o trabalho afirma ter.
func sortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].DocID < results[j].DocID
	})
}

// FoldDiacritics tira acentos e cedilha das vogais e consoantes do portugues
// e do espanhol, depois de baixar a caixa. E o que a producao usa antes de
// tokenizar: o usuario escreve "horaria" e o documento traz "horária".
//
// O experimento (bracos A a D) NAO dobra acentos de proposito: ele mede o
// sistema como foi publicado. A producao e um braco B com esta dobra a mais,
// e a diferenca fica declarada aqui.
func FoldDiacritics(text string) string {
	return diacriticsFolder.Replace(strings.ToLower(text))
}

var diacriticsFolder = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)
