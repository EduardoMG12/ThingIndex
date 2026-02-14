package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	// Load env just like populator
	if err := godotenv.Load(); err != nil {
		godotenv.Load("../../.env")
	}

	uri := os.Getenv("NEO4J_URI")
	user := os.Getenv("NEO4J_USERNAME")
	pass := os.Getenv("NEO4J_PASSWORD")

	if uri == "" {
		log.Fatal("NEO4J_URI not set")
	}

	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pass, ""))
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// 1. Count Nodes by Label
	printHeader("Node Counts")
	runQuery(ctx, session, "MATCH (n) RETURN head(labels(n)) as Label, count(n) as Count ORDER BY Count DESC")

	// 2. Count Relationships
	printHeader("Relationship Counts")
	runQuery(ctx, session, "MATCH ()-[r]->() RETURN type(r) as Type, count(r) as Count ORDER BY Count DESC")

	// 3. Sample Hierarchy (Law -> Articles)
	printHeader("Sample Hierarchy (Top 5 Laws by Article Count)")
	runQuery(ctx, session, `
		MATCH (l:Lei)-[:CONTEM]->(a:Artigo)
		RETURN l.numero as Lei, count(a) as Artigos
		ORDER BY Artigos DESC LIMIT 5
	`)
	// 4. Sample Deepest Path (Law -> Article -> Paragraph -> Inciso -> Alinea)
	printHeader("Sample Deepest Path (Full Hierarchy)")
	runQuery(ctx, session, `
		MATCH (l:Lei)-[:CONTEM]->(a:Artigo)-[:CONTEM]->(p:Paragrafo)-[:CONTEM]->(i:Inciso)-[:CONTEM]->(al:Alinea)
		RETURN l.numero as Lei, a.numero as Art, p.numero as Par, i.numero as Inc, al.numero as Ali
		LIMIT 5
	`)
	// 5. Sample Cross-References (Law -> Article -> Cites -> Law)
	printHeader("Sample Cross-References (Citations)")
	runQuery(ctx, session, `
		MATCH (l1:Lei)-[:CONTEM]->(a:Artigo)-[:CITA]->(l2:Lei)
		RETURN l1.numero as Lei_Origem, a.numero as Artigo, l2.numero as Lei_Citada
		LIMIT 10
	`)

	// 6. Check for "Ghost" Nodes (Created by MERGE but not fully populated yet)
	printHeader("Ghost Nodes (Laws referenced but not yet ingested)")
	runQuery(ctx, session, `
		MATCH (l:Lei)
		WHERE l.titulo IS NULL AND l.status = 'unknown'
		RETURN l.numero as Lei_Fantasma, count(l)
		LIMIT 10
	`)
}

func printHeader(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}

func runQuery(ctx context.Context, session neo4j.SessionWithContext, cypher string) {
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, cypher, nil)
		if err != nil {
			return nil, err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		
		// Print keys (headers) is hard without peeking, but we know what we asked for usually.
		// For generic printing, we'd need to fetch one record or use Keys() from result
		// But result.Keys() is available immediately after Run.
		keys, _ := result.Keys()
		for i, k := range keys {
			fmt.Fprintf(w, "%s", k)
			if i < len(keys)-1 {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintf(w, "\n")

		for result.Next(ctx) {
			record := result.Record()
			for i, val := range record.Values {
				fmt.Fprintf(w, "%v", val)
				if i < len(record.Values)-1 {
					fmt.Fprintf(w, "\t")
				}
			}
			fmt.Fprintf(w, "\n")
		}
		w.Flush()
		return nil, result.Err()
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
	}
}
