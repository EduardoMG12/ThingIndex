package retrieval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// Este arquivo implementa o BRACO C do experimento: recuperacao densa por
// similaridade de vetores, que e a abordagem contra a qual o artigo afirma
// nao-inferioridade em auditabilidade e determinismo.
//
// POR QUE O BRACO C E OBRIGATORIO
//
// O artigo sustenta, em quatro passagens, superioridade sobre "RAG vetorial".
// Ate aqui o experimento comparava o sistema com BM25 e com variacoes dele
// mesmo -- nenhuma das quais e o alvo declarado da tese. Sem um braco denso, a
// afirmacao central do trabalho permanece sem evidencia, e a banca so precisa
// perguntar "comparado com o que?" para desmontar a contribuicao.
//
// DUAS VARIANTES, E A DISTINCAO ENTRE ELAS E O ACHADO
//
//	C-exato  varredura completa, produto interno com todos os N documentos.
//	         Deterministico, mas O(N) por consulta.
//	C-ivf    IVF-Flat, o mesmo algoritmo do indice `ivfflat` do pgvector:
//	         particiona o espaco em `lists` centroides por k-means e visita
//	         apenas `probes` particoes. Sublinear, mas APROXIMADO.
//
// Medir apenas o exato produziria uma comparacao desonesta a favor da tese
// deste trabalho: "denso e deterministico igual ao relacional" e verdade no
// exato e falso no que se usa em producao, porque ninguem roda varredura
// completa em colecao grande -- e a propria premissa de escala do RAG vetorial
// e o indice aproximado. Medir as duas variantes e o que permite a afirmacao
// precisa, que e mais forte e mais defensavel do que a generica:
//
//	"o denso EXATO e deterministico, mas paga O(N); o denso APROXIMADO, que e
//	 o praticavel em escala, troca determinismo por latencia -- e essa troca
//	 nao existe na recuperacao relacional."
//
// ORDEM DE INSERCAO E SEED: AS DUAS FONTES DE NAO-DETERMINISMO
//
// O IVF-Flat depende de um k-means, e o k-means depende (a) da inicializacao
// dos centroides e (b) da ordem em que os vetores sao percorridos. Trocar
// qualquer um dos dois muda o particionamento, e um documento que estava na
// particao visitada passa a estar numa que nao e -- some do ranking sem que
// nada no modelo, no corpus ou na consulta tenha mudado.
//
// E por isso que a seed e um PARAMETRO EXPLICITO aqui em vez de um detalhe
// interno: o experimento precisa poder variar a seed de proposito para medir
// quanta variacao ela induz.

// DenseVector e um embedding de um item do corpus.
type DenseVector struct {
	ID     string
	Vector []float64
}

// DenseArtifact e o arquivo de embeddings pre-computados.
//
// Os embeddings sao gerados fora do processo (scripts/embed-corpus.py) e
// versionados. Ver a docstring daquele script para o racional completo; em
// resumo: carregar um modelo de 1,1 GB dentro do harness o tornaria
// nao-hermetico, nao-reproduzivel e contaminaria a medida de latencia com
// tempo de inferencia.
type DenseArtifact struct {
	Manifest  DenseManifest `json:"manifest"`
	Documents []DenseEntry  `json:"documents"`
	Queries   []DenseEntry  `json:"queries"`
}

// DenseManifest registra a procedencia dos vetores.
//
// Sem isso, um arquivo de embeddings e um monte de numeros sem significado
// verificavel -- e o artigo estaria reportando resultado de um modelo que
// ninguem consegue confirmar qual foi.
type DenseManifest struct {
	Model         string `json:"model"`
	Dimension     int    `json:"dimension"`
	Normalized    bool   `json:"normalized"`
	QueryPrefix   string `json:"query_prefix"`
	PassagePrefix string `json:"passage_prefix"`
	CorpusVersion string `json:"corpus_version"`
	Note          string `json:"note,omitempty"`
}

// DenseEntry e o vetor de um item, com o hash do texto que o originou.
type DenseEntry struct {
	ID         string    `json:"id"`
	TextSHA256 string    `json:"text_sha256"`
	Vector     []float64 `json:"vector"`
}

