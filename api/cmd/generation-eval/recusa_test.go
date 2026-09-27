package main

import (
	"testing"

	"thing-index/internal/evalcorpus"
)

// Linha de base da recusa constante: quanto de cada gabarito uma resposta
// fixa, que nao olha o acervo, "acerta". A banca apontou que as expressoes
// das formas existencia e premissa falsa aceitavam recusa. Este teste mede
// isso e trava o resultado: depois da correcao dos padroes, uma recusa so
// pode acertar as perguntas negativas.
func TestRecusaConstanteSoAcertaNegativas(t *testing.T) {
	recusas := []string{
		"Não há informações sobre isso nos documentos fornecidos.",
		"Não sei.",
		"Os documentos não trazem essa informação.",
	}
	for _, tema := range []string{"seguranca-trabalho", "produto", "licitacoes"} {
		g, err := evalcorpus.LoadGabarito("../../internal/evalcorpus/testdata/gabarito-" + tema + ".json")
		if err != nil {
			t.Fatal(err)
		}
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
