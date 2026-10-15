package evalcorpus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Os embeddings do braco denso sao gerados fora do Go (scripts/embed-corpus.py)
// e versionados. Estes testes falham se o corpus mudar sem regerar o artefato.
// Compara hash do texto, nao contagem, para pegar edicao.

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type embeddingEntry struct {
	ID         string    `json:"id"`
	TextSHA256 string    `json:"text_sha256"`
	Vector     []float64 `json:"vector"`
}

type embeddingArtifact struct {
	Manifest struct {
		Model         string `json:"model"`
		Dimension     int    `json:"dimension"`
		Normalized    bool   `json:"normalized"`
		CorpusVersion string `json:"corpus_version"`
	} `json:"manifest"`
	Documents []embeddingEntry `json:"documents"`
	Queries   []embeddingEntry `json:"queries"`
}

func loadArtifact(t *testing.T) (*Corpus, *embeddingArtifact) {
	t.Helper()

	corpusPath := filepath.Join("testdata", "corpus-c1.json")
	corpus, err := Load(corpusPath)
	if err != nil {
		t.Fatalf("carregar corpus: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "embeddings-c1.json"))
	if err != nil {
		t.Fatalf("carregar embeddings: %v\n"+
			"gere com: python3 scripts/embed-corpus.py --corpus api/%s --out api/internal/evalcorpus/testdata/embeddings-c1.json",
			err, corpusPath)
	}

	var artifact embeddingArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("decodificar embeddings: %v", err)
	}

	return corpus, &artifact
}

func TestEmbeddingsCobremTodoOCorpus(t *testing.T) {
	corpus, artifact := loadArtifact(t)

	documentos := map[string]bool{}
	for _, entry := range artifact.Documents {
		documentos[entry.ID] = true
	}
	consultas := map[string]bool{}
	for _, entry := range artifact.Queries {
		consultas[entry.ID] = true
	}

	for _, doc := range corpus.Documents {
		if !documentos[doc.ID] {
			t.Errorf("documento %q nao tem embedding: regenere o artefato", doc.ID)
		}
	}
	for _, query := range corpus.Queries {
		if !consultas[query.ID] {
			t.Errorf("consulta %q nao tem embedding: regenere o artefato", query.ID)
		}
	}

	// Embedding orfao: item removido do corpus sem regerar o artefato.
	if len(artifact.Documents) != len(corpus.Documents) {
		t.Errorf("artefato tem %d documentos, corpus tem %d",
			len(artifact.Documents), len(corpus.Documents))
	}
	if len(artifact.Queries) != len(corpus.Queries) {
		t.Errorf("artefato tem %d consultas, corpus tem %d",
			len(artifact.Queries), len(corpus.Queries))
	}
}

func TestEmbeddingsCorrespondemAoTextoAtualDoCorpus(t *testing.T) {
	// Mesmo ID, texto alterado depois que o vetor foi gerado.
	corpus, artifact := loadArtifact(t)

	hashDocumento := map[string]string{}
	for _, entry := range artifact.Documents {
		hashDocumento[entry.ID] = entry.TextSHA256
	}
	hashConsulta := map[string]string{}
	for _, entry := range artifact.Queries {
		hashConsulta[entry.ID] = entry.TextSHA256
	}

	for _, doc := range corpus.Documents {
		// Tem de bater com build_passage (scripts/embed-corpus.py) e buildArms
		// (cmd/retrieval-eval), senao os bracos indexam textos diferentes.
		esperado := hashText(doc.Title + "\n" + doc.Content)
		if hashDocumento[doc.ID] != esperado {
			t.Errorf("documento %q mudou depois da geracao dos embeddings\n"+
				"  artefato: %s\n  corpus:   %s\n"+
				"  regenere: python3 scripts/embed-corpus.py --corpus api/internal/evalcorpus/testdata/corpus-c1.json --out api/internal/evalcorpus/testdata/embeddings-c1.json",
				doc.ID, hashDocumento[doc.ID], esperado)
		}
	}

	for _, query := range corpus.Queries {
		esperado := hashText(query.Text)
		if hashConsulta[query.ID] != esperado {
			t.Errorf("consulta %q mudou depois da geracao dos embeddings\n"+
				"  artefato: %s\n  corpus:   %s\n"+
				"  regenere: python3 scripts/embed-corpus.py --corpus api/internal/evalcorpus/testdata/corpus-c1.json --out api/internal/evalcorpus/testdata/embeddings-c1.json",
				query.ID, hashConsulta[query.ID], esperado)
		}
	}
}

func TestManifestoDosEmbeddingsDeclaraOModeloReportadoNoArtigo(t *testing.T) {
	// Trava modelo e dimensao do artefato.
	_, artifact := loadArtifact(t)

	const modeloEsperado = "intfloat/multilingual-e5-base"
	const dimensaoEsperada = 768

	if artifact.Manifest.Model != modeloEsperado {
		t.Errorf("o artigo reporta %q, o artefato declara %q",
			modeloEsperado, artifact.Manifest.Model)
	}
	if artifact.Manifest.Dimension != dimensaoEsperada {
		t.Errorf("o artigo reporta dimensao %d, o artefato declara %d",
			dimensaoEsperada, artifact.Manifest.Dimension)
	}
	if !artifact.Manifest.Normalized {
		t.Error("o artefato precisa declarar vetores normalizados: " +
			"a busca usa produto interno como se fosse cosseno, o que so vale com norma 1")
	}
}

func TestVetoresTemADimensaoDeclarada(t *testing.T) {
	// Dimensao errada nao da panic: dotProduct devolve 0 e o ranking sai so do
	// desempate por ID.
	_, artifact := loadArtifact(t)

	for _, grupo := range []struct {
		nome    string
		entries []embeddingEntry
	}{
		{"documento", artifact.Documents},
		{"consulta", artifact.Queries},
	} {
		for _, entry := range grupo.entries {
			if len(entry.Vector) != artifact.Manifest.Dimension {
				t.Errorf("%s %q tem %d dimensoes, o manifesto declara %d",
					grupo.nome, entry.ID, len(entry.Vector), artifact.Manifest.Dimension)
			}
		}
	}
}
