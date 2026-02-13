package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/user/legisgraph-crawler/internal/crawler"
	"github.com/user/legisgraph-crawler/internal/fetcher"
	"github.com/user/legisgraph-crawler/internal/parser"
	"github.com/user/legisgraph-crawler/internal/storage"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults or system env vars")
	}

	startURL := os.Getenv("CRAWLER_START_URL")
	if startURL == "" {
		log.Fatal("CRAWLER_START_URL is required")
	}

	timeoutStr := os.Getenv("CRAWLER_TIMEOUT")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 10 * time.Second
	}

	maxDepthStr := os.Getenv("CRAWLER_MAX_DEPTH")
	maxDepth, err := strconv.Atoi(maxDepthStr)
	if err != nil {
		maxDepth = 20 // Higher default depth
	}

	storageType := os.Getenv("STORAGE_TYPE")
	if storageType == "" {
		storageType = "local"
	}

	localDir := os.Getenv("STORAGE_LOCAL_DIR")
	if localDir == "" {
		localDir = "./output_pdfs"
	}

	ignoreKeywordsStr := os.Getenv("CRAWLER_IGNORE_KEYWORDS")
	var ignoreKeywords []string
	if ignoreKeywordsStr != "" {
		parts := strings.Split(ignoreKeywordsStr, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				ignoreKeywords = append(ignoreKeywords, trimmed)
			}
		}
	} else {
		// Default ignore keywords (noise)
		ignoreKeywords = []string{
			"clic-clac", "pesquisar", "revogado", "vetado", "clicclac",
		}
	}

	filterName := os.Getenv("CRAWLER_FILTER_NAME")

	// Initialize components
	f := fetcher.NewHTTPFetcher(timeout, 3) 
	pdfP := parser.NewPDFParser()
	htmlP := parser.NewHTMLParser()
	
	s, err := storage.NewStorage(storage.Config{
		Type:     storageType,
		LocalDir: localDir,
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize Crawler
	c := crawler.NewCrawlerService(f, pdfP, htmlP, s, maxDepth, ignoreKeywords, filterName)

	// Start Crawling
	log.Printf("Starting crawler at %s", startURL)
	if err := c.Crawl(startURL); err != nil {
		log.Fatalf("Crawler failed: %v", err)
	}
	log.Println("Crawling finished successfully.")
}
