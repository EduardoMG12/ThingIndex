package dto

// IndexingPreferencesResponse/Request: o que a LLM gera na indexacao.
type IndexingPreferencesResponse struct {
	DocumentSynopsis bool `json:"document_synopsis"`
	PageSummaries    bool `json:"page_summaries"`
	SectionSummaries bool `json:"section_summaries"`
}

type IndexingPreferencesRequest = IndexingPreferencesResponse
