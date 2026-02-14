package domain

import "io"

// LinkContext represents the hierarchical position inside a legal document
// where a link was found (e.g., Art. 5, § 2, Inciso III, Alínea b).
type LinkContext struct {
	Artigo    string `json:"artigo,omitempty"`
	Paragrafo string `json:"paragrafo,omitempty"`
	Inciso    string `json:"inciso,omitempty"`
	Alinea    string `json:"alinea,omitempty"`
}

// EnrichedLink holds a discovered link plus its structural origin context
// within the parent document's hierarchy.
type EnrichedLink struct {
	URL     string      `json:"url"`
	Text    string      `json:"text,omitempty"`
	Context LinkContext `json:"context"`
}

// DocumentMetadata represents the metadata of a crawled document (PDF or HTML).
type DocumentMetadata struct {
	URL            string         `json:"url"`
	LocalPath      string         `json:"local_path"`
	ParentURL      string         `json:"parent_url,omitempty"`   // URL of the page where this document was found
	Description    string         `json:"description,omitempty"`  // Text of the link that led to this document
	ParentContext  LinkContext    `json:"parent_context,omitempty"` // Context in the parent document where the link was found
	ExtractedLinks []string       `json:"extracted_links,omitempty"`
	LinksContext   []EnrichedLink `json:"links_context,omitempty"`
}

// Fetcher defines the interface for fetching content from a URL.
type Fetcher interface {
	Fetch(url string) (io.ReadCloser, error)
}

// Parser defines the interface for extracting links from a PDF.
type Parser interface {
	ExtractLinks(content io.ReaderAt, size int64) ([]string, error)
}

// Storage defines the interface for saving content.
type Storage interface {
	Save(filename string, content io.Reader) (string, error)
	Exists(filename string) bool
	Read(filename string) (io.ReadCloser, error)
}

// Crawler defines the main crawling logic.
type Crawler interface {
	Crawl(startURL string) error
}
