package evalcorpus

import "testing"

// Propriedades do corpus C1 real (nao de fixtures).

const (
	// O extrator devolve 26 nos para este corpus, nao 19: os 7 a mais sao titulos
	// de capa ("ACORDAO", "ANEXO II - ..."), que nao respondem consulta nenhuma.
	documentosC1         = 5
	secoesC1             = 19
	paresSecaoJulgadosC1 = 22
)

func carregarC1(t *testing.T) *Corpus {
	t.Helper()
	corpus, err := Load("testdata/corpus-c1.json")
	if err != nil {
		t.Fatalf("carregar corpus C1: %v", err)
	}
	return corpus
}

func TestC1TemOUniversoDeSecoesDeclaradoNoArtigo(t *testing.T) {
	corpus := carregarC1(t)

	if len(corpus.Documents) != documentosC1 {
		t.Fatalf("documentos: esperava %d, veio %d", documentosC1, len(corpus.Documents))
	}

	total := 0
	for _, doc := range corpus.Documents {
		total += len(doc.Sections)
	}
	if total != secoesC1 {
		t.Fatalf("secoes: esperava %d, veio %d -- se a anotacao mudou de proposito, atualize a constante E o artigo", secoesC1, total)
	}
}

func TestC1AumentaOPoderDeDiscriminacao(t *testing.T) {
	corpus := carregarC1(t)

	secoes := 0
	for _, doc := range corpus.Documents {
		secoes += len(doc.Sections)
	}

	// Com 5 documentos, R@5 e 1,000 por aritmetica.
	const corteR = 5
	if secoes <= corteR {
		t.Fatalf("universo de secoes (%d) nao excede o corte R@%d: a metrica volta a ser aritmetica", secoes, corteR)
	}

	baselineDoc := 1.0 / float64(len(corpus.Documents))
	baselineSec := 1.0 / float64(secoes)
	if baselineSec >= baselineDoc/2 {
		t.Fatalf("baseline aleatoria mal melhorou: documento %.3f -> secao %.3f", baselineDoc, baselineSec)
	}
}

func TestC1TemGabaritoDeSecaoParaTodaConsultaPositiva(t *testing.T) {
	corpus := carregarC1(t)

	pares := 0
	for _, query := range corpus.Queries {
		if query.Negative {
			if len(query.RelevanceSections) != 0 {
				t.Fatalf("consulta negativa %q tem gabarito de secao", query.ID)
			}
			continue
		}
		if len(query.RelevanceSections) == 0 {
			t.Fatalf("consulta positiva %q nao tem gabarito de secao", query.ID)
		}
		pares += len(query.RelevanceSections)
	}

	if pares != paresSecaoJulgadosC1 {
		t.Fatalf("pares (consulta, secao): esperava %d, veio %d", paresSecaoJulgadosC1, pares)
	}
}

func TestC1NaoReintroduziuCircularidadeNaAnotacaoDeSecao(t *testing.T) {
	corpus := carregarC1(t)

	if avisos := corpus.CheckSectionAnchoring(); len(avisos) != 0 {
		t.Fatalf("o corpus C1 voltou a ancorar gabarito na saida do extrator:\n%v", avisos)
	}
}

// Toda secao julgada pertence a um documento julgado relevante na mesma consulta.
func TestC1MantemCoerenciaEntreOsDoisNiveis(t *testing.T) {
	corpus := carregarC1(t)

	donoDaSecao := map[string]string{}
	for _, doc := range corpus.Documents {
		for _, section := range doc.Sections {
			donoDaSecao[section.ID] = doc.ID
		}
	}

	for _, query := range corpus.Queries {
		for sectionID := range query.RelevanceSections {
			docID := donoDaSecao[sectionID]
			if _, ok := query.Relevance[docID]; !ok {
				t.Fatalf(
					"consulta %q julga a secao %q relevante mas nao julga o documento %q que a contem",
					query.ID, sectionID, docID,
				)
			}
		}
	}
}
