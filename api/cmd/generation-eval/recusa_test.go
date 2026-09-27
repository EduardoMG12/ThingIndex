package main

import (
	"testing"

	"thing-index/internal/evalcorpus"
)

var temasDaBateria = []string{"seguranca-trabalho", "produto", "licitacoes"}

func carregarTema(t *testing.T, tema string) *evalcorpus.Gabarito {
	t.Helper()
	g, err := evalcorpus.LoadGabarito("../../internal/evalcorpus/testdata/gabarito-" + tema + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// Linha de base da recusa constante: uma resposta fixa, que nao olha o
// acervo, so pode acertar as perguntas negativas. A banca mostrou duas vezes
// que os padroes aceitavam recusa (primeiro "nao" solto; depois "existe",
// "consta" e "trata" sem exigir afirmacao), entao a lista cobre as formas de
// negar que o modelo de fato usa nas respostas versionadas.
func TestRecusaConstanteSoAcertaNegativas(t *testing.T) {
	recusas := []string{
		"Não há informações sobre isso nos documentos fornecidos.",
		"Não sei.",
		"Os documentos não trazem essa informação.",
		"Não existe previsão sobre isso nos documentos.",
		"Não consta nos documentos.",
		"O documento não trata desse assunto.",
		"Não há previsão legal sobre esse tema nos trechos.",
		"Não, os documentos não preveem isso.",
		"Não é possível determinar com base nos documentos fornecidos.",
		// Terceira leitura da banca: formas que o v5 ainda aceitava.
		"Não está previsto nos documentos.",
		"Não há dispositivo que preveja isso.",
		"Nenhum dos documentos trata desse assunto.",
		"Não se trata disso nos documentos.",
		"Nada consta a respeito.",
		"A resposta não pode ser encontrada no contexto fornecido.",
	}
	for _, tema := range temasDaBateria {
		g := carregarTema(t, tema)
		for _, r := range recusas {
			for _, q := range g.Questions {
				if q.IsNegative() {
					continue
				}
				if answerCorrect(r, q) {
					t.Errorf("%s/%s (%s): a recusa %q e aceita como acerto", tema, q.ID, q.Stratum, r)
				}
			}
		}
	}
}

// A resposta oposta a certa nao pode passar. Em lc3 a resposta certa e que a
// Lei 8.666 foi revogada; o padrao "revogad" sozinho aceitava "nao foi
// revogada".
func TestRespostaOpostaNaoPassa(t *testing.T) {
	opostas := map[string][]string{
		"lc3L":  {"Não, a lei não foi revogada.", "Sim, a Lei 8.666 continua em vigor e não foi revogada.", "A Lei 8.666 não foi totalmente revogada."},
		"lc3P":  {"Sim, ainda vale: a lei não foi revogada."},
		"lc4L":  {"Pela Lei 14.133, que se aplica imediatamente a todos os contratos."},
		"lc4P":  {"Sim, muda: a lei nova se aplica imediatamente, e a anterior deixa de valer para ele."},
		"lc12L": {"Não, a Lei 10.520 não foi revogada."},
		"lc12P": {"Não, a Lei 10.520 não foi revogada."},
		"lc13L": {"Sim, o convite ainda existe; não foi revogado."},
	}
	// Confirmacoes dos outros acervos: a resposta oposta a do gabarito.
	opostas["sg4L"] = []string{"Não, o gerenciamento de riscos não inclui fatores psicossociais."}
	opostas["sg11L"] = []string{"Não, a empresa não pode ser dispensada."}
	opostas["sg11P"] = []string{"Não, não pode: a empresa precisa ter o programa."}
	opostas["pr9L"] = []string{"Não, o número de série não é exigido."}
	opostas["pr3L"] = []string{"Sim, a garantia cobre a instalação."}
	var qs []evalcorpus.GabaritoQuestion
	for _, tema := range temasDaBateria {
		qs = append(qs, carregarTema(t, tema).Questions...)
	}
	for _, q := range qs {
		for _, r := range opostas[q.ID] {
			if answerCorrect(r, q) {
				t.Errorf("%s: a resposta oposta %q e aceita como acerto", q.ID, r)
			}
		}
	}
}

// Linha de base de "Sim." por forma: registrada no artigo como o piso das
// formas cuja resposta certa e afirmativa. Este teste so imprime; a
// existencia e a confirmacao sao perguntas de sim ou nao, e o piso delas nao
// e zero por construcao.
// Falso negativo: "píxeis" (plural) e a resposta certa de pr8 e nao contem
// "pixel" depois de tirar os acentos.
func TestPixeisContaComoPixel(t *testing.T) {
	for _, q := range carregarTema(t, "produto").Questions {
		if q.ID == "pr8L" && !answerCorrect("Sim, o manual trata de defeitos de píxeis, na seção 7.1.", q) {
			t.Error("pr8L: 'píxeis' deveria contar como acerto")
		}
	}
}

func TestLinhaDeBaseSimConstante(t *testing.T) {
	porForma := map[string][2]int{}
	for _, tema := range temasDaBateria {
		for _, q := range carregarTema(t, tema).Questions {
			c := porForma[q.Stratum]
			c[1]++
			if answerCorrect("Sim.", q) {
				c[0]++
			}
			porForma[q.Stratum] = c
		}
	}
	for forma, c := range porForma {
		t.Logf("resposta constante \"Sim.\": %s %d/%d", forma, c[0], c[1])
	}
}
