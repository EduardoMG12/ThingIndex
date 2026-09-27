package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"thing-index/internal/evalcorpus"
)

// proseProvenance mede, para os dois bracos com o MESMO criterio, se a
// pagina escrita na prosa da resposta pertence as paginas-alvo. A banca
// apontou que a proveniencia reportada para T (pagina-alvo entre as fontes
// exibidas) e para F (pagina escrita pelo modelo) medem coisas diferentes;
// esta medida e a comparavel. Nao altera os CSV: le e resume.
//
// Devolve, por arquivo: respostas com alvo, respostas que citam alguma
// pagina na prosa, e respostas cuja pagina citada na prosa e alvo.
type proseStats struct {
	WithTarget, CitesPage, CitesTarget int
	CitesNonTarget                     int
}

func proseProvenance(path string, g *evalcorpus.Gabarito) (proseStats, error) {
	var s proseStats
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil || len(records) == 0 {
		return s, fmt.Errorf("csv vazio ou invalido: %v", err)
	}
	idx := map[string]int{}
	for i, h := range records[0] {
		idx[h] = i
	}
	questions := map[string]evalcorpus.GabaritoQuestion{}
	for _, q := range g.Questions {
		questions[q.ID] = q
	}
	for _, r := range records[1:] {
		q, ok := questions[r[idx["question_id"]]]
		if !ok || q.IsNegative() {
			continue
		}
		s.WithTarget++
		pages := citedPages(r[idx["answer"]])
		if len(pages) == 0 {
			continue
		}
		s.CitesPage++
		if provenanceByPage(q, pages) {
			s.CitesTarget++
		} else {
			s.CitesNonTarget++
		}
	}
	return s, nil
}

func printProseProvenance(paths string, g *evalcorpus.Gabarito) error {
	for _, p := range strings.Split(paths, ",") {
		s, err := proseProvenance(p, g)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		fmt.Printf("%s: com alvo %d; citam pagina na prosa %d; pagina da prosa e alvo %d (%.3f das com alvo); so paginas fora do alvo %d\n",
			p, s.WithTarget, s.CitesPage, s.CitesTarget, float64(s.CitesTarget)/float64(s.WithTarget), s.CitesNonTarget)
	}
	return nil
}
