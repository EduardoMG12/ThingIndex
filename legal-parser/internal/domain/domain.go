package domain

// SourceProvider is the input port. The core never knows where files come from.
// To add a new source (S3, GCS), implement this interface. No domain code changes needed.
type SourceProvider interface {
	// ListDocuments returns all available raw documents from the source.
	ListDocuments() ([]RawDocument, error)
	// GetContent returns the raw HTML/PDF bytes for a document by its ID.
	GetContent(id string) ([]byte, error)
	// GetMetadata returns the crawler metadata JSON for a document by its ID.
	GetMetadata(id string) (*CrawlerMetadata, error)
}

// OutputProvider is the output port for saving parsed documents.
type OutputProvider interface {
	Save(id string, data []byte) error
}

// --- Raw Input Types (from Crawler) ---

// RawDocument represents a discovered file pair (HTML + JSON metadata) from the crawler.
type RawDocument struct {
	ID           string // Hash prefix used to pair .htm and .json files
	ContentFile  string // Path/key to the .htm file
	MetadataFile string // Path/key to the .json file
}

// LinkContext represents the hierarchical position inside a legal document
// where a link was found (e.g., Art. 5, § 2, Inciso III, Alínea b).
type LinkContext struct {
	Artigo    string `json:"artigo,omitempty"`
	Paragrafo string `json:"paragrafo,omitempty"`
	Inciso    string `json:"inciso,omitempty"`
	Alinea    string `json:"alinea,omitempty"`
}

// EnrichedLink holds a discovered link plus its structural origin context.
type EnrichedLink struct {
	URL     string      `json:"url"`
	Text    string      `json:"text,omitempty"`
	Context LinkContext `json:"context"`
}

// CrawlerMetadata mirrors the JSON metadata written by the crawler.
type CrawlerMetadata struct {
	URL            string         `json:"url"`
	LocalPath      string         `json:"local_path"`
	ParentURL      string         `json:"parent_url,omitempty"`
	Description    string         `json:"description,omitempty"`
	ParentContext  LinkContext    `json:"parent_context,omitempty"`
	ExtractedLinks []string       `json:"extracted_links,omitempty"`
	LinksContext   []EnrichedLink `json:"links_context,omitempty"`
}

// --- Parsed Output Types ---

// Reference represents a cross-reference to another legal document,
// enriched with the hierarchical context of where the reference appears.
type Reference struct {
	ID      string       `json:"id"`
	Context *LinkContext `json:"context,omitempty"`
}

// LegalDocument is the final structured representation of a law.
type LegalDocument struct {
	ID          string      `json:"id"`
	Tipo        string      `json:"tipo"`
	Esfera      string      `json:"esfera"`
	Numero      string      `json:"numero"`
	Ano         string      `json:"ano"`
	Titulo      string      `json:"titulo"`
	Ementa        string      `json:"ementa"`
	URLOrigem     string      `json:"url_origem"`
	URLPai        string      `json:"url_pai,omitempty"`
	URLPaiContext *LinkContext `json:"url_pai_context,omitempty"`
	Artigos       []Article   `json:"artigos,omitempty"`
	Referencias   []Reference `json:"referencias,omitempty"`
}

// Article represents an article within a law.
type Article struct {
	Numero     string      `json:"numero"`
	Caput      string      `json:"caput"`
	Status     string      `json:"status"` // "vigente", "revogado", "vetado"
	Paragrafos []Paragraph `json:"paragrafos,omitempty"`
	Incisos    []Subsection `json:"incisos,omitempty"`
}

// Paragraph represents a paragraph (§) within an article.
type Paragraph struct {
	Numero  string      `json:"numero"`
	Texto   string      `json:"texto"`
	Status  string      `json:"status"`
	Incisos []Subsection `json:"incisos,omitempty"`
}

// Subsection represents an inciso (I, II, III...) within an article or paragraph.
type Subsection struct {
	Numero  string  `json:"numero"`
	Texto   string  `json:"texto"`
	Status  string  `json:"status"`
	Alineas []Item  `json:"alineas,omitempty"`
}

// Item represents an alínea (a, b, c...) within a subsection.
type Item struct {
	Numero string `json:"numero"`
	Texto  string `json:"texto"`
	Status string `json:"status"`
}
