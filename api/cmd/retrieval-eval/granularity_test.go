package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"thing-index/internal/evalcorpus"
	"thing-index/internal/retrieval"
)

func carregarCorpusDeTeste(t *testing.T) *evalcorpus.Corpus {
	t.Helper()
	corpus, err := evalcorpus.Load(filepath.Join("..", "..", "internal", "evalcorpus", "testdata", "corpus-c1.json"))
	if err != nil {
		t.Fatalf("carregar corpus: %v", err)
	}
	return corpus
}

// A medicao em nivel de documento tem de continuar identica a de antes da
// issue #16.
func TestDocumentosComoUnidadeUnicaPreservaAGranularidadeAntiga(t *testing.T) {
	corpus := carregarCorpusDeTeste(t)

	units := evalcorpus.RetrievalUnits(documentosComoUnidadeUnica(corpus))
	if len(units) != len(corpus.Documents) {
		t.Fatalf("esperava %d unidades (uma por documento), veio %d", len(corpus.Documents), len(units))
	}

	for i, unit := range units {
		if unit.ID != corpus.Documents[i].ID {
			t.Fatalf("unidade %d tem id %q, esperava %q", i, unit.ID, corpus.Documents[i].ID)
		}
		if unit.SectionID != "" {
			t.Fatalf("unidade de documento nao pode ter SectionID, veio %q", unit.SectionID)
		}
		// Documento completo, nao recortado.
		if !strings.Contains(unit.Text, strings.TrimSpace(corpus.Documents[i].Content)) {
			t.Fatalf("unidade %q nao contem o conteudo completo do documento", unit.ID)
		}
	}
}

// Em nivel de documento R@5 e 1,000 para todos os lexicais (5 de 5 sempre acha
// o relevante); em nivel de secao (19 unidades) a metrica volta a separar.
func TestGranularidadeDeSecaoDiscriminaMaisQueDocumento(t *testing.T) {
	corpus := carregarCorpusDeTeste(t)

	medir := func(units []evalcorpus.RetrievalUnit, judgments func(evalcorpus.Query) map[string]int) map[string]float64 {
		arms := buildArms(units, nil)
		rows := evaluate(corpus, arms, "x", judgments)

		soma := map[string]float64{}
		conta := map[string]int{}
		for _, r := range rows {
			if r.negative {
				continue
			}
			soma[r.metrics.Arm] += r.metrics.RecallAt5
			conta[r.metrics.Arm]++
		}
		media := map[string]float64{}
		for arm, total := range soma {
			media[arm] = total / float64(conta[arm])
		}
		return media
	}

	porDocumento := medir(
		evalcorpus.RetrievalUnits(documentosComoUnidadeUnica(corpus)),
		func(q evalcorpus.Query) map[string]int { return q.Relevance },
	)
	porSecao := medir(
		evalcorpus.RetrievalUnits(corpus.Documents),
		func(q evalcorpus.Query) map[string]int { return q.RelevanceSections },
	)

	// Em nivel de documento o R@5 e saturado: recuperar 5 de 5 e aritmetica.
	for arm, valor := range porDocumento {
		if valor < 0.999 {
			t.Fatalf(
				"R@5 do braco %s em nivel de documento deveria ser saturado (1,000), veio %.3f -- se o corpus mudou, a premissa da issue #16 precisa ser revista",
				arm, valor,
			)
		}
	}

	// Em nivel de secao a metrica precisa voltar a discriminar.
	saturados := 0
	for _, valor := range porSecao {
		if valor >= 0.999 {
			saturados++
		}
	}
	if saturados == len(porSecao) {
		t.Fatal("R@5 continua saturado em nivel de secao: a mudanca de unidade nao produziu poder de discriminacao")
	}
}

func TestCSVDistingueAsDuasGranularidades(t *testing.T) {
	corpus := carregarCorpusDeTeste(t)

	arms := buildArms(evalcorpus.RetrievalUnits(corpus.Documents), nil)
	rows := evaluate(corpus, arms, "section", func(q evalcorpus.Query) map[string]int {
		return q.RelevanceSections
	})

	arquivo, err := os.CreateTemp(t.TempDir(), "results-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer arquivo.Close()

	if err := writeCSV(arquivo, rows); err != nil {
		t.Fatalf("escrever csv: %v", err)
	}

	conteudo, err := os.ReadFile(arquivo.Name())
	if err != nil {
		t.Fatal(err)
	}

	linhas := strings.Split(strings.TrimSpace(string(conteudo)), "\n")
	if !strings.Contains(linhas[0], "granularity") {
		t.Fatalf("cabecalho sem a coluna granularity: %q", linhas[0])
	}
	for _, linha := range linhas[1:] {
		if !strings.Contains(linha, ",section,") {
			t.Fatalf("linha sem a granularidade declarada: %q", linha)
		}
	}
}

// Queda de P@1 literal -> parafrase no relacional: -0,429 por documento,
// -0,857 por secao.
func TestBracoRelacionalDependeMaisDoCasamentoLexicalEmNivelDeSecao(t *testing.T) {
	corpus := carregarCorpusDeTeste(t)

	quedaP1 := func(units []evalcorpus.RetrievalUnit, judgments func(evalcorpus.Query) map[string]int) float64 {
		arms := buildArms(units, nil)
		rows := evaluate(corpus, arms, "x", judgments)

		soma := map[string]float64{}
		conta := map[string]int{}
		for _, r := range rows {
			if r.negative || r.metrics.Arm != "A-relacional" {
				continue
			}
			soma[r.variant] += r.metrics.PrecisionAt1
			conta[r.variant]++
		}
		literal := soma["literal"] / float64(conta["literal"])
		parafrase := soma["paraphrase"] / float64(conta["paraphrase"])
		return parafrase - literal
	}

	porDocumento := quedaP1(
		evalcorpus.RetrievalUnits(documentosComoUnidadeUnica(corpus)),
		func(q evalcorpus.Query) map[string]int { return q.Relevance },
	)
	porSecao := quedaP1(
		evalcorpus.RetrievalUnits(corpus.Documents),
		func(q evalcorpus.Query) map[string]int { return q.RelevanceSections },
	)

	if porSecao >= porDocumento {
		t.Fatalf(
			"esperava queda MAIOR em nivel de secao; documento %.3f, secao %.3f -- se a diferenca sumiu, a granularidade fina deixou de revelar a dependencia lexical",
			porDocumento, porSecao,
		)
	}
}

func TestTokenizeEhEstavel(t *testing.T) {
	entrada := "Horas Extraordinárias, adicional NOTURNO!"
	primeira := retrieval.Tokenize(entrada)
	for i := 0; i < 10; i++ {
		outra := retrieval.Tokenize(entrada)
		if len(outra) != len(primeira) {
			t.Fatal("tokenizacao instavel entre chamadas")
		}
		for j := range primeira {
			if primeira[j] != outra[j] {
				t.Fatalf("token %d variou: %q vs %q", j, primeira[j], outra[j])
			}
		}
	}
}
