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
		for _, q := range QueryEnsureConstraints {
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
		if _, err := tx.Run(ctx, QueryCreateLaw, paramsLaw); err != nil {
			return nil, fmt.Errorf("failed to create Law node: %w", err)
		}

		// 2. Process Articles
		for _, art := range law.Artigos {
			// Article ID: lawID + "_art_" + numero
			artID := fmt.Sprintf("%s_art_%s", law.ID, art.Numero)
			paramsArt := map[string]any{
				"lawID":  law.ID,
				"id":     artID,
				"numero": art.Numero,
				"caput":  art.Caput,
				"status": art.Status,
			}
			if _, err := tx.Run(ctx, QueryCreateArticle, paramsArt); err != nil {
				return nil, fmt.Errorf("failed to create Article %s: %w", art.Numero, err)
			}

			// 2a. Process Paragraphs
			for _, par := range art.Paragrafos {
				parID := fmt.Sprintf("%s_par_%s", artID, par.Numero)
				paramsPar := map[string]any{
					"artID":  artID,
					"id":     parID,
					"numero": par.Numero,
					"texto":  par.Texto,
					"status": par.Status,
				}
				if _, err := tx.Run(ctx, QueryCreateParagraph, paramsPar); err != nil {
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
		queryInc := fmt.Sprintf(QueryCreateInciso, parentLabel)

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
			paramsAli := map[string]any{
				"incID":  incID,
				"id":     alineaID,
				"numero": alinea.Numero,
				"texto":  alinea.Texto,
				"status": alinea.Status,
			}
			if _, err := tx.Run(ctx, QueryCreateAlinea, paramsAli); err != nil {
				return fmt.Errorf("failed to create Alinea %s: %w", alinea.Numero, err)
			}
		}
	}
	return nil
}

// LinkArticleToLaw creates a CITATION relationship between an Article and a referenced Law.
// It uses MERGE on the target Law, creating a "shell" node if it doesn't exist yet.
func (c *Neo4jClient) LinkArticleToLaw(ctx context.Context, articleID string, targetLawType, targetLawNum string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		params := map[string]any{
			"articleID":  articleID,
			"targetNum":  targetLawNum,
			"targetType": targetLawType,
		}

		if _, err := tx.Run(ctx, QueryLinkArticleToLaw, params); err != nil {
			return nil, err
		}
		return nil, nil
	})

	return err
}
