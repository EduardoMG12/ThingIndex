package domain

import "io"

// PDFMetadata represents the metadata of a crawled PDF.
type PDFMetadata struct {
	URL          string   `json:"url"`
	LocalPath    string   `json:"local_path"`
	ParentURL    string   `json:"parent_url,omitempty"` // URL of the PDF where this link was found
	ExtractedLinks []string `json:"extracted_links,omitempty"`
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
