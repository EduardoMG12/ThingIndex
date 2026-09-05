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

// TestSecaoPrecisaTerAncoraQueExisteNoDocumento trava o requisito central do
// gabarito em nivel de no: a secao e declarada PELO CORPUS, com uma ancora que
// precisa existir no texto-fonte.
//
// Sem essa checagem, uma ancora com erro de digitacao nunca casaria com a saida
// do extrator, e a consulta correspondente simplesmente nunca seria satisfeita
// -- o braco levaria a culpa por um erro de anotacao.
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

// TestSecaoNaoPodeTerIDDuplicado protege a ancoragem estavel: dois nos com o
// mesmo id tornam o gabarito ambiguo.
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

// TestRelevanciaDeSecaoPrecisaApontarSecaoExistente e o analogo, em nivel de no,
// da checagem que ja existe para documento: gabarito apontando para alvo
// inexistente e o erro de anotacao mais traicoeiro, porque nao falha -- apenas
// nunca e satisfeito.
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

// TestConsultaPositivaComSecoesPrecisaJulgarSecao garante a coerencia entre os
// dois niveis de gabarito: se o documento tem secoes declaradas e a consulta o
// julga relevante, ela precisa dizer QUAL secao. Sem isso o gabarito de no
// ficaria incompleto em silencio, e a metrica cairia sem que a recuperacao
// tivesse piorado.
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

// TestCheckCircularityDeSecaoRecusaRotuloCopiadoDoExtrator e o teste mais
// importante deste arquivo.
//
// O corpus antigo foi invalidado porque `expected_top_node_title` era copia
// literal da saida do extrator estrutural: renomear a heuristica de heading
// derrubava a metrica sem que a recuperacao tivesse mudado. O gabarito media
// autoconsistencia, nao recuperacao.
//
// Ao introduzir gabarito em nivel de no existe o risco de reintroduzir
// exatamente esse defeito. Esta checagem trava isso: a ancora precisa ser texto
// do DOCUMENTO, e o id precisa ser autoral e estavel.
func TestCheckCircularityDeSecaoRecusaRotuloCopiadoDoExtrator(t *testing.T) {
	corpus := &Corpus{
		Version: "teste",
		Documents: []Document{{
			ID:      "doc-a",
			Content: "1 RELATORIO\nO reclamante alega horas extras.",
			Sections: []Section{
				// Id que carrega o titulo extraido em vez de um identificador
				// estavel: se o extrator renomear, o gabarito quebra.
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

// TestCorpusValidoComSecoesPassa e o controle: sem ele os testes acima
// passariam com uma validacao que recusa tudo.
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

// TestDocumentoSemSecoesContinuaValido garante a degradacao exigida pelo
// RNF-1 do spec: documento sem secoes declaradas entra como unidade unica, sem
// quebrar o experimento.
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
