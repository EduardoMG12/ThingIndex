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
		log.Printf("Successfully saved law: %s", law.ID)
	}
}