// LoadDenseArtifact le e valida o arquivo de embeddings.
//
// A validacao e severa de proposito. Um vetor com dimensao errada ou norma
// diferente de 1 nao causa erro em tempo de execucao: causa um RANKING
// PLAUSIVEL E ERRADO, que so seria descoberto na arguicao. Barrar na carga e a
// unica forma de garantir que o numero publicado veio de vetores consistentes.
func LoadDenseArtifact(path string) (*DenseArtifact, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ler embeddings: %w", err)
	}

	var artifact DenseArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return nil, fmt.Errorf("decodificar embeddings: %w", err)
	}

	if err := artifact.Validate(); err != nil {
		return nil, err
	}

	return &artifact, nil
}

// normToleranceEpsilon e a folga aceita na verificacao de norma unitaria.
//
// Os vetores sao gerados em float32 e convertidos para float64. A conversao e
// exata, mas a normalizacao foi feita em float32, entao a norma recalculada em
// float64 fica a ~1e-7 de 1. A tolerancia acomoda isso e ainda assim rejeita
// um vetor genuinamente nao normalizado.
const normToleranceEpsilon = 1e-5

// Validate recusa artefato inconsistente.
func (a *DenseArtifact) Validate() error {
	if a.Manifest.Dimension <= 0 {
		return fmt.Errorf("manifesto sem dimensao")
	}
	if len(a.Documents) == 0 {
		return fmt.Errorf("artefato sem vetores de documento")
	}
	if len(a.Queries) == 0 {
		return fmt.Errorf("artefato sem vetores de consulta")
	}

	check := func(kind string, entries []DenseEntry) error {
		seen := map[string]bool{}
		for _, entry := range entries {
			if entry.ID == "" {
				return fmt.Errorf("%s sem id", kind)
			}
			if seen[entry.ID] {
				return fmt.Errorf("%s duplicado: %q", kind, entry.ID)
			}
			seen[entry.ID] = true

			if len(entry.Vector) != a.Manifest.Dimension {
				return fmt.Errorf(
					"%s %q tem dimensao %d, manifesto declara %d",
					kind, entry.ID, len(entry.Vector), a.Manifest.Dimension,
				)
			}

			// A norma so e conferida quando o manifesto AFIRMA normalizacao.
			// O codigo de busca usa produto interno como se fosse cosseno, o
			// que so vale com norma 1; um vetor nao normalizado passando aqui
			// viraria um score inflado por magnitude, favorecendo documentos
			// longos por razao numerica e nao semantica.
			if a.Manifest.Normalized {
				norm := 0.0
				for _, value := range entry.Vector {
					norm += value * value
				}
				norm = math.Sqrt(norm)
				if math.Abs(norm-1) > normToleranceEpsilon {
					return fmt.Errorf(
						"%s %q tem norma %.8f, manifesto declara vetores normalizados",
						kind, entry.ID, norm,
					)
				}
			}
		}
		return nil
	}

	if err := check("documento", a.Documents); err != nil {
		return err
	}
	return check("consulta", a.Queries)
}

// DocumentVectors devolve os vetores de documento em ordem estavel de ID.
//
// A ordenacao explicita importa: o IVF-Flat e sensivel a ordem de insercao
// (ver o cabecalho deste arquivo), entao a ordem precisa ser uma propriedade
// declarada do codigo e nao um acidente da ordem do arquivo JSON.
func (a *DenseArtifact) DocumentVectors() []DenseVector {
	vectors := make([]DenseVector, 0, len(a.Documents))
	for _, entry := range a.Documents {
		vectors = append(vectors, DenseVector{ID: entry.ID, Vector: entry.Vector})
	}
	sort.Slice(vectors, func(i, j int) bool { return vectors[i].ID < vectors[j].ID })
	return vectors
}

// QueryVector devolve o vetor de uma consulta pelo ID.
func (a *DenseArtifact) QueryVector(queryID string) ([]float64, bool) {
	for _, entry := range a.Queries {
		if entry.ID == queryID {
			return entry.Vector, true
		}
	}
	return nil, false
}

// dotProduct e a similaridade de cosseno quando ambos os vetores tem norma 1.
//
// Somar em ordem fixa de indice mantem o resultado bit-a-bit identico entre
// execucoes: soma de ponto flutuante nao e associativa, entao paralelizar ou
// reordenar a soma produziria scores minimamente diferentes -- suficiente para
// trocar a ordem de dois documentos quase empatados e quebrar o determinismo
// que este trabalho afirma ter.
func dotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// DenseIndex e o braco C-exato: varredura completa.
type DenseIndex struct {
	vectors []DenseVector
}

