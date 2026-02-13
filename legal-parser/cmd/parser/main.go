package main

import (
	"flag"
	"log"

	inputLocal "github.com/user/legisgraph/legal-parser/internal/adapters/input/local"
	outputLocal "github.com/user/legisgraph/legal-parser/internal/adapters/output/local"
	"github.com/user/legisgraph/legal-parser/internal/service"
)

func main() {
	// CLI flags
	inputDir := flag.String("input", "./raw_data", "Directory containing raw HTML/JSON files from the crawler")
	outputDir := flag.String("output", "./parsed_data", "Directory to write parsed JSON documents")
	flag.Parse()

	log.Printf("Legal Parser starting...")
	log.Printf("  Input:  %s", *inputDir)
	log.Printf("  Output: %s", *outputDir)

	// --- Dependency Injection ---
	// Swap these adapters to read from S3, GCS, etc.
	// The service layer knows NOTHING about where files come from.
	source := inputLocal.NewFileSystemSource(*inputDir)
	output := outputLocal.NewFileSystemOutput(*outputDir)

	// Wire up the service
	parser := service.NewParserService(source, output)

	// Run
	if err := parser.ProcessAll(); err != nil {
		log.Fatalf("Parser failed: %v", err)
	}

	log.Println("Legal Parser finished successfully.")
}
