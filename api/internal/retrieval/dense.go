package retrieval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// Recuperacao densa (braco C) em duas variantes: varredura exata (deterministica,
// O(N)) e IVF-Flat como o ivfflat do pgvector (sublinear, aproximada). O IVF
// depende da seed e da ordem de insercao do k-means; por isso a seed e parametro.

type DenseVector struct {
	ID     string
	Vector []float64
}

// DenseArtifact sao os embeddings pre-computados fora do processo, para nao
// carregar o modelo no harness nem misturar inferencia na latencia medida.
type DenseArtifact struct {
	Manifest  DenseManifest `json:"manifest"`
	Documents []DenseEntry  `json:"documents"`
	Queries   []DenseEntry  `json:"queries"`
}

// DenseManifest registra de qual modelo/corpus vieram os vetores.
type DenseManifest struct {
	Model         string `json:"model"`
	Dimension     int    `json:"dimension"`
	Normalized    bool   `json:"normalized"`
	QueryPrefix   string `json:"query_prefix"`
	PassagePrefix string `json:"passage_prefix"`
	CorpusVersion string `json:"corpus_version"`
	Note          string `json:"note,omitempty"`
}

type DenseEntry struct {
	ID         string    `json:"id"`
	TextSHA256 string    `json:"text_sha256"`
	Vector     []float64 `json:"vector"`
}

// LoadDenseArtifact valida na carga: dimensao ou norma errada nao da erro,
// da um ranking plausivel e errado.
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

// Normalizados em float32: a norma recalculada em float64 fica a ~1e-7 de 1.
const normToleranceEpsilon = 1e-5

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

			// A busca usa produto interno como cosseno, o que so vale com norma 1.
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

// DocumentVectors ordena por ID: o IVF-Flat e sensivel a ordem de insercao.
func (a *DenseArtifact) DocumentVectors() []DenseVector {
	vectors := make([]DenseVector, 0, len(a.Documents))
	for _, entry := range a.Documents {
		vectors = append(vectors, DenseVector{ID: entry.ID, Vector: entry.Vector})
	}
	sort.Slice(vectors, func(i, j int) bool { return vectors[i].ID < vectors[j].ID })
	return vectors
}

func (a *DenseArtifact) QueryVector(queryID string) ([]float64, bool) {
	for _, entry := range a.Queries {
		if entry.ID == queryID {
			return entry.Vector, true
		}
	}
	return nil, false
}

// dotProduct soma em ordem fixa: soma de float nao e associativa, e reordenar
// pode trocar dois documentos quase empatados.
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

// DenseIndex e a variante exata (varredura completa).
type DenseIndex struct {
	vectors []DenseVector
}

func NewDenseIndex(vectors []DenseVector) *DenseIndex {
	stable := make([]DenseVector, len(vectors))
	copy(stable, vectors)
	sort.Slice(stable, func(i, j int) bool { return stable[i].ID < stable[j].ID })
	return &DenseIndex{vectors: stable}
}

// DenseSimilarityFloor = 0 (sem corte). Medido no corpus C1: as similaridades
// de relevantes (0,785-0,881) e irrelevantes (0,763-0,848) se sobrepoem, entao
// nenhum limiar separa os dois (anisotropia do espaco de embeddings).
const DenseSimilarityFloor = 0.0

// passesFloor com piso 0 aceita inclusive score negativo; cortar em zero seria
// um limiar implicito.
func passesFloor(score float64) bool {
	if DenseSimilarityFloor <= 0 {
		return true
	}
	return score >= DenseSimilarityFloor
}

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
