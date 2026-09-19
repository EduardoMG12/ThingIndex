package evalcorpus

import "testing"

// Os gabaritos por tema existem para sustentar a afirmacao generica do trabalho
// com medicao em varios acervos, e todos compartilham a MESMA taxonomia de
// forma de pergunta. Isso permite ler o resultado em duas direcoes: por tema
// (o sistema vai bem neste acervo?) e por forma (o sistema vai bem quando a
// pergunta e feita assim?).
//
// A forma "premissa-falsa" e a mais informativa das cinco: a pergunta embute um
// dado errado e a resposta certa e corrigir. Um sistema que concorda com a
// premissa acerta a aparencia e erra o conteudo, e nenhuma metrica de acerto
// sobre perguntas bem-formuladas pega isso.
func TestGabaritosPorTemaSaoValidos(t *testing.T) {
	t.Parallel()

	temas := map[string]int{
		"testdata/gabarito-seguranca-trabalho.json": 10,
		"testdata/gabarito-produto.json":            10,
	}

	formasEsperadas := map[string]bool{
		"aberta": true, "confirmacao": true, "existencia": true,
		"premissa-falsa": true, "negativa": true,
	}

	for caminho, esperado := range temas {
		g, err := LoadGabarito(caminho)
		if err != nil {
			t.Errorf("%s: %v", caminho, err)
			continue
		}
		if len(g.Questions) != esperado {
			t.Errorf("%s: %d perguntas, esperava %d", caminho, len(g.Questions), esperado)
		}
		vistas := map[string]bool{}
		for _, q := range g.Questions {
			if !formasEsperadas[q.Stratum] {
				t.Errorf("%s: pergunta %q usa forma %q, fora da taxonomia", caminho, q.ID, q.Stratum)
			}
			vistas[q.Stratum] = true
		}
		for forma := range formasEsperadas {
			if !vistas[forma] {
				t.Errorf("%s: nenhuma pergunta da forma %q; o tema nao exercita a taxonomia inteira", caminho, forma)
			}
		}
	}
}

// Pergunta de premissa falsa com alvo declarado e proposital: diferente da
// negativa, a informacao EXISTE no acervo. O que nao existe e o dado que a
// pergunta afirma. Sem alvo, ela seria indistinguivel de uma negativa.
func TestPremissaFalsaApontaOndeEstaACorrecao(t *testing.T) {
	t.Parallel()

	for _, caminho := range []string{"testdata/gabarito-seguranca-trabalho.json", "testdata/gabarito-produto.json"} {
		g, err := LoadGabarito(caminho)
		if err != nil {
			t.Fatalf("%s: %v", caminho, err)
		}
		for _, q := range g.Questions {
			if q.Stratum != "premissa-falsa" {
				continue
			}
			if len(q.Targets) == 0 {
				t.Errorf("%s: pergunta %q de premissa falsa sem alvo; ela precisa apontar onde esta o dado correto", caminho, q.ID)
			}
		}
	}
}
