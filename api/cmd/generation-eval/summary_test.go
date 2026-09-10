package main

import (
	"math"
	"strings"
	"testing"
)

func TestBootstrapEDeterministicoEContemAMedia(t *testing.T) {
	v := []float64{1, 0, 1, 1, 0, 1, 0, 1, 1, 1}
	m1, lo1, hi1 := bootstrapCI(v, 1000, 1)
	m2, lo2, hi2 := bootstrapCI(v, 1000, 1)
	if m1 != 0.7 || m1 != m2 || lo1 != lo2 || hi1 != hi2 {
		t.Fatalf("bootstrap nao deterministico ou media errada: %v %v %v / %v %v %v", m1, lo1, hi1, m2, lo2, hi2)
	}
	if !(lo1 <= m1 && m1 <= hi1) || lo1 == hi1 {
		t.Fatalf("intervalo degenerado: %v..%v", lo1, hi1)
	}
	if m, _, _ := bootstrapCI(nil, 10, 1); !math.IsNaN(m) {
		t.Fatal("vazio deve dar NaN, nao zero")
	}
}

func TestAgregaPorPerguntaAntesDeResumir(t *testing.T) {
	yes, no := true, false
	rows := []resultRow{
		{QuestionID: "q1", Stratum: "tabela", Arm: "T", Correct: &yes, RetrievalHit: &yes, ProvenanceOK: &yes, PromptTokens: 100},
		{QuestionID: "q1", Stratum: "tabela", Arm: "T", Correct: &no, RetrievalHit: &yes, ProvenanceOK: &no, PromptTokens: 300},
		{QuestionID: "q2", Stratum: "negativa", Arm: "T", Correct: &yes, Err: true},
	}
	qs := aggregate(rows)
	if len(qs) != 2 || qs[0].Correct != 0.5 || qs[0].Provenance != 0.5 || qs[0].Hit != 1 || qs[0].PromptTokens != 200 {
		t.Fatalf("agregacao errada: %+v", qs)
	}
	if qs[1].HasProvenance || qs[1].Errors != 1 {
		t.Fatalf("negativa nao tem proveniencia e o erro precisa contar: %+v", qs[1])
	}
	linha := summaryLine("todas", qs)
	if !strings.Contains(linha, "| todas | 2 | 75.0%") {
		t.Fatalf("linha do resumo: %s", linha)
	}
}
