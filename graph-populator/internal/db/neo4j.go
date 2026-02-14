package db

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/user/legisgraph/graph-populator/internal/model"
)

type Neo4jClient struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jClient(uri, username, password string) (*Neo4jClient, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify connectivity: %w", err)
	}

	return &Neo4jClient{driver: driver}, nil
}

func (c *Neo4jClient) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// EnsureConstraints creates unique constraints for the graph schema
func (c *Neo4jClient) EnsureConstraints(ctx context.Context) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		queries := []string{
			"CREATE CONSTRAINT IF NOT EXISTS FOR (l:Lei) REQUIRE l.id IS UNIQUE",
			"CREATE INDEX IF NOT EXISTS FOR (l:Lei) ON (l.numero)",
		}

		for _, q := range queries {
			if _, err := tx.Run(ctx, q, nil); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

// SaveLaw persists a Law and its structural hierarchy into the graph.
func (c *Neo4jClient) SaveLaw(ctx context.Context, law model.Law) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// 1. Create Law Node
		queryLaw := `
			MERGE (l:Lei {id: $id})
			SET l.tipo = $tipo,
				l.esfera = $esfera,
				l.numero = $numero,
				l.ano = $ano,
				l.titulo = $titulo,
				l.ementa = $ementa,
				l.url_origem = $url_origem,
				l.url_pai = $url_pai
		`
		paramsLaw := map[string]any{
			"id":         law.ID,
			"tipo":       law.Tipo,
			"esfera":     law.Esfera,
			"numero":     law.Numero,
			"ano":        law.Ano,
			"titulo":     law.Titulo,
			"ementa":     law.Ementa,
			"url_origem": law.URLOrigem,
			"url_pai":    law.URLPai,
		}
		if _, err := tx.Run(ctx, queryLaw, paramsLaw); err != nil {
			return nil, fmt.Errorf("failed to create Law node: %w", err)
		}

		// 2. Process Articles
		for _, art := range law.Artigos {
			// Article ID: lawID + "_art_" + numero
			artID := fmt.Sprintf("%s_art_%s", law.ID, art.Numero)
			queryArt := `
				MATCH (l:Lei {id: $lawID})
				MERGE (a:Artigo {id: $id})
				SET a.numero = $numero,
					a.caput = $caput,
					a.status = $status
				MERGE (l)-[:CONTEM]->(a)
			`
			paramsArt := map[string]any{
				"lawID":  law.ID,
				"id":     artID,
				"numero": art.Numero,
				"caput":  art.Caput,
				"status": art.Status,
			}
			if _, err := tx.Run(ctx, queryArt, paramsArt); err != nil {
				return nil, fmt.Errorf("failed to create Article %s: %w", art.Numero, err)
			}

			// 2a. Process Paragraphs
			for _, par := range art.Paragrafos {
				parID := fmt.Sprintf("%s_par_%s", artID, par.Numero)
				queryPar := `
					MATCH (a:Artigo {id: $artID})
					MERGE (p:Paragrafo {id: $id})
					SET p.numero = $numero,
						p.texto = $texto,
						p.status = $status
					MERGE (a)-[:CONTEM]->(p)
				`
				paramsPar := map[string]any{
					"artID":  artID,
					"id":     parID,
					"numero": par.Numero,
					"texto":  par.Texto,
					"status": par.Status,
				}
				if _, err := tx.Run(ctx, queryPar, paramsPar); err != nil {
					return nil, fmt.Errorf("failed to create Paragraph %s: %w", par.Numero, err)
				}

				// 2a-i. Incisos inside Paragraphs
				if err := processIncisos(ctx, tx, parID, "Paragrafo", par.Incisos); err != nil {
					return nil, err
				}
			}

			// 2b. Process Incisos directly in Article
			if err := processIncisos(ctx, tx, artID, "Artigo", art.Incisos); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

// processIncisos handles the creation of Incisos and their potential Alineas
func processIncisos(ctx context.Context, tx neo4j.ManagedTransaction, parentID string, parentLabel string, incisos []model.Inciso) error {
	for _, inc := range incisos {
		incID := fmt.Sprintf("%s_inc_%s", parentID, inc.Numero)
		queryInc := fmt.Sprintf(`
			MATCH (p:%s {id: $parentID})
			MERGE (i:Inciso {id: $id})
			SET i.numero = $numero,
				i.texto = $texto,
				i.status = $status
			MERGE (p)-[:CONTEM]->(i)
		`, parentLabel)

		paramsInc := map[string]any{
			"parentID": parentID,
			"id":       incID,
			"numero":   inc.Numero,
			"texto":    inc.Texto,
			"status":   inc.Status,
		}
		if _, err := tx.Run(ctx, queryInc, paramsInc); err != nil {
			return fmt.Errorf("failed to create Inciso %s: %w", inc.Numero, err)
		}

		// Process Alineas using a similar helper or inline
		for _, alinea := range inc.Alineas {
			alineaID := fmt.Sprintf("%s_ali_%s", incID, alinea.Numero)
			queryAli := `
				MATCH (i:Inciso {id: $incID})
				MERGE (al:Alinea {id: $id})
				SET al.numero = $numero,
					al.texto = $texto,
					al.status = $status
				MERGE (i)-[:CONTEM]->(al)
			`
			paramsAli := map[string]any{
				"incID":  incID,
				"id":     alineaID,
				"numero": alinea.Numero,
				"texto":  alinea.Texto,
				"status": alinea.Status,
			}
			if _, err := tx.Run(ctx, queryAli, paramsAli); err != nil {
				return fmt.Errorf("failed to create Alinea %s: %w", alinea.Numero, err)
			}
		}
	}
	return nil
}
