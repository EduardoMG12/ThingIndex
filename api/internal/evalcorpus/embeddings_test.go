package evalcorpus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Este teste e o GUARD do braco denso do experimento.
//
// O PROBLEMA QUE ELE RESOLVE
//
// Os embeddings do braco C sao gerados fora do processo Go, por
// scripts/embed-corpus.py, e versionados como artefato. Isso mantem o harness
// hermetico, mas cria um risco especifico: alguem edita o corpus -- corrige um
// tipo, acrescenta uma consulta, reescreve um documento -- e ESQUECE de
// regerar os embeddings.
//
// A falha resultante e a pior classe possivel: nada quebra. O experimento roda,
// o CSV e escrito, as tabelas LaTeX sao geradas, e o artigo publica numeros do
// braco denso calculados sobre uma versao ANTERIOR do corpus. Ninguem percebe,
// porque o sintoma e um numero plausivel.
//
// Este teste transforma esse silencio numa falha de CI.
//
// POR QUE POR HASH DO TEXTO, E NAO POR CONTAGEM DE ITENS
//
// Conferir so a quantidade pegaria insercao e remocao, mas nao EDICAO -- que e
// justamente o caso mais provavel (ajustar a redacao de uma consulta). O hash
// do texto exato enviado ao modelo pega os tres casos.

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

	// A direcao inversa tambem importa: um embedding orfao significa que um
	// item foi REMOVIDO do corpus sem o artefato ser regerado. Nao afeta o
	// resultado, mas indica que o artefato esta velho -- e se esta velho para
	// remocao, pode estar velho para edicao.
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
	// O TESTE MAIS IMPORTANTE DESTE ARQUIVO.
	//
	// Ele detecta a edicao silenciosa: o item continua existindo, com o mesmo
	// ID, mas o TEXTO mudou depois que o vetor foi gerado.
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
		// A montagem TEM de ser identica a de scripts/embed-corpus.py
		// (build_passage) e a de cmd/retrieval-eval (buildArms). As tres
		// concordarem e o que garante que todos os bracos indexam a mesma
		// unidade -- se divergissem, a comparacao mediria diferenca de campo
		// indexado em vez de diferenca de modelo de recuperacao.
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
	// O artigo nomeia o modelo e a dimensao. Se o artefato fosse regerado com
	// outro modelo, o artigo passaria a descrever um experimento que nao foi o
	// executado -- e a banca nao teria como perceber.
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
	// Dimensao errada nao causa panic no codigo de busca (dotProduct devolve 0
	// para tamanhos diferentes) -- causa um ranking em que TODOS os scores sao
	// zero e a ordem sai do desempate por ID. Um resultado perfeitamente
	// plausivel e completamente sem sentido.
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
