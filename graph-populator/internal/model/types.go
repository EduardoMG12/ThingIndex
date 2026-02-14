package model

// Law represents the top-level legislation node (Lei, Decreto, Constituiçao, etc.)
type Law struct {
	ID        string `json:"id"`
	Tipo      string `json:"tipo"`
	Esfera    string `json:"esfera"`
	Numero    string `json:"numero"`
	Ano       string `json:"ano"`
	Titulo    string `json:"titulo"`
	Ementa    string `json:"ementa"`
	URLOrigem string `json:"url_origem"`
	URLPai    string `json:"url_pai,omitempty"` // For hierarchy (e.g., Decreto regulating a Law)
	Artigos   []Article `json:"artigos"`
}

// Article represents an Artigo node
type Article struct {
	Numero     string      `json:"numero"`
	Caput      string      `json:"caput"`
	Status     string      `json:"status"`
	Paragrafos []Paragraph `json:"paragrafos,omitempty"`
	Incisos    []Inciso    `json:"incisos,omitempty"`
}

// Paragraph represents a Paragrafo node
type Paragraph struct {
	Numero  string   `json:"numero"`
	Texto   string   `json:"texto"`
	Status  string   `json:"status"`
	Incisos []Inciso `json:"incisos,omitempty"`
}

// Inciso represents an Inciso node
type Inciso struct {
	Numero  string   `json:"numero"`
	Texto   string   `json:"texto"`
	Status  string   `json:"status"`
	Alineas []Alinea `json:"alineas,omitempty"`
}

// Alinea represents an Alinea node
type Alinea struct {
	Numero string `json:"numero"`
	Texto  string `json:"texto"`
	Status string `json:"status"`
}
