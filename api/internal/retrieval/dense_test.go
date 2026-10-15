package retrieval

import (
	"math"
	"testing"
)

// Valores esperados calculados a mao, nao copiados da saida do codigo.

// unitVector: com norma 1 o produto interno e o cosseno.
func unitVector(dim, axis int) []float64 {
	vector := make([]float64, dim)
	vector[axis] = 1
	return vector
}

func TestDotProductIsCosineForUnitVectors(t *testing.T) {
	// Vetores ortogonais: cosseno 0. Vetor consigo mesmo: cosseno 1.
	a := unitVector(3, 0)
	b := unitVector(3, 1)

	if got := dotProduct(a, b); got != 0 {
		t.Fatalf("vetores ortogonais deveriam dar 0, deu %v", got)
	}
	if got := dotProduct(a, a); got != 1 {
		t.Fatalf("vetor consigo mesmo deveria dar 1, deu %v", got)
	}

	// 45 graus entre eixos: cosseno = sqrt(2)/2 ~ 0,7071.
	diagonal := []float64{math.Sqrt2 / 2, math.Sqrt2 / 2, 0}
	got := dotProduct(a, diagonal)
	if math.Abs(got-math.Sqrt2/2) > 1e-12 {
		t.Fatalf("45 graus deveria dar %.6f, deu %.6f", math.Sqrt2/2, got)
	}
}

func TestDotProductRejectsDimensionMismatch(t *testing.T) {
	// o perigo e somar so o prefixo comum e devolver um score plausivel
	if got := dotProduct([]float64{1, 0}, []float64{1, 0, 0}); got != 0 {
		t.Fatalf("dimensoes incompativeis deveriam dar 0, deu %v", got)
	}
}

func TestDenseIndexRanksBySimilarity(t *testing.T) {
	// quanto mais alinhado ao eixo 0, maior o score
	vectors := []DenseVector{
		{ID: "alinhado", Vector: []float64{1, 0, 0}},
		{ID: "meio", Vector: []float64{0.9, math.Sqrt(1 - 0.81), 0}},
		{ID: "distante", Vector: []float64{0.82, math.Sqrt(1 - 0.6724), 0}},
	}

	index := NewDenseIndex(vectors)
	results := index.Search([]float64{1, 0, 0})

	if len(results) != 3 {
		t.Fatalf("esperava 3 resultados acima do piso, veio %d", len(results))
	}

	expectedOrder := []string{"alinhado", "meio", "distante"}
	for i, want := range expectedOrder {
		if results[i].DocID != want {
			t.Fatalf("posicao %d: esperava %q, veio %q", i, want, results[i].DocID)
		}
	}

	// O score do primeiro e o cosseno consigo mesmo = 1, exato.
	if math.Abs(results[0].Score-1) > 1e-12 {
		t.Fatalf("score do alinhado deveria ser 1, veio %.8f", results[0].Score)
	}
	// O do segundo e 0,9 por construcao.
	if math.Abs(results[1].Score-0.9) > 1e-12 {
		t.Fatalf("score do meio deveria ser 0,9, veio %.8f", results[1].Score)
	}
}

func TestDenseIndexReturnsEveryDocumentByDefault(t *testing.T) {
	// Sem piso de similaridade: no corpus C1 relevantes e irrelevantes se
	// sobrepoem (ver DenseSimilarityFloor).
	vectors := []DenseVector{
		{ID: "alinhado", Vector: []float64{1, 0, 0}},
		{ID: "ortogonal", Vector: []float64{0, 1, 0}},
		{ID: "oposto", Vector: []float64{-1, 0, 0}},
	}

	results := NewDenseIndex(vectors).Search([]float64{1, 0, 0})

	if len(results) != 3 {
		t.Fatalf("sem piso, os 3 documentos deveriam entrar no ranking; entraram %d", len(results))
	}

	// inclusive o de similaridade negativa
	if results[2].DocID != "oposto" {
		t.Fatalf("o documento oposto deveria vir por ultimo, veio %q", results[2].DocID)
	}
	if results[2].Score >= 0 {
		t.Fatalf("o documento oposto deveria ter score negativo, veio %.4f", results[2].Score)
	}

	if DenseSimilarityFloor != 0 {
		t.Fatalf("DenseSimilarityFloor deveria ser 0 (decisao medida, ver dense.go); esta %v", DenseSimilarityFloor)
	}
}

