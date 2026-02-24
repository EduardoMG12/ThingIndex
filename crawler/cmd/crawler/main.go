package main

import (
	"log"
	"strings"

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
	
	// Utilizing queue name as bucket or folder name, but replacing `_` with `-` and lowercasing to comply with S3 rules
	bucketName := strings.ToLower(strings.ReplaceAll(cfg.QueueName, "_", "-"))
	log.Printf("Initializing Minio storage with bucket name: '%s'", bucketName)
	
	s, err := storage.NewStorage(storage.Config{
		Type:           cfg.StorageType,
		LocalDir:       cfg.LocalDir,
		MinioEndpoint:  cfg.MinioEndpoint,
		MinioAccessKey: cfg.MinioAccessKey,
		MinioSecretKey: cfg.MinioSecretKey,
		MinioUseSSL:    cfg.MinioUseSSL,
		BucketName:     bucketName,
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
	c := crawler.NewCrawlerService(f, pdfP, htmlP, s, q, cfg.MaxDepth, cfg.IgnoreKeywords, cfg.FilterName, bucketName, cfg.BackpressureLimit)

	// Start Crawling
	log.Printf("Starting crawler, reading from queue...")
	if err := c.Crawl(); err != nil {
		log.Fatalf("Crawler failed: %v", err)
	}
	log.Println("Crawling finished successfully.")
}