// NewDenseIndex constroi o indice exato.
func NewDenseIndex(vectors []DenseVector) *DenseIndex {
	stable := make([]DenseVector, len(vectors))
	copy(stable, vectors)
	sort.Slice(stable, func(i, j int) bool { return stable[i].ID < stable[j].ID })
	return &DenseIndex{vectors: stable}
}

// DenseSimilarityFloor e o piso de similaridade para um documento entrar no
// ranking. O valor 0 significa SEM CORTE -- todos os documentos entram.
//
// ESTE VALOR FOI MEDIDO, NAO ESCOLHIDO. A primeira versao deste codigo usava
// 0,80, sob a hipotese razoavel de que um limiar separaria relevantes de
// irrelevantes e permitiria ao braco denso responder "nada encontrado" nos
// casos negativos. A medicao sobre o corpus C1 refutou a hipotese
// (scripts/dense-floor-diagnostic.py):
//
//	relevantes    n=18  min=0,7852  mediana=0,8335  max=0,8810
//	irrelevantes  n=62  min=0,7626  mediana=0,8029  max=0,8483
//
// As distribuicoes se SOBREPOEM em quase toda a extensao. Nao existe limiar
// que preserve os relevantes e corte os irrelevantes: em 0,80 o corte ja
// descarta 2 relevantes enquanto 32 irrelevantes sobrevivem; em 0,84 restam 6
// de 18 relevantes.
//
// A causa e conhecida na literatura e se chama ANISOTROPIA do espaco de
// embeddings: modelos treinados com objetivo contrastivo concentram os vetores
// num cone estreito, de modo que a similaridade de cosseno tem piso alto
// mesmo entre textos sem relacao nenhuma. A similaridade e util como ORDEM
// (ranking) e nao como MAGNITUDE (limiar absoluto).
//
// Manter 0,80 teria duas consequencias, ambas inaceitaveis:
//
//  1. PREJUDICARIA A BASELINE. O braco denso e o concorrente da tese deste
//     trabalho. Descartar 2 de 18 relevantes por um limiar mal calibrado
//     produziria numeros piores para ele e melhores para a tese -- vencer um
//     espantalho que eu mesmo construi.
//  2. ESCONDERIA O ACHADO. A incapacidade de responder "nada encontrado" e um
//     resultado do experimento, e um dos mais interessantes, porque atinge os
//     tres bracos por razoes DIFERENTES: no lexical porque qualquer termo em
//     comum gera score positivo; no denso porque a geometria do espaco nao
//     admite limiar. Mascarar isso com um limiar tunado no proprio conjunto de
//     teste seria tuning contra o teste, que e exatamente o vicio que o
//     experimento evita ao fixar k1/b do BM25 e k do RRF.
//
// A constante permanece no codigo, em vez de ser removida, porque o artigo
// reporta a analise de separabilidade: o leitor precisa poder localizar o
// parametro que foi avaliado e zerado por evidencia.
const DenseSimilarityFloor = 0.0

// passesFloor decide se um documento entra no ranking.
//
// Com DenseSimilarityFloor == 0 a funcao aceita TUDO, inclusive similaridade
// negativa. A distincao importa: `score < 0` cortaria os documentos opostos a
// consulta, o que seria um limiar implicito de 0 em vez da ausencia de
// limiar. Como a decisao de nao filtrar foi tomada por medicao (ver
// DenseSimilarityFloor), ela precisa valer sem excecao -- um corte escondido
// em zero reintroduziria pela porta dos fundos exatamente o que a evidencia
// mandou remover.
func passesFloor(score float64) bool {
	if DenseSimilarityFloor <= 0 {
		return true
	}
	return score >= DenseSimilarityFloor
}

// Search devolve os documentos ordenados por similaridade decrescente.
func (idx *DenseIndex) Search(queryVector []float64) []Result {
	if idx == nil || len(idx.vectors) == 0 {
		return nil
	}

	results := make([]Result, 0, len(idx.vectors))
	for _, vector := range idx.vectors {
		score := dotProduct(queryVector, vector.Vector)
		if !passesFloor(score) {
			continue
		}
		results = append(results, Result{DocID: vector.ID, Score: score})
	}

	sortResults(results)
	return results
}
