package main

import (
	"log"

	"github.com/user/legisgraph-crawler/internal/config"
	"github.com/user/legisgraph-crawler/internal/crawler"
	"github.com/user/legisgraph-crawler/internal/fetcher"
	"github.com/user/legisgraph-crawler/internal/parser"
	"github.com/user/legisgraph-crawler/internal/queue"
	"github.com/user/legisgraph-crawler/internal/storage"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize components
	f := fetcher.NewHTTPFetcher(cfg.Timeout, 3) 
	pdfP := parser.NewPDFParser()
	htmlP := parser.NewHTMLParser()
	
	s, err := storage.NewStorage(storage.Config{
		Type:     cfg.StorageType,
		LocalDir: cfg.LocalDir,
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize Queue
	q, err := queue.NewRabbitMQClient(cfg.RabbitURL, cfg.QueueName)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}
	if err := q.Connect(); err != nil {
		log.Fatalf("Failed to connect to queue: %v", err)
	}
	defer q.Close()

	// Initialize Crawler
	c := crawler.NewCrawlerService(f, pdfP, htmlP, s, q, cfg.MaxDepth, cfg.IgnoreKeywords, cfg.FilterName)

	// Start Crawling
	log.Printf("Starting crawler, reading from queue...")
	if err := c.Crawl(); err != nil {
		log.Fatalf("Crawler failed: %v", err)
	}
	log.Println("Crawling finished successfully.")
}
