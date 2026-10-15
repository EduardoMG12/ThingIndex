// Package provenance mede a auditabilidade do indice persistido:
//
//	TPC (taxa de proveniencia completa)  -- % de itens cuja origem e resolvivel
//	TCV (taxa de citacao verificavel)    -- % de citacoes cujo trecho existe de fato
package provenance

import (
	"sort"
	"strings"
	"unicode"
)

type Item struct {
	// Kind: "paragraph", "table", "metadata" ou "chunk".
	Kind string

	DocumentID uint
	// PageNumber e ponteiro porque o modelo permite nulo; int com zero-value
	// esconderia justamente o que se quer medir.
	PageNumber    *int
	SequenceOrder int
	Content       string
	// Scope diz em que unidade a origem do item pode ser apontada:
	//
	//	ScopePage     -- a evidencia existe numa pagina do documento
	//	ScopeDocument -- o item descreve o documento inteiro ("paragraph_count")
	//
	// Vazio vale como ScopePage: item novo nao classificado cai no criterio estrito.
	Scope string
}

const (
	ScopePage     = "page"
	ScopeDocument = "document"
)

func scopeOf(item Item) string {
	if item.Scope == ScopeDocument {
		return ScopeDocument
	}
	return ScopePage
}

type Report struct {
	Total int
	// Complete: document_id e page_number nao nulos, sequence_order >= 0 e
	// conteudo nao vazio.
	Complete int
	// Causas de incompletude.
	MissingDocumentID    int
	MissingPageNumber    int
	InvalidSequenceOrder int
	EmptyContent         int
	ByKind               map[string]KindReport
	ByScope              map[string]KindReport
}

type KindReport struct {
	Total    int
	Complete int
}

// Rate devolve a TPC em [0,1]. Indice vazio da 0, nao 1.
func (r Report) Rate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Complete) / float64(r.Total)
}

func (r Report) RateFor(kind string) float64 {
	item, ok := r.ByKind[kind]
	if !ok || item.Total == 0 {
		return 0
	}
	return float64(item.Complete) / float64(item.Total)
}

func (r Report) RateForScope(scope string) float64 {
	item, ok := r.ByScope[scope]
	if !ok || item.Total == 0 {
		return 0
	}
	return float64(item.Complete) / float64(item.Total)
}

// Scopes devolve os escopos presentes, ordenados.
func (r Report) Scopes() []string {
	scopes := make([]string, 0, len(r.ByScope))
	for scope := range r.ByScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

// Kinds devolve os tipos presentes, ordenados.
func (r Report) Kinds() []string {
	kinds := make([]string, 0, len(r.ByKind))
	for kind := range r.ByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// Evaluate calcula a TPC. Um item so e completo com todas as coordenadas que
// se aplicam a ele; a pagina so e exigida de itens ScopePage.
func Evaluate(items []Item) Report {
	report := Report{
		Total:   len(items),
		ByKind:  map[string]KindReport{},
		ByScope: map[string]KindReport{},
	}

	for _, item := range items {
		kind := item.Kind
		if kind == "" {
			kind = "unknown"
		}
		scope := scopeOf(item)

		entry := report.ByKind[kind]
		entry.Total++
		scopeEntry := report.ByScope[scope]
		scopeEntry.Total++

		complete := true

		if item.DocumentID == 0 {
			report.MissingDocumentID++
			complete = false
		}
		if scope == ScopePage && item.PageNumber == nil {
			report.MissingPageNumber++
			complete = false
		}
		if item.SequenceOrder < 0 {
			report.InvalidSequenceOrder++
			complete = false
		}
		if strings.TrimSpace(item.Content) == "" {
			report.EmptyContent++
			complete = false
		}

		if complete {
			report.Complete++
			entry.Complete++
			scopeEntry.Complete++
		}

		report.ByKind[kind] = entry
		report.ByScope[scope] = scopeEntry
	}

	return report
}

type Citation struct {
	Snippet       string
	SourceContent string
}

type CitationReport struct {
	Total    int
	Verified int
}

// Rate devolve a TCV em [0,1]; vazio da 0.
func (r CitationReport) Rate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Verified) / float64(r.Total)
}

// VerifyCitations conta as citacoes cujo trecho existe de fato na fonte.
// Normaliza espaco e caixa, mas nao pontuacao nem acento.
func VerifyCitations(citations []Citation) CitationReport {
	report := CitationReport{Total: len(citations)}

	for _, citation := range citations {
		snippet := normalizeForComparison(citation.Snippet)
		source := normalizeForComparison(citation.SourceContent)

		if snippet == "" || source == "" {
			continue
		}
		if strings.Contains(source, snippet) {
			report.Verified++
		}
	}

	return report
}

// normalizeForComparison colapsa espaco em branco e caixa (a renderizacao
// introduz quebras e espacos duplos).
func normalizeForComparison(value string) string {
	fields := strings.FieldsFunc(strings.ToLower(value), unicode.IsSpace)
	return strings.Join(fields, " ")
}
