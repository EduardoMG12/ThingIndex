package main

import (
	"reflect"
	"testing"

	"thing-index/internal/evalcorpus"
)

func TestMatchesAnswerDobraAcentoECaixaEExigeTodos(t *testing.T) {
	ok := matchesAnswer("O pré-requisito é SIS07206 – Programação de Computadores I.", []string{`sis07206|programacao de computadores i\b`})
	if !ok {
		t.Fatal("deveria casar com acento dobrado")
	}
	if matchesAnswer("4º período", []string{`\b4\b|quarto`, `banco de dados i\b`}) {
		t.Fatal("todos os padroes precisam casar")
	}
	if matchesAnswer("qualquer coisa", nil) {
		t.Fatal("sem padrao nao ha acerto")
	}
}

func TestCitedPagesLeFormasComuns(t *testing.T) {
	got := citedPages("Consta na p. 54 e nas páginas 9 e 134 do PPC (pág. 3 da matriz).")
	if !reflect.DeepEqual(got, []int{54, 9, 134, 3}) {
		t.Fatalf("paginas citadas: %v", got)
	}
	if len(citedPages("nenhuma citacao aqui, 2026 nao e pagina")) != 0 {
		t.Fatal("numero solto nao e pagina")
	}
}

func TestProvenanciaEstruturadaEPorPagina(t *testing.T) {
	q := evalcorpus.GabaritoQuestion{Targets: []evalcorpus.GabaritoTarget{{DocumentID: "matriz", Pages: []int{1}}, {DocumentID: "ppc-si", Pages: []int{56}}}}
	if !provenanceOK(q, []citation{{"ppc-si", 12}, {"matriz", 1}}) {
		t.Fatal("matriz p.1 e alvo")
	}
	if provenanceOK(q, []citation{{"ppc-si", 1}}) {
		t.Fatal("p.1 do PPC nao e alvo: a pagina certa no documento errado nao conta")
	}
	if !provenanceByPage(q, []int{56}) || provenanceByPage(q, []int{2}) {
		t.Fatal("proveniencia por pagina frouxa")
	}
}

// O titulo gravado na colecao nem sempre e o do gabarito ("NR 6 - Equipamentos
// de Protecao Individual" vs "NR 6 - EPI"); tem de resolver pelo id.
func TestCitacaoPeloTituloDaColecaoCasaComOAlvoViaAlias(t *testing.T) {
	g := &evalcorpus.Gabarito{Documents: []evalcorpus.GabaritoDocument{
		{ID: "nr-06", Title: "NR 6 - EPI", Aliases: []string{"NR 6 - Equipamentos de Protecao Individual"}},
		{ID: "lei-10520", Title: "Lei 10.520: Pregao"},
	}}
	resolve := g.DocumentResolver(fold)
	q := evalcorpus.GabaritoQuestion{Targets: []evalcorpus.GabaritoTarget{{DocumentID: "nr-06", Pages: []int{2}}}}

	cites := parseCitations("NR 6 - Equipamentos de Proteção Individual:2;nr-18:5", resolve)
	if !provenanceOK(q, cites) {
		t.Fatalf("o alias (com acento dobrado) deveria resolver para nr-06: %+v", cites)
	}
	if got := parseCitations("Lei 10.520: Pregao:7", resolve); len(got) != 1 || got[0] != (citation{"lei-10520", 7}) {
		t.Fatalf("titulo com dois-pontos precisa cortar no ultimo ':': %+v", got)
	}
	if got := resolve("documento que ninguem conhece"); got != "documento que ninguem conhece" {
		t.Fatalf("nome desconhecido deve voltar como veio, veio %q", got)
	}
}