func TestDenseIndexIsDeterministicAcrossRuns(t *testing.T) {
	// O exato precisa ser estavel para a variacao do IVF ser atribuida ao indice.
	vectors := []DenseVector{
		{ID: "b", Vector: []float64{0.95, math.Sqrt(1 - 0.9025), 0}},
		{ID: "a", Vector: []float64{0.95, math.Sqrt(1 - 0.9025), 0}},
		{ID: "c", Vector: []float64{1, 0, 0}},
	}

	index := NewDenseIndex(vectors)
	first := index.Search([]float64{1, 0, 0})

	for run := 0; run < 50; run++ {
		current := index.Search([]float64{1, 0, 0})
		if len(current) != len(first) {
			t.Fatalf("execucao %d devolveu %d itens, a primeira devolveu %d", run, len(current), len(first))
		}
		for i := range current {
			if current[i].DocID != first[i].DocID {
				t.Fatalf("execucao %d divergiu na posicao %d: %q vs %q",
					run, i, current[i].DocID, first[i].DocID)
			}
		}
	}

	// "a" e "b" tem vetores identicos; confere que o cenario empata de fato,
	// senao o teste passaria mesmo sem o desempate por DocID.
	var scoreA, scoreB float64
	for _, result := range first {
		switch result.DocID {
		case "a":
			scoreA = result.Score
		case "b":
			scoreB = result.Score
		}
	}
	if scoreA != scoreB {
		t.Fatalf("o cenario deveria produzir empate entre a e b, mas deu %.8f vs %.8f", scoreA, scoreB)
	}
	if first[1].DocID != "a" || first[2].DocID != "b" {
		t.Fatalf("empate deveria ser desempatado por DocID (a antes de b), veio %q depois %q",
			first[1].DocID, first[2].DocID)
	}
}

func TestDenseArtifactValidateRejectsWrongDimension(t *testing.T) {
	artifact := &DenseArtifact{
		Manifest:  DenseManifest{Dimension: 3, Normalized: true},
		Documents: []DenseEntry{{ID: "d1", Vector: []float64{1, 0}}},
		Queries:   []DenseEntry{{ID: "q1", Vector: []float64{1, 0, 0}}},
	}

	err := artifact.Validate()
	if err == nil {
		t.Fatal("dimensao divergente do manifesto deveria ser recusada")
	}
}

func TestDenseArtifactValidateRejectsUnnormalizedVector(t *testing.T) {
	// norma 2 num artefato declarado normalizado: score dobrado por magnitude
	artifact := &DenseArtifact{
		Manifest:  DenseManifest{Dimension: 3, Normalized: true},
		Documents: []DenseEntry{{ID: "d1", Vector: []float64{2, 0, 0}}},
		Queries:   []DenseEntry{{ID: "q1", Vector: []float64{1, 0, 0}}},
	}

	if err := artifact.Validate(); err == nil {
		t.Fatal("vetor nao normalizado deveria ser recusado")
	}
}

func TestDenseArtifactValidateAcceptsFloat32RoundTrip(t *testing.T) {
	// normalizado em float32: norma a ~1e-7 de 1 tem de passar
	nearlyOne := []float64{0.9999999, 0, 0}
	artifact := &DenseArtifact{
		Manifest:  DenseManifest{Dimension: 3, Normalized: true},
		Documents: []DenseEntry{{ID: "d1", Vector: nearlyOne}},
		Queries:   []DenseEntry{{ID: "q1", Vector: nearlyOne}},
	}

	if err := artifact.Validate(); err != nil {
		t.Fatalf("erro de arredondamento de float32 deveria ser aceito: %v", err)
	}
}

func TestDenseArtifactValidateRejectsDuplicateID(t *testing.T) {
	artifact := &DenseArtifact{
		Manifest: DenseManifest{Dimension: 3, Normalized: false},
		Documents: []DenseEntry{
			{ID: "d1", Vector: []float64{1, 0, 0}},
			{ID: "d1", Vector: []float64{0, 1, 0}},
		},
		Queries: []DenseEntry{{ID: "q1", Vector: []float64{1, 0, 0}}},
	}

	if err := artifact.Validate(); err == nil {
		t.Fatal("id duplicado deveria ser recusado")
	}
}
