package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
)

// resultRow e uma linha do CSV ja tipada.
type resultRow struct {
	QuestionID, Stratum, Origin, Arm, Scope, Model string
	Correct, ProvenanceOK, RetrievalHit            *bool
	PromptTokens, CompletionTokens, LatencyMS      float64
	CostUSD                                        *float64
	Truncated                                      bool
	Err                                            bool
}

func readResults(paths []string) ([]resultRow, error) {
	var rows []resultRow
	for _, path := range paths {
		if path = strings.TrimSpace(path); path == "" {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		records, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if len(records) == 0 {
			continue
		}
		idx := map[string]int{}
		for i, h := range records[0] {
			idx[h] = i
		}
		get := func(r []string, k string) string {
			if i, ok := idx[k]; ok && i < len(r) {
				return r[i]
			}
			return ""
		}
		for _, r := range records[1:] {
			row := resultRow{QuestionID: get(r, "question_id"), Stratum: get(r, "stratum"), Origin: get(r, "origin"), Arm: get(r, "arm"), Scope: get(r, "scope"), Model: get(r, "model")}
			row.Correct = parseBool(get(r, "correct"))
			row.ProvenanceOK = parseBool(get(r, "provenance_ok"))
			row.RetrievalHit = parseBool(get(r, "retrieval_hit"))
			row.PromptTokens, _ = strconv.ParseFloat(get(r, "prompt_tokens"), 64)
			row.CompletionTokens, _ = strconv.ParseFloat(get(r, "completion_tokens"), 64)
			row.LatencyMS, _ = strconv.ParseFloat(get(r, "latency_ms"), 64)
			if c, err := strconv.ParseFloat(get(r, "cost_usd"), 64); err == nil {
				row.CostUSD = &c
			}
			row.Truncated = get(r, "truncated") == "true"
			row.Err = get(r, "error") != ""
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func parseBool(s string) *bool {
	switch s {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	}
	return nil
}

// perQuestion agrega as repeticoes de uma pergunta: a unidade estatistica e a
// PERGUNTA, nao a chamada, senao as 3 repeticoes contam como 3 observacoes
// independentes e o intervalo fica estreito demais.
type perQuestion struct {
	ID, Stratum, Origin       string
	Correct, Provenance, Hit  float64 // fracao das repeticoes
	HasProvenance, HasHit     bool
	PromptTokens, CompletionT float64
	LatencyMS                 float64
	Cost                      float64
	HasCost                   bool
	Truncated, Errors, N      int
}

func aggregate(rows []resultRow) []perQuestion {
	byID := map[string]*perQuestion{}
	var order []string
	for _, r := range rows {
		pq, ok := byID[r.QuestionID]
		if !ok {
			pq = &perQuestion{ID: r.QuestionID, Stratum: r.Stratum, Origin: r.Origin}
			byID[r.QuestionID] = pq
			order = append(order, r.QuestionID)
		}
		pq.N++
		if r.Correct != nil && *r.Correct {
			pq.Correct++
		}
		if r.ProvenanceOK != nil {
			pq.HasProvenance = true
			if *r.ProvenanceOK {
				pq.Provenance++
			}
		}
		if r.RetrievalHit != nil {
			pq.HasHit = true
			if *r.RetrievalHit {
				pq.Hit++
			}
		}
		pq.PromptTokens += r.PromptTokens
		pq.CompletionT += r.CompletionTokens
		pq.LatencyMS += r.LatencyMS
		if r.CostUSD != nil {
			pq.HasCost = true
			pq.Cost += *r.CostUSD
		}
		if r.Truncated {
			pq.Truncated++
		}
		if r.Err {
			pq.Errors++
		}
	}
	out := make([]perQuestion, 0, len(order))
	for _, id := range order {
		pq := byID[id]
		n := float64(pq.N)
		pq.Correct /= n
		pq.Provenance /= n
		pq.Hit /= n
		pq.PromptTokens /= n
		pq.CompletionT /= n
		pq.LatencyMS /= n
		pq.Cost /= n
		out = append(out, *pq)
	}
	return out
}

// bootstrapCI devolve media e IC 95% por bootstrap percentil sobre as
// perguntas, com semente fixa: o mesmo CSV produz sempre o mesmo intervalo.
func bootstrapCI(values []float64, resamples int, seed int64) (mean, lo, hi float64) {
	if len(values) == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	rng := rand.New(rand.NewSource(seed))
	means := make([]float64, resamples)
	for i := range means {
		var s float64
		for range values {
			s += values[rng.Intn(len(values))]
		}
		means[i] = s / float64(len(values))
	}
	sort.Float64s(means)
	lo = means[int(0.025*float64(resamples))]
	hi = means[int(math.Min(float64(resamples-1), math.Ceil(0.975*float64(resamples))))]
	return mean, lo, hi
}

type groupKey struct{ Arm, Scope, Model string }

func printSummary(paths []string, w io.Writer) error {
	rows, err := readResults(paths)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("nenhuma linha lida de %v", paths)
	}
	groups := map[groupKey][]resultRow{}
	var keys []groupKey
	for _, r := range rows {
		k := groupKey{r.Arm, r.Scope, r.Model}
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], r)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Arm != keys[j].Arm {
			return keys[i].Arm < keys[j].Arm
		}
		if keys[i].Scope != keys[j].Scope {
			return keys[i].Scope < keys[j].Scope
		}
		return keys[i].Model < keys[j].Model
	})

	perGroup := map[groupKey][]perQuestion{}
	for _, k := range keys {
		qs := aggregate(groups[k])
		perGroup[k] = qs
		fmt.Fprintf(w, "## Braço %s", k.Arm)
		if k.Scope != "" {
			fmt.Fprintf(w, " (escopo %s)", k.Scope)
		}
		fmt.Fprintf(w, " — modelo %s — %d perguntas, %d chamadas\n\n", k.Model, len(qs), len(groups[k]))
		fmt.Fprintln(w, "| Recorte | n | Acerto (IC 95%) | Proveniência | Recuperação@6 | Tokens prompt | Tokens resp. | Latência ms | Custo USD/consulta | Truncadas | Erros |")
		fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|")
		slices := []struct {
			name   string
			filter func(perQuestion) bool
		}{
			{"todas", func(perQuestion) bool { return true }},
			{"sem rodada-cc5", func(q perQuestion) bool { return q.Origin != "rodada-cc5" }},
			{"prosa", func(q perQuestion) bool { return q.Stratum == "prosa" }},
			{"tabela", func(q perQuestion) bool { return q.Stratum == "tabela" }},
			{"calendario", func(q perQuestion) bool { return q.Stratum == "calendario" }},
			{"negativa", func(q perQuestion) bool { return q.Stratum == "negativa" }},
		}
		for _, s := range slices {
			var sub []perQuestion
			for _, q := range qs {
				if s.filter(q) {
					sub = append(sub, q)
				}
			}
			if len(sub) == 0 {
				continue
			}
			fmt.Fprintln(w, summaryLine(s.name, sub))
		}
		fmt.Fprintln(w)
	}

	// Comparacao pareada: mesma pergunta nos dois bracos.
	if len(keys) >= 2 {
		fmt.Fprintln(w, "## Pareado por pergunta (fração de repetições certas)")
		fmt.Fprintln(w)
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				a, b := perGroup[keys[i]], perGroup[keys[j]]
				bByID := map[string]perQuestion{}
				for _, q := range b {
					bByID[q.ID] = q
				}
				var wins, losses, ties int
				var diffs []float64
				for _, qa := range a {
					qb, ok := bByID[qa.ID]
					if !ok {
						continue
					}
					diffs = append(diffs, qa.Correct-qb.Correct)
					switch {
					case qa.Correct > qb.Correct:
						wins++
					case qa.Correct < qb.Correct:
						losses++
					default:
						ties++
					}
				}
				if len(diffs) == 0 {
					continue
				}
				mean, lo, hi := bootstrapCI(diffs, 1000, 1)
				fmt.Fprintf(w, "- %s vs %s: %s vence em %d, perde em %d, empata em %d (n=%d); diferença média de acerto %+.3f (IC 95%% %+.3f a %+.3f)\n",
					label(keys[i]), label(keys[j]), label(keys[i]), wins, losses, ties, len(diffs), mean, lo, hi)
			}
		}
	}
	return nil
}

