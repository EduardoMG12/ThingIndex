package evalcorpus

import "testing"

// Cada unidade tem o texto da propria secao, nao o documento inteiro.
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

	if !contains(units[0].Text, "uptime de 99,5") {
		t.Fatalf("unidade 1 nao contem o proprio conteudo: %q", units[0].Text)
	}
	if contains(units[0].Text, "desconto de 5") {
		t.Fatalf("unidade 1 vazou o conteudo da unidade 2: %q", units[0].Text)
	}
	if !contains(units[1].Text, "desconto de 5") {
		t.Fatalf("unidade 2 nao contem o proprio conteudo: %q", units[1].Text)
	}

	// Texto antes da primeira ancora (capa) nao entra em nenhuma secao.
	if contains(units[0].Text, "texto de capa") {
		t.Fatalf("a capa vazou para dentro da primeira secao: %q", units[0].Text)
	}

	for _, unit := range units {
		if unit.DocumentID != "doc-a" {
			t.Fatalf("unidade %q perdeu a proveniencia do documento", unit.ID)
		}
	}
}

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
