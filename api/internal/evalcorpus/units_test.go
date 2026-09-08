package evalcorpus

import "testing"

// TestRetrievalUnitsFatiaPorSecao prova a conversao central da issue #16: um
// documento com secoes declaradas vira N unidades recuperaveis, cada uma com o
// texto da PROPRIA secao -- nao o documento inteiro repetido.
//
// Se o texto de cada unidade fosse o documento completo, todos os nos de um
// mesmo documento teriam score identico e o ranking em nivel de secao seria
// indistinguivel do ranking em nivel de documento: a mudanca de unidade seria
// cosmetica.
func TestRetrievalUnitsFatiaPorSecao(t *testing.T) {
	doc := Document{
		ID:      "doc-a",
		Title:   "Contrato",
		Content: "CAPA\ntexto de capa\n\n1 DISPONIBILIDADE\nuptime de 99,5 por cento\n\n2 PENALIDADES\ndesconto de 5 por cento na fatura",
		Sections: []Section{
			{ID: "doc-a#s1", Anchor: "1 DISPONIBILIDADE", Title: "Disponibilidade"},
			{ID: "doc-a#s2", Anchor: "2 PENALIDADES", Title: "Penalidades"},
		},
	}

	units := RetrievalUnits([]Document{doc})
	if len(units) != 2 {
		t.Fatalf("esperava 2 unidades, veio %d", len(units))
	}

	if units[0].ID != "doc-a#s1" || units[1].ID != "doc-a#s2" {
		t.Fatalf("ids fora de ordem ou errados: %q, %q", units[0].ID, units[1].ID)
	}

	// A primeira unidade termina onde a segunda comeca.
	if !contains(units[0].Text, "uptime de 99,5") {
		t.Fatalf("unidade 1 nao contem o proprio conteudo: %q", units[0].Text)
	}
	if contains(units[0].Text, "desconto de 5") {
		t.Fatalf("unidade 1 vazou o conteudo da unidade 2: %q", units[0].Text)
	}
	if !contains(units[1].Text, "desconto de 5") {
		t.Fatalf("unidade 2 nao contem o proprio conteudo: %q", units[1].Text)
	}

	// O texto anterior a primeira ancora (capa) nao pertence a nenhuma secao
	// julgavel e nao deve poluir a unidade seguinte.
	if contains(units[0].Text, "texto de capa") {
		t.Fatalf("a capa vazou para dentro da primeira secao: %q", units[0].Text)
	}

	for _, unit := range units {
		if unit.DocumentID != "doc-a" {
			t.Fatalf("unidade %q perdeu a proveniencia do documento", unit.ID)
		}
	}
}

// TestRetrievalUnitsDegradaParaDocumentoInteiro trava o RNF-1 da spec 001:
// documento sem secoes declaradas continua sendo uma unidade recuperavel, e o
// experimento nao quebra por causa dele.
func TestRetrievalUnitsDegradaParaDocumentoInteiro(t *testing.T) {
	docs := []Document{
		{ID: "com-secao", Content: "1 UM\nalfa\n\n2 DOIS\nbeta", Sections: []Section{
			{ID: "com-secao#s1", Anchor: "1 UM"},
			{ID: "com-secao#s2", Anchor: "2 DOIS"},
		}},
		{ID: "sem-secao", Title: "Solto", Content: "texto corrido sem estrutura"},
	}

	units := RetrievalUnits(docs)
	if len(units) != 3 {
		t.Fatalf("esperava 3 unidades (2 secoes + 1 documento inteiro), veio %d", len(units))
	}

	ultima := units[2]
	if ultima.ID != "sem-secao" {
		t.Fatalf("documento sem secoes deveria manter o proprio id, veio %q", ultima.ID)
	}
	if ultima.DocumentID != "sem-secao" {
		t.Fatalf("proveniencia errada: %q", ultima.DocumentID)
	}
	if !contains(ultima.Text, "texto corrido") {
		t.Fatalf("documento inteiro perdeu o conteudo: %q", ultima.Text)
	}
}

// TestRetrievalUnitsEhDeterministica trava o RNF-3: duas chamadas produzem a
// mesma ordem. Sem isso o ranking herdaria a ordem de iteracao de mapa e o
// experimento deixaria de ser reproduzivel -- exatamente o defeito que o
// trabalho denuncia no indice vetorial aproximado.
func TestRetrievalUnitsEhDeterministica(t *testing.T) {
	docs := []Document{
		{ID: "b", Content: "1 UM\nalfa", Sections: []Section{{ID: "b#s1", Anchor: "1 UM"}}},
		{ID: "a", Content: "1 UM\nbeta", Sections: []Section{{ID: "a#s1", Anchor: "1 UM"}}},
	}

	primeira := RetrievalUnits(docs)
	for i := 0; i < 20; i++ {
		outra := RetrievalUnits(docs)
		if len(outra) != len(primeira) {
			t.Fatalf("quantidade de unidades variou entre execucoes")
		}
		for j := range primeira {
			if primeira[j].ID != outra[j].ID {
				t.Fatalf("ordem variou na posicao %d: %q vs %q", j, primeira[j].ID, outra[j].ID)
			}
		}
	}
}

// TestRetrievalUnitsUsaCharStartParaDesempatar cobre o caso em que a mesma
// ancora aparece mais de uma vez no documento.
func TestRetrievalUnitsUsaCharStartParaDesempatar(t *testing.T) {
	content := "1 ITEM\nprimeira ocorrencia\n\n1 ITEM\nsegunda ocorrencia"
	segundaPos := len("1 ITEM\nprimeira ocorrencia\n\n")

	doc := Document{
		ID:      "doc-a",
		Content: content,
		Sections: []Section{
			{ID: "doc-a#s2", Anchor: "1 ITEM", CharStart: segundaPos},
		},
	}

	units := RetrievalUnits([]Document{doc})
	if len(units) != 1 {
		t.Fatalf("esperava 1 unidade, veio %d", len(units))
	}
	if !contains(units[0].Text, "segunda ocorrencia") {
		t.Fatalf("char_start ignorado; texto veio: %q", units[0].Text)
	}
	if contains(units[0].Text, "primeira ocorrencia") {
		t.Fatalf("casou a ocorrencia errada: %q", units[0].Text)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
