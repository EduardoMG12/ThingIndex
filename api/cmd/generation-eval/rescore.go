package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"thing-index/internal/evalcorpus"
)

// rescoreFile reavalia correct/provenance_ok/retrieval_hit de um CSV com o
// gabarito atual, a partir das respostas e fontes gravadas. Serve para uma
// versao nova do gabarito (regex mais tolerante, pagina-alvo a mais) valer
// para resultados ja pagos, sem chamar o modelo de novo. As colunas de
// resposta, fontes, tokens e latencia nao mudam.
func rescoreFile(path string, g *evalcorpus.Gabarito) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	records, err := csv.NewReader(f).ReadAll()
	f.Close()
	if err != nil || len(records) == 0 {
		return 0, fmt.Errorf("csv vazio ou invalido: %v", err)
	}
	idx := map[string]int{}
	for i, h := range records[0] {
		idx[h] = i
	}
	for _, k := range []string{"question_id", "arm", "correct", "cited_pages", "provenance_ok", "retrieval_hit", "error", "answer"} {
		if _, ok := idx[k]; !ok {
			return 0, fmt.Errorf("coluna %q ausente", k)
		}
	}
	resolve := g.DocumentResolver(fold)
	questions := map[string]evalcorpus.GabaritoQuestion{}
	for _, q := range g.Questions {
		questions[q.ID] = q
	}
	changed := 0
	for _, r := range records[1:] {
		q, ok := questions[r[idx["question_id"]]]
		if !ok {
			continue
		}
		before := r[idx["correct"]] + r[idx["provenance_ok"]]
		correct := r[idx["error"]] == "" && matchesAnswer(r[idx["answer"]], q.AnswerPatterns)
		r[idx["correct"]] = strconv.FormatBool(correct)
		if !q.IsNegative() {
			if r[idx["arm"]] == "T" {
				cites := parseCitations(r[idx["cited_pages"]], resolve)
				prov := strconv.FormatBool(provenanceOK(q, cites))
				r[idx["provenance_ok"]], r[idx["retrieval_hit"]] = prov, prov
			} else {
				r[idx["provenance_ok"]] = strconv.FormatBool(provenanceByPage(q, parsePages(r[idx["cited_pages"]])))
			}
		}
		if before != r[idx["correct"]]+r[idx["provenance_ok"]] {
			changed++
		}
	}
	out, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	w := csv.NewWriter(out)
	if err := w.WriteAll(records); err != nil {
		return 0, err
	}
	return changed, nil
}

// parseCitations le "documento:pagina;..." e normaliza o documento pelo
// resolvedor do gabarito. O corte e no ULTIMO ':', porque titulo de documento
// pode ter dois-pontos.
func parseCitations(s string, resolve func(string) string) []citation {
	var out []citation
	for _, part := range strings.Split(s, ";") {
		i := strings.LastIndex(part, ":")
		if i < 0 {
			continue
		}
		doc, page := part[:i], part[i+1:]
		if n, err := strconv.Atoi(page); err == nil {
			out = append(out, citation{DocumentID: resolve(doc), Page: n})
		}
	}
	return out
}

func parsePages(s string) []int {
	var out []int
	for _, part := range strings.Split(s, ";") {
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			out = append(out, n)
		}
	}
	return out
}
