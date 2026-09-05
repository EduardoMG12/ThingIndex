package evalcorpus

import "testing"

// Este arquivo trava as propriedades do corpus C1 REAL (nao de fixtures
// sinteticas). Constituicao, principio III: numero citado no artigo vem de
// execucao real, e o teste trava a constante para que altera-la obrigue a
// revisar o artigo.

const (
	// documentosC1 e secoesC1 sao os universos de recuperacao. A razao entre
	// eles e o que da poder de discriminacao ao experimento.
	documentosC1 = 5
	secoesC1     = 19
	// paresSecaoJulgadosC1 e o total de julgamentos (consulta, secao).
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

// TestC1TemOUniversoDeSecoesDeclaradoNoArtigo trava os dois numeros que
// sustentam a afirmacao de que a mudanca de unidade aumenta a discriminacao.
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

// TestC1AumentaOPoderDeDiscriminacao e o teste que justifica a issue #16
// inteira. Ele falha se alguem reduzir o corpus a ponto de o experimento voltar
// a nao discriminar.
func TestC1AumentaOPoderDeDiscriminacao(t *testing.T) {
	corpus := carregarC1(t)

	secoes := 0
	for _, doc := range corpus.Documents {
		secoes += len(doc.Sections)
	}

	// Com 5 documentos, R@5 e 1,000 por aritmetica: recuperar 5 de 5 sempre
	// acha o relevante. Com o universo de secoes maior que a profundidade de
	// corte, a metrica volta a significar alguma coisa.
	const corteR = 5
	if secoes <= corteR {
		t.Fatalf("universo de secoes (%d) nao excede o corte R@%d: a metrica volta a ser aritmetica", secoes, corteR)
	}

	// A baseline aleatoria de P@1 precisa cair substancialmente; e ela que
	// explica por que os bracos empatavam.
	baselineDoc := 1.0 / float64(len(corpus.Documents))
	baselineSec := 1.0 / float64(secoes)
	if baselineSec >= baselineDoc/2 {
		t.Fatalf("baseline aleatoria mal melhorou: documento %.3f -> secao %.3f", baselineDoc, baselineSec)
	}
}

// TestC1TemGabaritoDeSecaoParaTodaConsultaPositiva garante que a anotacao
// cobriu o corpus inteiro. Uma consulta positiva sem gabarito de secao seria
// medida como falha do braco, nao como lacuna de anotacao.
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

// TestC1NaoReintroduziuCircularidadeNaAnotacaoDeSecao e a guarda mais
// importante sobre o corpus real: roda a checagem de ancoragem sobre o arquivo
// de verdade, nao sobre fixture.
func TestC1NaoReintroduziuCircularidadeNaAnotacaoDeSecao(t *testing.T) {
	corpus := carregarC1(t)

	if avisos := corpus.CheckSectionAnchoring(); len(avisos) != 0 {
		t.Fatalf("o corpus C1 voltou a ancorar gabarito na saida do extrator:\n%v", avisos)
	}
}

// TestC1MantemCoerenciaEntreOsDoisNiveis garante que o gabarito de secao nao
// contradiz o de documento -- toda secao julgada pertence a um documento
// tambem julgado relevante na mesma consulta.
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
