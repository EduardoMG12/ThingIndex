package main

import (
	"os"
	"path/filepath"
	"testing"

	"thing-index/internal/evalcorpus"
)

func TestProseProvenanceUsaAMesmaRegraNosDoisBracos(t *testing.T) {
	g := &evalcorpus.Gabarito{Questions: []evalcorpus.GabaritoQuestion{
		{ID: "q1", Targets: []evalcorpus.GabaritoTarget{{DocumentID: "d", Pages: []int{4}}}},
		{ID: "n1"},
	}}
	csvText := "question_id,arm,answer\n" +
		"q1,T,\"Segundo a página 4, sim.\"\n" +
		"q1,F,\"Conforme a página 7, não.\"\n" +
		"q1,T,\"Sim, sem citar página.\"\n" +
		"n1,T,\"Não há informação (página 2).\"\n"
	path := filepath.Join(t.TempDir(), "r.csv")
	if err := os.WriteFile(path, []byte(csvText), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := proseProvenance(path, g)
	if err != nil {
		t.Fatal(err)
	}
	// A negativa n1 nao entra; das tres com alvo, duas citam pagina, uma e alvo.
	if s != (proseStats{WithTarget: 3, CitesPage: 2, CitesTarget: 1, CitesNonTarget: 1}) {
		t.Fatalf("contagem inesperada: %+v", s)
	}
}
