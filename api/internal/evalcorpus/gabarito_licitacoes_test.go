package evalcorpus

import (
	"strings"
	"testing"
)

// O gabarito de licitacoes e o segundo dominio medido pelo trabalho, e existe
// para sustentar a afirmacao generica do artigo com mais de um acervo.
//
// Ele estreia um estrato que o gabarito de ensino nao tem: DIVERGENTE, em que a
// mesma pergunta tem resposta diferente em cada lei. Acertar ali significa
// distinguir as duas, nao escolher uma; responder so pela lei revogada e o modo
// de falha que motivou o corpus.
func TestGabaritoDeLicitacoesEhValido(t *testing.T) {
	t.Parallel()

	g, err := LoadGabarito("testdata/gabarito-licitacoes.json")
	if err != nil {
		t.Fatalf("carregar gabarito de licitacoes: %v", err)
	}

	if len(g.Documents) != 2 {
		t.Fatalf("esperava 2 documentos, veio %d", len(g.Documents))
	}

	porEstrato := map[string]int{}
	for _, q := range g.Questions {
		porEstrato[q.Stratum]++
	}
	for estrato, minimo := range g.Strata {
		if porEstrato[estrato] < minimo {
			t.Errorf("estrato %q tem %d perguntas, minimo declarado %d", estrato, porEstrato[estrato], minimo)
		}
	}

	// Pergunta divergente sem alvo nos DOIS documentos nao e divergente: ela
	// seria respondivel por uma lei so, e o estrato perderia o sentido.
	for _, q := range g.Questions {
		if q.Stratum != "divergente" {
			continue
		}
		if len(q.Targets) < 2 {
			t.Errorf("pergunta divergente %q aponta %d documento(s); precisa apontar as duas leis", q.ID, len(q.Targets))
		}
	}
}

// O gabarito de ensino nao declara estratos proprios e precisa continuar
// valendo pelos padroes historicos. Se esta regressao quebrar, a mudanca que
// tornou os estratos configuraveis afrouxou o gabarito que ja existia.
func TestGabaritoDeEnsinoContinuaValidoComOsPadroes(t *testing.T) {
	t.Parallel()

	g, err := LoadGabarito("testdata/gabarito-si.json")
	if err != nil {
		t.Fatalf("carregar gabarito de ensino: %v", err)
	}
	if len(g.Strata) != 0 {
		t.Fatalf("o gabarito de ensino nao deveria declarar estratos proprios, veio %v", g.Strata)
	}
	if len(g.Questions) < GabaritoMinQuestions {
		t.Fatalf("o minimo padrao deixou de ser aplicado: %d perguntas", len(g.Questions))
	}
}

// Estrato nao declarado tem que ser recusado, e a mensagem precisa dizer quais
// valem naquele gabarito. Sem isso, um erro de digitacao no estrato passaria
// despercebido e a pergunta sairia da contagem de cobertura sem aviso.
func TestEstratoForaDoDeclaradoEhRecusado(t *testing.T) {
	t.Parallel()

	g := gabaritoMinimo()
	g.Questions[0].Stratum = "calendario" // valido no ensino, inexistente aqui
	g.FrozenSHA256 = g.QuestionsSHA256()

	err := g.Validate()
	if err == nil {
		t.Fatal("esperava recusa de estrato nao declarado")
	}
	if !contemTodos(err.Error(), "calendario", "divergente") {
		t.Fatalf("a mensagem precisa nomear o estrato recusado e os aceitos, veio: %v", err)
	}
}

// Estrato declarado mas nao cumprido tambem e recusado, com o nome do estrato:
// um gabarito que promete cinco perguntas divergentes e entrega duas nao mede o
// que diz medir.
func TestEstratoDeclaradoENaoCumpridoEhRecusado(t *testing.T) {
	t.Parallel()

	g := gabaritoMinimo()
	g.Strata = map[string]int{"divergente": 99}
	g.FrozenSHA256 = g.QuestionsSHA256()

	err := g.Validate()
	if err == nil {
		t.Fatal("esperava recusa por estrato incompleto")
	}
	if !contemTodos(err.Error(), "divergente", "99") {
		t.Fatalf("a mensagem precisa nomear o estrato e o minimo, veio: %v", err)
	}
}

func contemTodos(texto string, partes ...string) bool {
	for _, p := range partes {
		if !strings.Contains(texto, p) {
			return false
		}
	}
	return true
}

// gabaritoMinimo monta um gabarito valido de um estrato so, para os testes de
// validacao mexerem em um campo por vez.
func gabaritoMinimo() *Gabarito {
	perguntas := make([]GabaritoQuestion, 0, 2)
	for i := 0; i < 2; i++ {
		perguntas = append(perguntas, GabaritoQuestion{
			ID:             string(rune('a' + i)),
			Text:           "pergunta",
			Stratum:        "divergente",
			Targets:        []GabaritoTarget{{DocumentID: "d1", Pages: []int{1}}},
			AnswerPatterns: []string{"resposta"},
			Expected:       "resposta",
			Origin:         "pdf",
		})
	}
	g := &Gabarito{
		Version:      "teste",
		Strata:       map[string]int{"divergente": 2},
		MinQuestions: 2,
		Documents:    []GabaritoDocument{{ID: "d1", File: "d1.pdf", Title: "D1", Pages: 10}},
		Questions:    perguntas,
	}
	g.FrozenSHA256 = g.QuestionsSHA256()
	return g
}
