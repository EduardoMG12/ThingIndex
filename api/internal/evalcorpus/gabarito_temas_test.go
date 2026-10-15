package evalcorpus

import (
	"strings"
	"testing"
)

var gabaritosPorTema = []string{
	"testdata/gabarito-seguranca-trabalho.json",
	"testdata/gabarito-produto.json",
	"testdata/gabarito-licitacoes.json",
}

// As cinco formas de pergunta, iguais em todos os temas.
var formasValidas = map[string]bool{
	"aberta": true, "confirmacao": true, "existencia": true,
	"premissa-falsa": true, "negativa": true,
}

func TestGabaritosPorTemaSaoValidos(t *testing.T) {
	t.Parallel()

	for _, caminho := range gabaritosPorTema {
		g, err := LoadGabarito(caminho)
		if err != nil {
			t.Errorf("%s: %v", caminho, err)
			continue
		}

		if len(g.Documents) < 3 {
			t.Errorf("%s: %d documentos; um tema com menos de 3 nao representa um acervo",
				caminho, len(g.Documents))
		}

		vistas := map[string]bool{}
		for _, q := range g.Questions {
			if !formasValidas[q.Stratum] {
				t.Errorf("%s: pergunta %q usa forma %q, fora da taxonomia", caminho, q.ID, q.Stratum)
			}
			vistas[q.Stratum] = true
		}
		for forma := range formasValidas {
			if !vistas[forma] {
				t.Errorf("%s: nenhuma pergunta da forma %q; o tema nao exercita a taxonomia inteira",
					caminho, forma)
			}
		}
	}
}

// A estrutura do par ja e checada em Validate; aqui so se a parafrase de fato
// usa outras palavras.
func TestParafraseNaoRepeteOVocabularioDaVersaoLiteral(t *testing.T) {
	t.Parallel()

	for _, caminho := range gabaritosPorTema {
		g, err := LoadGabarito(caminho)
		if err != nil {
			t.Fatalf("%s: %v", caminho, err)
		}

		porPar := map[string]map[string]string{}
		for _, q := range g.Questions {
			if q.PairID == "" {
				continue
			}
			if porPar[q.PairID] == nil {
				porPar[q.PairID] = map[string]string{}
			}
			porPar[q.PairID][q.Variant] = q.Text
		}
		if len(porPar) == 0 {
			t.Errorf("%s: nenhum par literal/parafrase", caminho)
		}

		for pid, variantes := range porPar {
			literal := termosSignificativos(variantes[VariantLiteral])
			parafrase := termosSignificativos(variantes[VariantParafrase])
			if len(literal) == 0 || len(parafrase) == 0 {
				t.Errorf("%s: par %q tem variante vazia", caminho, pid)
				continue
			}
			compartilhados := 0
			for termo := range parafrase {
				if literal[termo] {
					compartilhados++
				}
			}
			// Metade ou mais dos termos em comum: nao e parafrase.
			if fracao := float64(compartilhados) / float64(len(parafrase)); fracao >= 0.5 {
				t.Errorf("%s: par %q compartilha %.0f%% dos termos entre literal e parafrase; reformule a parafrase\n  literal:   %s\n  parafrase: %s",
					caminho, pid, fracao*100, variantes[VariantLiteral], variantes[VariantParafrase])
			}
		}
	}
}

// termosSignificativos: palavras de 4+ letras, minusculas, fora das funcionais.
func termosSignificativos(texto string) map[string]bool {
	funcionais := map[string]bool{
		"para": true, "pela": true, "pelo": true, "como": true, "qual": true,
		"quais": true, "esta": true, "este": true, "essa": true, "isso": true,
		"pode": true, "deve": true, "tem": true, "que": true, "uma": true,
		"dos": true, "das": true, "com": true, "por": true, "mais": true,
		"onde": true, "quando": true, "sobre": true, "meu": true, "minha": true,
	}
	out := map[string]bool{}
	for _, palavra := range strings.Fields(strings.ToLower(texto)) {
		palavra = strings.Trim(palavra, "?.,;:()\"'")
		if len(palavra) >= 4 && !funcionais[palavra] {
			out[palavra] = true
		}
	}
	return out
}

// Premissa falsa tem alvo: a informacao existe, o dado afirmado e que nao.
func TestPremissaFalsaApontaOndeEstaACorrecao(t *testing.T) {
	t.Parallel()

	for _, caminho := range gabaritosPorTema {
		g, err := LoadGabarito(caminho)
		if err != nil {
			t.Fatalf("%s: %v", caminho, err)
		}
		for _, q := range g.Questions {
			if q.Stratum == "premissa-falsa" && len(q.Targets) == 0 {
				t.Errorf("%s: pergunta %q de premissa falsa sem alvo; ela precisa apontar onde esta o dado correto",
					caminho, q.ID)
			}
		}
	}
}

// O gabarito de ensino usa os estratos padrao.
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

// O edital municipal e documento unico e fica fora das regras de tema; o hash
// vale igual.
func TestGabaritoPrefeituraConfereOHash(t *testing.T) {
	t.Parallel()
	if _, err := LoadGabarito("testdata/gabarito-prefeitura.json"); err != nil {
		t.Fatal(err)
	}
}

// Gabarito de legislacao (19 perguntas, ids d01..n03), restaurado de caf7624
// com outro nome porque gabarito-licitacoes.json passou a ser o da bateria.
func TestGabaritoLegislacaoConfereOHash(t *testing.T) {
	t.Parallel()
	g, err := LoadGabarito("testdata/gabarito-legislacao.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Questions) != 19 {
		t.Fatalf("gabarito da Secao 8.7 com %d perguntas; o artigo cita 19", len(g.Questions))
	}
}
