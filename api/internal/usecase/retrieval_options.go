package usecase

import "os"

// RetrievalOptions guarda os bracos experimentais da recuperacao, lidos UMA vez
// na subida do processo e injetados nos casos de uso.
//
// Antes, THINGINDEX_PAGE_SEARCH_FIELDS era lido por os.Getenv para cada pagina
// de cada pergunta (e o aviso de valor invalido saia uma vez por pagina), e
// THINGINDEX_QUERY_EXPANSION a cada pergunta do chat (parecer B5). Os valores
// padrao e o aviso sao os mesmos; so a leitura mudou de lugar.
//
// Os scripts de experimento (scripts/reexecuta-bracos-enriquecimento.sh,
// experimento-bracos.sh, experimento-expansao.sh) trocam o braco reiniciando o
// processo da API com a variavel no ambiente, entao ler na subida preserva o
// comportamento deles.
type RetrievalOptions struct {
	// PageSearchFields: content, content+summary (padrao), content+questions
	// ou max(content,questions).
	PageSearchFields string
	// QueryExpansion liga a expansao da pergunta por LLM antes da busca.
	QueryExpansion bool
}

// NewRetrievalOptions valida os valores crus (como vieram do ambiente). Valor
// desconhecido de pageSearchFields cai no padrao com um aviso no log.
func NewRetrievalOptions(pageSearchFields, queryExpansion string) RetrievalOptions {
	return RetrievalOptions{
		PageSearchFields: normalizePageSearchFields(pageSearchFields),
		QueryExpansion:   parseQueryExpansion(queryExpansion),
	}
}

// RetrievalOptionsFromEnv le THINGINDEX_PAGE_SEARCH_FIELDS e
// THINGINDEX_QUERY_EXPANSION do ambiente do processo. Para binarios que nao
// carregam config.Config (cmd/indexing-benchmark).
func RetrievalOptionsFromEnv() RetrievalOptions {
	return NewRetrievalOptions(os.Getenv("THINGINDEX_PAGE_SEARCH_FIELDS"), os.Getenv("THINGINDEX_QUERY_EXPANSION"))
}

// DefaultRetrievalOptions e o comportamento sem nenhuma variavel definida.
func DefaultRetrievalOptions() RetrievalOptions {
	return RetrievalOptions{PageSearchFields: pageFieldsContentSummary}
}

// pageSearchFields devolve o braco ativo. O valor zero da struct (usado por
// testes que montam o caso de uso a mao) equivale ao padrao, sem aviso.
func (o RetrievalOptions) pageSearchFields() string {
	if o.PageSearchFields == "" {
		return pageFieldsContentSummary
	}
	return o.PageSearchFields
}
