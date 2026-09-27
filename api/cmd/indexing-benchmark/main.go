package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"thing-index/internal/usecase"
)

func main() {
	report, err := usecase.NewIndexingBenchmarkProvider(usecase.RetrievalOptionsFromEnv()).GetPublicIndexingBenchmark(context.Background())
	if err != nil {
		log.Fatalf("failed to build indexing benchmark report: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		log.Fatalf("failed to encode indexing benchmark report: %v", err)
	}
}
