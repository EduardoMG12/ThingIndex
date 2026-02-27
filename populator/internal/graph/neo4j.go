package graph

import (
	"context"
	"crypto/sha1"
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
	Artigos     []Article   `json:"artigos,omitempty"`
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
// It also creates granular nodes for Artigos, Paragrafos, Incisos, and Alineas.
func (c *Neo4jClient) UpsertDocument(ctx context.Context, doc *LegalDocument, topic string) error {
	if strings.TrimSpace(topic) == "" {
		topic = "Geral"
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.dbName})
	defer session.Close(ctx)

	// Sanitize topic for use as a Neo4j label (capitalize, remove special chars)
	label := sanitizeLabel(topic)
	
	// Create a secondary label based on the document type (e.g., Lei, Decreto, Artigo) for visual distinction
	subLabel := sanitizeLabel(doc.Tipo)
	if subLabel == "" {
		subLabel = "Documento"
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// MERGE the root Origin node (the sun center)
		// MERGE the central Topic node
		// Connect Topic to Origin
		if _, err := tx.Run(ctx, QueryCreateTopic, map[string]any{"topic": topic}); err != nil {
			return nil, err
		}

		// MERGE the LegalDocument node with its specific labels
		// If it has a URLPai, link it to the parent. Otherwise, link to the Topic.
		parentID := ""
		if doc.URLPai != "" {
			parentID = GetIDFromURL(doc.URLPai)
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
			"artigos":    "", // Deprecated raw access
			"topic":      topic,
		}

		if parentID != "" {
			cypher = fmt.Sprintf(QueryCreateDocLinkedToParent, label, subLabel)
			params["parent_id"] = parentID
		} else {
			cypher = fmt.Sprintf(QueryCreateDocLinkedToTopic, label, subLabel)
		}

		if _, err := tx.Run(ctx, cypher, params); err != nil {
			return nil, err
		}

		// Process Articles hierarchically
		for _, art := range doc.Artigos {
			artID := fmt.Sprintf("%s_art_%s", doc.ID, art.Numero)
			paramsArt := map[string]any{
				"lawID":  doc.ID,
				"id":     artID,
				"numero": art.Numero,
				"caput":  art.Caput,
				"status": art.Status,
				"topic":  topic,
			}
			if _, err := tx.Run(ctx, QueryCreateArticle, paramsArt); err != nil {
				return nil, fmt.Errorf("failed to create Article %s: %w", art.Numero, err)
			}

			// Process Paragraphs inside Article
			for _, par := range art.Paragrafos {
				parID := fmt.Sprintf("%s_par_%s", artID, par.Numero)
				paramsPar := map[string]any{
					"artID":  artID,
					"id":     parID,
					"numero": par.Numero,
					"texto":  par.Texto,
					"status": par.Status,
					"topic":  topic,
				}
				if _, err := tx.Run(ctx, QueryCreateParagraph, paramsPar); err != nil {
					return nil, fmt.Errorf("failed to create Paragraph %s: %w", par.Numero, err)
				}

				// Incisos inside Paragraph
				if err := processIncisos(ctx, tx, parID, "Paragrafo", par.Incisos, topic); err != nil {
					return nil, err
				}
			}

			// Incisos directly inside Article
			if err := processIncisos(ctx, tx, artID, "Artigo", art.Incisos, topic); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

// processIncisos handles the creation of Incisos and their Alineas
func processIncisos(ctx context.Context, tx neo4j.ManagedTransaction, parentID string, parentLabel string, incisos []Inciso, topic string) error {
	for _, inc := range incisos {
		incID := fmt.Sprintf("%s_inc_%s", parentID, inc.Numero)
		queryInc := fmt.Sprintf(QueryCreateInciso, parentLabel)

		paramsInc := map[string]any{
			"parentID": parentID,
			"id":       incID,
			"numero":   inc.Numero,
			"texto":    inc.Texto,
			"status":   inc.Status,
			"topic":    topic,
		}
		if _, err := tx.Run(ctx, queryInc, paramsInc); err != nil {
			return fmt.Errorf("failed to create Inciso %s: %w", inc.Numero, err)
		}

		for _, alinea := range inc.Alineas {
			alineaID := fmt.Sprintf("%s_ali_%s", incID, alinea.Numero)
			paramsAli := map[string]any{
				"incID":  incID,
				"id":     alineaID,
				"numero": alinea.Numero,
				"texto":  alinea.Texto,
				"status": alinea.Status,
				"topic":  topic,
			}
			if _, err := tx.Run(ctx, QueryCreateAlinea, paramsAli); err != nil {
				return fmt.Errorf("failed to create Alinea %s: %w", alinea.Numero, err)
			}
		}
	}
	return nil
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
		// We add :Documento as a fallback sub-label so it gets a distinct color even before it's fully parsed
		artigo, paragrafo, inciso, alinea := "", "", "", ""
		if ref.Context != nil {
			artigo = ref.Context.Artigo
			paragrafo = ref.Context.Paragrafo
			inciso = ref.Context.Inciso
			alinea = ref.Context.Alinea
		}

		_, err := session.Run(ctx, QueryCreateReference, map[string]any{
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
