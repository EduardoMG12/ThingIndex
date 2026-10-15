package retrieval

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Bracos lexicais: A (contagem de substring, copia do scorer de
// document_usecase.go) e B (BM25). O A e reimplementado aqui porque o de
// producao esta acoplado ao repositorio.

type Document struct {
	ID   string
	Text string
}

// Tokenize e a mesma para todos os bracos lexicais. Nao remove acentos.
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

// SubstringScorer: +1 por termo da consulta que aparece como substring. Sem
// IDF, sem normalizacao por comprimento ("pag" casa em "pagamento" e "pagina").
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

// BM25Params: k1 e b ficam nos valores canonicos, sem tuning no conjunto de teste.
type BM25Params struct {
	K1 float64
	B  float64
}

func DefaultBM25Params() BM25Params {
	return BM25Params{K1: 1.2, B: 0.75}
}

type BM25Index struct {
	params        BM25Params
	docIDs        []string
	termFreqs     []map[string]int
	docLengths    []float64
	avgDocLength  float64
	docFreq       map[string]int
	totalDocCount int
}

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

// Search usa idf(t) = ln(1 + (N - df + 0.5) / (df + 0.5)); sem o `1 +` um
// termo em mais da metade do corpus teria IDF negativo.
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

// sortResults desempata por DocID para a ordem ser reproduzivel.
func sortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].DocID < results[j].DocID
	})
}

// FoldDiacritics tira acentos e cedilha (pt/es) depois de baixar a caixa. A
// producao usa antes de tokenizar; os bracos do experimento nao.
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
