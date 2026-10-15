package evalcorpus

import (
	"strings"
	"testing"
)

// tokenizeSimples replica a tokenizacao usada pelo harness para os testes de
// circularidade nao dependerem do pacote de recuperacao.
func tokenizeSimples(text string) []string {
	campos := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !('a' <= r && r <= 'z') && !('0' <= r && r <= '9') && r < 128
	})
	return campos
}

func TestSecaoPrecisaTerAncoraQueExisteNoDocumento(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:      "doc-a",
			Content: "ACORDAO\n\n1 RELATORIO\nTexto do relatorio.\n\n2 DISPOSITIVO\nTexto do dispositivo.",
			Sections: []Section{
				{ID: "doc-a#s1", Anchor: "1 RELATORIO"},
				{ID: "doc-a#s2", Anchor: "2 DISPOSITIVO NAO EXISTE"},
			},
		}},
		Queries: []Query{{
			ID: "q1", Text: "relatorio", Variant: "literal",
			Relevance:         map[string]int{"doc-a": 3},
			RelevanceSections: map[string]int{"doc-a#s1": 3},
		}},
	}

	err := corpus.Validate()
	if err == nil {
		t.Fatal("esperava recusa de ancora inexistente no texto-fonte")
	}
	if !strings.Contains(err.Error(), "doc-a#s2") {
		t.Fatalf("o erro precisa apontar a secao problematica, veio: %v", err)
	}
}

func TestSecaoNaoPodeTerIDDuplicado(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:      "doc-a",
			Content: "1 UM\ntexto\n\n2 DOIS\ntexto",
			Sections: []Section{
				{ID: "doc-a#s1", Anchor: "1 UM"},
				{ID: "doc-a#s1", Anchor: "2 DOIS"},
			},
		}},
		Queries: []Query{{
			ID: "q1", Text: "um", Variant: "literal",
			Relevance:         map[string]int{"doc-a": 3},
			RelevanceSections: map[string]int{"doc-a#s1": 3},
		}},
	}

	if err := corpus.Validate(); err == nil {
		t.Fatal("esperava recusa de id de secao duplicado")
	}
}

func TestRelevanciaDeSecaoPrecisaApontarSecaoExistente(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:       "doc-a",
			Content:  "1 UM\ntexto",
			Sections: []Section{{ID: "doc-a#s1", Anchor: "1 UM"}},
		}},
		Queries: []Query{{
			ID: "q1", Text: "um", Variant: "literal",
			Relevance:         map[string]int{"doc-a": 3},
			RelevanceSections: map[string]int{"doc-a#s99": 3},
		}},
	}

	if err := corpus.Validate(); err == nil {
		t.Fatal("esperava recusa de relevancia para secao inexistente")
	}
}

func TestConsultaPositivaComSecoesPrecisaJulgarSecao(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:       "doc-a",
			Content:  "1 UM\ntexto",
			Sections: []Section{{ID: "doc-a#s1", Anchor: "1 UM"}},
		}},
		Queries: []Query{{
			ID: "q1", Text: "um", Variant: "literal",
			Relevance: map[string]int{"doc-a": 3},
			// RelevanceSections ausente de proposito.
		}},
	}

	err := corpus.Validate()
	if err == nil {
		t.Fatal("esperava recusa de consulta que julga documento com secoes mas nao julga secao")
	}
	if !strings.Contains(err.Error(), "q1") {
		t.Fatalf("o erro precisa nomear a consulta, veio: %v", err)
	}
}

func TestCheckCircularityDeSecaoRecusaRotuloCopiadoDoExtrator(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:      "doc-a",
			Content: "1 RELATORIO\nO reclamante alega horas extras.",
			Sections: []Section{
				// Id com o titulo extraido em vez de um id estavel.
				{ID: "1 RELATORIO", Anchor: "1 RELATORIO"},
			},
		}},
		Queries: []Query{{
			ID: "q1", Text: "horas extras", Variant: "paraphrase",
			Relevance:         map[string]int{"doc-a": 3},
			RelevanceSections: map[string]int{"1 RELATORIO": 3},
		}},
	}

	avisos := corpus.CheckSectionAnchoring()
	if len(avisos) == 0 {
		t.Fatal("esperava aviso de id de secao derivado do titulo extraido")
	}
	if !strings.Contains(avisos[0], "doc-a") {
		t.Fatalf("o aviso precisa nomear o documento, veio: %v", avisos)
	}
}

// Controle: sem ele os testes acima passariam com uma validacao que recusa tudo.
func TestCorpusValidoComSecoesPassa(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:      "doc-a",
			Content: "ACORDAO\n\n1 RELATORIO\nO reclamante alega horas extras.\n\n2 DISPOSITIVO\nJulgo procedente.",
			Sections: []Section{
				{ID: "doc-a#s1", Anchor: "1 RELATORIO"},
				{ID: "doc-a#s2", Anchor: "2 DISPOSITIVO"},
			},
		}},
		Queries: []Query{
			{
				ID: "q1", Text: "jornada suplementar", Variant: "paraphrase",
				Relevance:         map[string]int{"doc-a": 3},
				RelevanceSections: map[string]int{"doc-a#s1": 3},
			},
			{
				ID: "q2", Text: "receita de bolo", Variant: "paraphrase",
				Negative: true,
			},
		},
	}

	if err := corpus.Validate(); err != nil {
		t.Fatalf("corpus valido foi recusado: %v", err)
	}
	if avisos := corpus.CheckSectionAnchoring(); len(avisos) != 0 {
		t.Fatalf("corpus bem ancorado gerou aviso: %v", avisos)
	}
	if avisos := corpus.CheckCircularity(tokenizeSimples); len(avisos) != 0 {
		t.Fatalf("consulta parafraseada foi acusada de circular: %v", avisos)
	}
}

func TestDocumentoSemSecoesContinuaValido(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:      "doc-a",
			Content: "Texto corrido sem estrutura declarada.",
		}},
		Queries: []Query{{
			ID: "q1", Text: "texto", Variant: "literal",
			Relevance: map[string]int{"doc-a": 3},
		}},
	}

	if err := corpus.Validate(); err != nil {
		t.Fatalf("documento sem secoes deveria continuar valido: %v", err)
	}
}
