package usecase

import "os"

// RetrievalOptions guarda as opcoes experimentais da recuperacao, lidas uma
// vez na subida do processo. Os scripts de experimento trocam o valor
// reiniciando a API.
type RetrievalOptions struct {
	// PageSearchFields: content, content+summary (padrao), content+questions
	// ou max(content,questions).
	PageSearchFields string
	// QueryExpansion liga a expansao da pergunta por LLM antes da busca.
	QueryExpansion bool
}

// NewRetrievalOptions: valor desconhecido de pageSearchFields cai no padrao
// com aviso no log.
func NewRetrievalOptions(pageSearchFields, queryExpansion string) RetrievalOptions {
	return RetrievalOptions{
		PageSearchFields: normalizePageSearchFields(pageSearchFields),
		QueryExpansion:   parseQueryExpansion(queryExpansion),
	}
}

// RetrievalOptionsFromEnv e para binarios que nao carregam config.Config.
func RetrievalOptionsFromEnv() RetrievalOptions {
	return NewRetrievalOptions(os.Getenv("THINGINDEX_PAGE_SEARCH_FIELDS"), os.Getenv("THINGINDEX_QUERY_EXPANSION"))
}

func DefaultRetrievalOptions() RetrievalOptions {
	return RetrievalOptions{PageSearchFields: pageFieldsContentSummary}
}

// pageSearchFields: o valor zero da struct (testes) equivale ao padrao, sem aviso.
func (o RetrievalOptions) pageSearchFields() string {
	if o.PageSearchFields == "" {
		return pageFieldsContentSummary
	}
	return o.PageSearchFields
}
