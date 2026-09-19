// Package provenance mede a auditabilidade do indice produzido pelo ThingIndex.
//
// As metricas aqui sao CONTRIBUICAO PROPRIA do trabalho, e nao emprestadas da
// literatura de RI. A razao: o artigo se chama auditavel e afirma auditabilidade
// como vantagem central sobre RAG denso, mas nunca definiu o termo de forma
// mensuravel. "Auditavel" sem metrica e adjetivo de marketing; com metrica, e
// resultado.
//
// Duas metricas, ambas sobre o indice ja persistido:
//
//	TPC (taxa de proveniencia completa)  -- % de itens cuja origem e resolvivel
//	TCV (taxa de citacao verificavel)    -- % de citacoes cujo trecho existe de fato
//
// O furo que a TPC expoe e concreto: `ParagraphNode.PageNumber` e `*int`,
// portanto NULAVEL, e blocos vindos do tesseract entram sem caixa delimitadora.
// Um item sem numero de pagina nao pode ser apontado no documento original --
// e "aponte onde voce leu isso" e exatamente o que auditabilidade significa.
package provenance

import (
	"sort"
	"strings"
	"unicode"
)

// Item e uma unidade indexada cuja proveniencia sera avaliada.
type Item struct {
	// Kind identifica a origem estrutural ("paragraph", "table", "metadata",
	// "chunk"). Reportar por tipo importa: se so os paragrafos tem pagina e as
	// tabelas nao, a media global esconde que metade da evidencia e inauditavel.
	Kind string

	DocumentID uint
	// PageNumber e ponteiro porque o modelo real permite nulo. Manter o
	// ponteiro aqui e deliberado -- trocar por int com zero-value mascararia
	// exatamente o defeito que a metrica existe para medir.
	PageNumber    *int
	SequenceOrder int
	Content       string
	// Scope diz em que unidade a origem do item pode ser apontada:
	//
	//	ScopePage     -- a evidencia existe numa pagina do documento
	//	ScopeDocument -- o item descreve o documento inteiro
	//
	// A distincao existe porque nem todo item indexado e um trecho. Os pares
	// de metadado que o indexador gera sao estatisticas do documento
	// ("paragraph_count", "tree_builder"): eles nao estao em pagina nenhuma.
	// Exigir pagina deles deixava duas saidas ruins -- contar como falha
	// permanente, afundando a media com um defeito que nao existe, ou
	// preencher uma pagina qualquer, que seria FABRICAR proveniencia e viola
	// o principio I da constituicao. Classificar o escopo e a terceira saida:
	// o item continua visivel na conta, com o criterio que cabe a ele.
	//
	// Vazio e tratado como ScopePage de proposito. O padrao tem que ser o
	// criterio ESTRITO: se alguem esquecer de classificar um item novo, ele
	// aparece como incompleto e o teste reclama. O contrario faria um
	// esquecimento virar silenciosamente um item "completo".
	Scope string
}

// Escopos de proveniencia. Ver o comentario de Item.Scope.
const (
	ScopePage     = "page"
	ScopeDocument = "document"
)

// scopeOf devolve o escopo efetivo do item, aplicando o padrao estrito.
func scopeOf(item Item) string {
	if item.Scope == ScopeDocument {
		return ScopeDocument
	}
	return ScopePage
}

// Report e o resultado da avaliacao de proveniencia.
type Report struct {
	Total int
	// Complete e o numero de itens com proveniencia resolvivel: document_id
	// nao nulo, page_number nao nulo, sequence_order >= 0 e conteudo nao vazio.
	Complete int
	// Breakdown das causas de incompletude. Um total de falhas nao orienta
	// correcao; saber que 100% delas sao page_number ausente orienta.
	MissingDocumentID    int
	MissingPageNumber    int
	InvalidSequenceOrder int
	EmptyContent         int
	// ByKind permite reportar TPC por tipo estrutural.
	ByKind map[string]KindReport
	// ByScope permite reportar TPC por escopo de origem. Sem esse recorte, um
	// leitor nao consegue separar "item que deveria ter pagina e nao tem" de
	// "item que nunca teria pagina", e as duas coisas viram o mesmo numero.
	ByScope map[string]KindReport
}

