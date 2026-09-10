package evalcorpus

import (
	"path/filepath"
	"strings"
	"testing"
)

func gabaritoPath() string { return filepath.Join("testdata", "gabarito-si.json") }

func TestGabaritoSICarregaEValida(t *testing.T) {
	g, err := LoadGabarito(gabaritoPath())
	if err != nil {
		t.Fatalf("gabarito invalido: %v", err)
	}
	if len(g.Questions) < GabaritoMinQuestions {
		t.Fatalf("esperava >= %d perguntas, veio %d", GabaritoMinQuestions, len(g.Questions))
	}
	cc5 := 0
	for _, q := range g.Questions {
		if q.Origin == "rodada-cc5" {
			cc5++
		}
	}
	if cc5 < 11 {
		t.Fatalf("as 11 perguntas da rodada CC5 precisam estar marcadas; achei %d", cc5)
	}
}

func TestGabaritoRecusaPaginaForaDoDocumento(t *testing.T) {
	g, err := LoadGabarito(gabaritoPath())
	if err != nil {
		t.Fatal(err)
	}
	g.Questions[0].Targets[0].Pages = append(g.Questions[0].Targets[0].Pages, 9999)
	g.FrozenSHA256 = g.QuestionsSHA256()
	err = g.Validate()
	if err == nil || !strings.Contains(err.Error(), "fora de 1..") {
		t.Fatalf("esperava erro de pagina fora do documento, veio %v", err)
	}
}

func TestGabaritoRecusaPerguntaAlteradaSemHash(t *testing.T) {
	g, err := LoadGabarito(gabaritoPath())
	if err != nil {
		t.Fatal(err)
	}
	g.Questions[0].Text += " (editada depois de congelar)"
	err = g.Validate()
	if err == nil || !strings.Contains(err.Error(), "frozen_sha256") {
		t.Fatalf("esperava erro de hash, veio %v", err)
	}
}
