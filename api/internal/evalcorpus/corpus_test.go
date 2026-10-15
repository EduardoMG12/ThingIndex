package evalcorpus

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"thing-index/internal/retrieval"
)

func corpusPath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("nao consegui resolver o caminho do arquivo de teste")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", name)
}

func TestCorpusC1CarregaEValida(t *testing.T) {
	t.Parallel()

	corpus, err := Load(corpusPath(t, "corpus-c1.json"))
	if err != nil {
		t.Fatalf("carregar corpus C1: %v", err)
	}

	stats := corpus.Describe(retrieval.Tokenize)
	t.Logf("corpus C1: %d documentos, %d consultas, %d pares julgados, %.1f tokens/consulta",
		stats.Documents, stats.Queries, stats.JudgedPairs, stats.AvgQueryTokens)

	if stats.NegativeCases == 0 {
		t.Error("sem casos negativos nao se mede falso positivo: um sistema que sempre responde algo pontua igual a um que sabe dizer 'nao encontrei'")
	}
	for _, variant := range []string{"literal", "paraphrase", "noisy"} {
		if stats.ByVariant[variant] == 0 {
			t.Errorf("faltam consultas da variante %q; a queda literal->paraphrase->noisy e o grafico mais informativo do experimento", variant)
		}
	}
}

// Paraphrase/noisy nao pode ser literal na pratica; "literal" pode.
func TestCorpusC1NaoTemCircularidadeNasVariantesNaoLiterais(t *testing.T) {
	t.Parallel()

	corpus, err := Load(corpusPath(t, "corpus-c1.json"))
	if err != nil {
		t.Fatalf("carregar corpus C1: %v", err)
	}

	if warnings := corpus.CheckCircularity(retrieval.Tokenize); len(warnings) > 0 {
		t.Fatalf("circularidade detectada:\n  %s", strings.Join(warnings, "\n  "))
	}
}

func TestValidateRecusaGabaritoApontandoParaDocumentoInexistente(t *testing.T) {
	t.Parallel()

	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "texto"}},
		Queries: []Query{{
			ID: "q1", Text: "consulta", Variant: "literal",
			Relevance: map[string]int{"documento-que-nao-existe": 3},
		}},
	}

	err := corpus.Validate()
	if err == nil {
		t.Fatal("esperava erro para gabarito apontando documento inexistente")
	}
	if !strings.Contains(err.Error(), "inexistente") {
		t.Fatalf("mensagem pouco util: %v", err)
	}
}

func TestValidateRecusaGrauForaDaEscala(t *testing.T) {
	t.Parallel()

	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "texto"}},
		Queries: []Query{{
			ID: "q1", Text: "consulta", Variant: "literal",
			Relevance: map[string]int{"d1": 7},
		}},
	}

	if err := corpus.Validate(); err == nil {
		t.Fatal("esperava erro para grau fora da escala 0-3")
	}
}

func TestValidateExigeMarcacaoExplicitaDeCasoNegativo(t *testing.T) {
	t.Parallel()

	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "texto"}},
		Queries: []Query{{
			ID: "q1", Text: "consulta", Variant: "literal",
			Relevance: map[string]int{},
		}},
	}

	err := corpus.Validate()
	if err == nil {
		t.Fatal("consulta sem julgamento e sem negative:true deveria ser recusada")
	}
	if !strings.Contains(err.Error(), "negative") {
		t.Fatalf("a mensagem precisa ensinar a correcao, veio: %v", err)
	}
}

func TestValidateRecusaCasoNegativoComJulgamento(t *testing.T) {
	t.Parallel()

	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "texto"}},
		Queries: []Query{{
			ID: "q1", Text: "consulta", Variant: "literal",
			Relevance: map[string]int{"d1": 3},
			Negative:  true,
		}},
	}

	if err := corpus.Validate(); err == nil {
		t.Fatal("caso negativo com julgamento e contradicao e deveria ser recusado")
	}
}

func TestValidateRecusaVarianteDesconhecida(t *testing.T) {
	t.Parallel()

	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "texto"}},
		Queries: []Query{{
			ID: "q1", Text: "consulta", Variant: "inventada",
			Relevance: map[string]int{"d1": 3},
		}},
	}

	if err := corpus.Validate(); err == nil {
		t.Fatal("variante desconhecida deveria ser recusada")
	}
}

func TestCheckCircularityDetectaParaphraseQueEhLiteral(t *testing.T) {
	t.Parallel()

	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "o prazo de pagamento e de quinze dias"}},
		Queries: []Query{{
			ID: "q1", Text: "prazo de pagamento", Variant: "paraphrase",
			Relevance: map[string]int{"d1": 3},
		}},
	}

	warnings := corpus.CheckCircularity(retrieval.Tokenize)
	if len(warnings) != 1 {
		t.Fatalf("esperava 1 aviso de circularidade, veio %d: %v", len(warnings), warnings)
	}
}

func TestCheckCircularityIgnoraVarianteLiteral(t *testing.T) {
	t.Parallel()

	// "literal" copia termos do alvo de proposito.
	corpus := &Corpus{
		Documents: []Document{{ID: "d1", Content: "o prazo de pagamento e de quinze dias"}},
		Queries: []Query{{
			ID: "q1", Text: "prazo de pagamento", Variant: "literal",
			Relevance: map[string]int{"d1": 3},
		}},
	}

	if warnings := corpus.CheckCircularity(retrieval.Tokenize); len(warnings) != 0 {
		t.Fatalf("nao deveria avisar sobre variante literal, veio: %v", warnings)
	}
}
