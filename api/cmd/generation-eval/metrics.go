package main

import (
	"regexp"
	"strconv"
	"strings"

	"thing-index/internal/evalcorpus"
	"thing-index/internal/retrieval"
)

// fold dobra acento e caixa: e a mesma dobra do retrieval de producao, para
// que "Programação" no gabarito e "programacao" na resposta casem.
func fold(s string) string { return strings.ToLower(retrieval.FoldDiacritics(s)) }

// matchesAnswer diz se TODOS os padroes do gabarito casam na resposta.
func matchesAnswer(answer string, patterns []string) bool {
	folded := fold(answer)
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil || !re.MatchString(folded) {
			return false
		}
	}
	return len(patterns) > 0
}

// answerCorrect aplica o criterio completo de uma pergunta: todas as
// AnswerPatterns casam e nenhuma RejectPattern casa. E a unica funcao que o
// braco ao vivo e o rescore usam, para que os dois nunca divirjam.
func answerCorrect(answer string, q evalcorpus.GabaritoQuestion) bool {
	if !matchesAnswer(answer, q.AnswerPatterns) {
		return false
	}
	folded := fold(answer)
	for _, p := range q.RejectPatterns {
		re, err := regexp.Compile(p)
		if err != nil || re.MatchString(folded) {
			return false
		}
	}
	return true
}

var citedPagesRE = regexp.MustCompile(`\b(?:p\.|pp\.|pag\.|pags?\.|pagina|paginas)\s*((?:\d+\s*(?:,|e|a|-|ate)?\s*)+)`)
var numberRE = regexp.MustCompile(`\d+`)

// citedPages extrai as paginas que uma resposta em prosa cita ("p. 54",
// "paginas 9 e 54", "pag. 3"). Serve ao braco F, que nao tem fontes
// estruturadas: a unica proveniencia possivel e a que o modelo escreve.
func citedPages(answer string) []int {
	seen := map[int]bool{}
	var out []int
	for _, m := range citedPagesRE.FindAllStringSubmatch(fold(answer), -1) {
		for _, n := range numberRE.FindAllString(m[1], -1) {
			v, err := strconv.Atoi(n)
			if err != nil || v <= 0 || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// citation e uma fonte estruturada (braco T): documento do gabarito e pagina.
type citation struct {
	DocumentID string
	Page       int
}

// provenanceOK diz se alguma citacao aponta para (documento, pagina) do alvo.
func provenanceOK(q evalcorpus.GabaritoQuestion, cites []citation) bool {
	targets := q.TargetPages()
	for _, c := range cites {
		for _, p := range targets[c.DocumentID] {
			if p == c.Page {
				return true
			}
		}
	}
	return false
}

// provenanceByPage e a versao frouxa para o braco F: a pagina citada em prosa
// nao diz de que documento e, entao basta pertencer a uniao das paginas-alvo.
// Isso favorece o braco F; o relatorio diz isso.
func provenanceByPage(q evalcorpus.GabaritoQuestion, pages []int) bool {
	for _, t := range q.Targets {
		for _, p := range t.Pages {
			for _, c := range pages {
				if c == p {
					return true
				}
			}
		}
	}
	return false
}

// estimateTokens e a mesma estimativa do chat de producao (runas/4), usada so
// para detectar truncamento quando o provedor reporta bem menos que o enviado.
func estimateTokens(s string) int { return (len([]rune(s)) + 3) / 4 }

func formatCitations(cites []citation) string {
	parts := make([]string, 0, len(cites))
	for _, c := range cites {
		parts = append(parts, c.DocumentID+":"+strconv.Itoa(c.Page))
	}
	return strings.Join(parts, ";")
}

func formatPages(pages []int) string {
	parts := make([]string, 0, len(pages))
	for _, p := range pages {
		parts = append(parts, strconv.Itoa(p))
	}
	return strings.Join(parts, ";")
}