// KindReport e o recorte por tipo estrutural.
type KindReport struct {
	Total    int
	Complete int
}

// Rate devolve a TPC no intervalo [0,1].
//
// Sem itens, devolve 0 e nao 1: um indice vazio nao e "perfeitamente
// auditavel", e reportar 100% para o vazio seria a pior falha possivel numa
// metrica de auditabilidade.
func (r Report) Rate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Complete) / float64(r.Total)
}

// RateFor devolve a TPC de um tipo estrutural.
func (r Report) RateFor(kind string) float64 {
	item, ok := r.ByKind[kind]
	if !ok || item.Total == 0 {
		return 0
	}
	return float64(item.Complete) / float64(item.Total)
}

// RateForScope devolve a TPC de um escopo de origem.
func (r Report) RateForScope(scope string) float64 {
	item, ok := r.ByScope[scope]
	if !ok || item.Total == 0 {
		return 0
	}
	return float64(item.Complete) / float64(item.Total)
}

// Scopes devolve os escopos presentes, ordenados, para reporte estavel.
func (r Report) Scopes() []string {
	scopes := make([]string, 0, len(r.ByScope))
	for scope := range r.ByScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

// Kinds devolve os tipos presentes, ordenados, para reporte estavel em tabela.
func (r Report) Kinds() []string {
	kinds := make([]string, 0, len(r.ByKind))
	for kind := range r.ByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// Evaluate calcula a taxa de proveniencia completa.
//
// Um item so conta como completo quando TODAS as coordenadas de origem que
// CABEM NELE estao presentes. O criterio e deliberadamente rigido: proveniencia
// parcial nao permite apontar o trecho no documento original, e portanto nao
// serve para auditoria -- saber que algo veio do documento 7 sem saber de qual
// pagina nao e melhor, na pratica, do que nao saber nada.
//
// "Que cabem nele" nao e afrouxamento: a pagina so e exigida de item cuja
// origem E uma pagina (ScopePage). Uma estatistica do documento inteiro
// (ScopeDocument) esta completamente rastreada quando aponta o documento, e
// exigir pagina dela nao aumentaria auditabilidade nenhuma -- apenas
// produziria um numero pior sem que exista defeito a corrigir. O relatorio
// publica os dois recortes lado a lado justamente para o leitor conferir que a
// distincao nao esta escondendo item que deveria ter pagina.
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

// Citation e uma fonte citada por uma resposta de chat.
type Citation struct {
	// Snippet e o trecho que a resposta apresentou ao usuario.
	Snippet string
	// SourceContent e o conteudo real do item apontado pela citacao.
	SourceContent string
}

// CitationReport e o resultado da verificacao de citacoes.
type CitationReport struct {
	Total    int
	Verified int
}

// Rate devolve a TCV no intervalo [0,1]. Mesma decisao da TPC: vazio devolve 0.
func (r CitationReport) Rate() float64 {
	if r.Total == 0 {
		return 0
	}
	return float64(r.Verified) / float64(r.Total)
}

// VerifyCitations mede quantas citacoes tem o trecho REALMENTE presente na
// fonte apontada.
//
// E a metrica que separa "citou" de "citou corretamente". Um sistema pode
// apontar o documento certo e ainda assim exibir um trecho que nao existe ali
// -- caso classico de alucinacao com aparencia de fundamentacao, e o modo de
// falha mais perigoso num sistema que se apresenta como auditavel.
//
// A comparacao normaliza espaco em branco e caixa, mas NAO normaliza
// pontuacao nem acento: relaxar mais que isso passaria a aceitar como
// verificada uma citacao que o leitor humano nao encontraria no documento.
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

// normalizeForComparison colapsa espaco em branco e caixa.
//
// A diferenca entre o texto exibido e o texto armazenado costuma ser quebra de
// linha e espaco duplo, introduzidos na renderizacao. Ignorar isso evita falso
// negativo; ignorar pontuacao e acento produziria falso POSITIVO, que numa
// metrica de auditabilidade e o erro mais grave.
func normalizeForComparison(value string) string {
	fields := strings.FieldsFunc(strings.ToLower(value), unicode.IsSpace)
	return strings.Join(fields, " ")
}