func label(k groupKey) string {
	if k.Scope != "" {
		return k.Arm + "/" + k.Scope
	}
	return k.Arm
}

func summaryLine(name string, qs []perQuestion) string {
	var correct, prov, hit, pt, ct, lat, cost []float64
	var trunc, errs int
	hasCost := false
	for _, q := range qs {
		correct = append(correct, q.Correct)
		if q.HasProvenance {
			prov = append(prov, q.Provenance)
		}
		if q.HasHit {
			hit = append(hit, q.Hit)
		}
		pt = append(pt, q.PromptTokens)
		ct = append(ct, q.CompletionT)
		lat = append(lat, q.LatencyMS)
		if q.HasCost {
			hasCost = true
			cost = append(cost, q.Cost)
		}
		trunc += q.Truncated
		errs += q.Errors
	}
	m, lo, hi := bootstrapCI(correct, 1000, 1)
	pm, _, _ := bootstrapCI(prov, 1, 1)
	hm, _, _ := bootstrapCI(hit, 1, 1)
	ptm, _, _ := bootstrapCI(pt, 1, 1)
	ctm, _, _ := bootstrapCI(ct, 1, 1)
	lm, _, _ := bootstrapCI(lat, 1, 1)
	costText := "—"
	if hasCost {
		cm, _, _ := bootstrapCI(cost, 1, 1)
		costText = fmt.Sprintf("%.4f", cm)
	}
	return fmt.Sprintf("| %s | %d | %.1f%% (%.1f–%.1f) | %s | %s | %.0f | %.0f | %.0f | %s | %d | %d |",
		name, len(qs), 100*m, 100*lo, 100*hi, pct(pm), pct(hm), ptm, ctm, lm, costText, trunc, errs)
}

func pct(v float64) string {
	if math.IsNaN(v) {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", 100*v)
}
