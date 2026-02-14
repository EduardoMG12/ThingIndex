package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/user/legisgraph/graph-populator/internal/db"
	"github.com/user/legisgraph/graph-populator/internal/model"
	"github.com/user/legisgraph/graph-populator/internal/parser"
)

func main() {
	log.Println("Starting graph-populator...")
	// Load from .env file (try multiple locations or default)
	if err := godotenv.Load(); err != nil {
		// Fallback: try looking in parent directories if running from cmd/...
		if err := godotenv.Load("../../.env"); err != nil {
			log.Println("Warning: .env file not found, relying on system env vars")
		}
	}

	uri := os.Getenv("NEO4J_URI")
	user := os.Getenv("NEO4J_USERNAME")
	pass := os.Getenv("NEO4J_PASSWORD")

	if uri == "" || user == "" || pass == "" {
		log.Fatal("Missing NEO4J_URI, NEO4J_USERNAME, or NEO4J_PASSWORD environment variables")
	}

	client, err := db.NewNeo4jClient(uri, user, pass)
	if err != nil {
		log.Fatalf("Failed to initialize Neo4j client: %v", err)
	}
	defer client.Close(context.Background())

	ctx := context.Background()
	if err := client.EnsureConstraints(ctx); err != nil {
		log.Fatalf("Failed to ensure constraints: %v", err)
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		// Default to relative path assuming running from graph-populator root
		dataDir = "../legal-parser/parsed_data"
	}

	files, err := filepath.Glob(filepath.Join(dataDir, "*.json"))
	if err != nil {
		log.Fatalf("Failed to scan directory %s: %v", dataDir, err)
	}

	if len(files) == 0 {
		// Try absolute path if relative failed or found nothing, just in case
		absDir, _ := filepath.Abs(dataDir)
		log.Printf("Warning: No JSON files found in %s (Absolute: %s). Please check DATA_DIR.", dataDir, absDir)
	}

	log.Printf("Found %d files to process in %s", len(files), dataDir)

	for _, file := range files {
		processFile(client, file)
	}
}

func processFile(client *db.Neo4jClient, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file %s: %v", path, err)
		return
	}

	var law model.Law
	if err := json.Unmarshal(data, &law); err != nil {
		log.Printf("Error parsing JSON %s: %v", path, err)
		return
	}

	fmt.Printf("Processing Law: %s (%s)\n", law.Titulo, law.ID)

	if err := client.SaveLaw(context.Background(), law); err != nil {
		log.Printf("Failed to save law %s: %v", law.ID, err)
	} else {
		log.Printf("Successfully saved law structure: %s", law.ID)
	}

	// Process Cross-References (Horizontal Links)
	// 1. Articles
	for _, art := range law.Artigos {
		artID := fmt.Sprintf("%s_art_%s", law.ID, art.Numero)
		citations := parser.ExtractCitations(art.Caput)
		for _, cit := range citations {
			if err := client.LinkArticleToLaw(context.Background(), artID, cit.TargetType, cit.TargetNum); err != nil {
				log.Printf("Failed to link article %s to %s %s: %v", artID, cit.TargetType, cit.TargetNum, err)
			} else {
				log.Printf("Linked %s -> %s %s", artID, cit.TargetType, cit.TargetNum)
			}
		}

		// 2. Paragraphs
		for _, par := range art.Paragrafos {
			// Linking from Paragraph is valid, but our LinkArticleToLaw currently expects ArticleID.
			// For simplicity, we can link the PARAGRAPH's parent ARTICLE to the target Law,
			// OR we update neo4j.go to allow linking Paragraphs.
			// Given the user request usually implies "This article/law cites...", linking the Article is a safe default.
			// But specialized "LinkParagraphToLaw" would be better.
			// For now, let's extract citations from paragraphs and link the PARENT ARTICLE.
			// This maintains the "Article X cites Law Y" logic even if the text is in the §.
			citations := parser.ExtractCitations(par.Texto)
			for _, cit := range citations {
				if err := client.LinkArticleToLaw(context.Background(), artID, cit.TargetType, cit.TargetNum); err != nil {
					log.Printf("Failed to link article %s (via par %s) to %s %s: %v", artID, par.Numero, cit.TargetType, cit.TargetNum, err)
				}
			}
		}
	}
}
