package graph

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// LegalDocument mirrors the parsed JSON structure from the parser
type LegalDocument struct {
	ID          string      `json:"id"`
	Tipo        string      `json:"tipo"`
	Esfera      string      `json:"esfera"`
	Numero      string      `json:"numero"`
	Ano         string      `json:"ano"`
	Titulo      string      `json:"titulo"`
	Ementa      string      `json:"ementa"`
	URLOrigem   string      `json:"url_origem"`
	URLPai      string      `json:"url_pai,omitempty"`
	Artigos     json.RawMessage `json:"artigos,omitempty"`
	Referencias []Reference `json:"referencias,omitempty"`
}

type Reference struct {
	ID      string       `json:"id"`
	Context *LinkContext `json:"context,omitempty"`
}

type LinkContext struct {
	Artigo    string `json:"artigo,omitempty"`
	Paragrafo string `json:"paragrafo,omitempty"`
	Inciso    string `json:"inciso,omitempty"`
	Alinea    string `json:"alinea,omitempty"`
}

// Neo4jClient handles all graph database operations
type Neo4jClient struct {
	driver neo4j.DriverWithContext
	dbName string
}

// NewNeo4jClient creates a new Neo4j client
func NewNeo4jClient(uri, username, password, dbName string) (*Neo4jClient, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	log.Printf("Connected to Neo4j at %s", uri)

	return &Neo4jClient{
		driver: driver,
		dbName: dbName,
	}, nil
}

// UpsertDocument creates or updates a LegalDocument node with a topic label,
// and connects it to a central Topic node to organize the graph visually.
func (c *Neo4jClient) UpsertDocument(ctx context.Context, doc *LegalDocument, topic string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.dbName})
	defer session.Close(ctx)

	// Sanitize topic for use as a Neo4j label (capitalize, remove special chars)
	label := sanitizeLabel(topic)
	
	// Create a secondary label based on the document type (e.g., Lei, Decreto, Artigo) for visual distinction
	subLabel := sanitizeLabel(doc.Tipo)
	if subLabel == "" {
		subLabel = "Documento"
	}

	// MERGE the root Origin node (the sun center)
	// MERGE the central Topic node
	// Connect Topic to Origin
	query := `
		MERGE (o:Origem {name: "Brasil"})
		MERGE (t:Topic {name: $topic})
		MERGE (t)-[:PERTENCE]->(o)
	`
	_, err := session.Run(ctx, query, map[string]any{"topic": topic})
	if err != nil {
		return err
	}

	// MERGE the LegalDocument node with its specific labels
	// If it has a URLPai, link it to the parent. Otherwise, link to the Topic.
	parentID := ""
	if doc.URLPai != "" {
		parentID = GetIDFromURL(doc.URLPai)
	}

	artigos := ""
	if doc.Artigos != nil {
		artigos = string(doc.Artigos)
	}

	var cypher string
	params := map[string]any{
		"id":         doc.ID,
		"tipo":       doc.Tipo,
		"esfera":     doc.Esfera,
		"numero":     doc.Numero,
		"ano":        doc.Ano,
		"titulo":     doc.Titulo,
		"ementa":     doc.Ementa,
		"url_origem": doc.URLOrigem,
		"url_pai":    doc.URLPai,
		"artigos":    artigos,
		"topic":      topic,
	}

	if parentID != "" {
		// Link to parent
		cypher = fmt.Sprintf(`
			MERGE (d:LegalDocument:%s:%s {id: $id})
			MERGE (p:LegalDocument {id: $parent_id})
			MERGE (d)-[:CONTIDO_EM]->(p)
			SET d.tipo      = $tipo,
			    d.esfera    = $esfera,
			    d.numero    = $numero,
			    d.ano       = $ano,
			    d.titulo    = $titulo,
			    d.ementa    = $ementa,
			    d.url_origem = $url_origem,
			    d.url_pai   = $url_pai,
			    d.artigos   = $artigos,
			    d.topic     = $topic
		`, label, subLabel)
		params["parent_id"] = parentID
	} else {
		// Link to Topic (Root document of a law/code)
		cypher = fmt.Sprintf(`
			MERGE (t:Topic {name: $topic})
			MERGE (d:LegalDocument:%s:%s {id: $id})
			MERGE (d)-[:PERTENCE]->(t)
			SET d.tipo      = $tipo,
			    d.esfera    = $esfera,
			    d.numero    = $numero,
			    d.ano       = $ano,
			    d.titulo    = $titulo,
			    d.ementa    = $ementa,
			    d.url_origem = $url_origem,
			    d.url_pai   = $url_pai,
			    d.artigos   = $artigos,
			    d.topic     = $topic
		`, label, subLabel)
	}

	_, err = session.Run(ctx, cypher, params)
	return err
}

// CreateReferences creates REFERENCIA relationships between documents
func (c *Neo4jClient) CreateReferences(ctx context.Context, docID string, refs []Reference) error {
	if len(refs) == 0 {
		return nil
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.dbName})
	defer session.Close(ctx)

	for _, ref := range refs {
		// MERGE the target node (it may not exist yet — placeholder)
		// MERGE the target node (it may not exist yet — placeholder)
		// We add :Documento as a fallback sub-label so it gets a distinct color even before it's fully parsed
		query := `
			MATCH (source:LegalDocument {id: $source_id})
			MERGE (target:LegalDocument:Documento {id: $target_id})
			MERGE (source)-[r:REFERENCIA]->(target)
			SET r.artigo    = $artigo,
			    r.paragrafo = $paragrafo,
			    r.inciso    = $inciso,
			    r.alinea    = $alinea
		`

		artigo, paragrafo, inciso, alinea := "", "", "", ""
		if ref.Context != nil {
			artigo = ref.Context.Artigo
			paragrafo = ref.Context.Paragrafo
			inciso = ref.Context.Inciso
			alinea = ref.Context.Alinea
		}

		_, err := session.Run(ctx, query, map[string]any{
			"source_id": docID,
			"target_id": ref.ID,
			"artigo":    artigo,
			"paragrafo": paragrafo,
			"inciso":    inciso,
			"alinea":    alinea,
		})
		if err != nil {
			log.Printf("Failed to create reference %s -> %s: %v", docID, ref.ID, err)
		}
	}

	return nil
}

// Close shuts down the Neo4j driver
func (c *Neo4jClient) Close(ctx context.Context) {
	c.driver.Close(ctx)
}

// GetIDFromURL returns the same deterministic ID used by the crawler for a given URL
func GetIDFromURL(u string) string {
	h := sha1.New()
	h.Write([]byte(u))
	return fmt.Sprintf("%x", h.Sum(nil))[:10]
}

// sanitizeLabel converts a topic string to a valid Neo4j label
func sanitizeLabel(topic string) string {
	// Remove special characters, capitalize each word
	parts := strings.FieldsFunc(topic, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	// If single short word (like "ctb"), uppercase all
	if len(parts) == 1 && len(parts[0]) <= 4 {
		return strings.ToUpper(parts[0])
	}

	// Otherwise, capitalize each word
	var result strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			result.WriteString(strings.ToUpper(p[:1]))
			result.WriteString(strings.ToLower(p[1:]))
		}
	}
	return result.String()
}
