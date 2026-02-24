package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	inputMinio "github.com/user/legisgraph/legal-parser/internal/adapters/input/minio"
	"github.com/user/legisgraph/legal-parser/internal/queue"
	"github.com/user/legisgraph/legal-parser/internal/service"
)

// RawDocumentTask is the per-file message from the crawler
type RawDocumentTask struct {
	ContentKey  string `json:"content_key"`
	MetadataKey string `json:"metadata_key"`
	Bucket      string `json:"bucket"`
	Topic       string `json:"topic"`
}

func main() {
	log.Printf("Legal Parser starting in streaming mode...")

	// --- Environment ---
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	inputQueue := getEnv("PARSER_INPUT_QUEUE", "raw_documents")
	outputQueue := getEnv("PARSER_OUTPUT_QUEUE", "parsed_documents")
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	minioUseSSL := strings.ToLower(os.Getenv("MINIO_USE_SSL")) == "true"

	if rabbitmqURL == "" || minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" {
		log.Fatalf("Critical environment variables missing (RABBITMQ_URL, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY)")
	}

	// --- RabbitMQ ---
	q := queue.NewRabbitMQClient(rabbitmqURL, inputQueue)
	if err := q.Connect(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer q.Close()

	msgs, err := q.Consume()
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	log.Printf("Waiting for documents on queue '%s'...", inputQueue)

	// --- Consumer loop (blocks forever, processes 1 file at a time) ---
	for msg := range msgs {
		var task RawDocumentTask
		if err := json.Unmarshal(msg.Body, &task); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			msg.Nack(false, false)
			continue
		}

		log.Printf("Received: content=%s topic=%s bucket=%s", task.ContentKey, task.Topic, task.Bucket)

		// Skip non-HTML files (PDFs don't have parseable legal text structure)
		lower := strings.ToLower(task.ContentKey)
		if !strings.HasSuffix(lower, ".htm") && !strings.HasSuffix(lower, ".html") {
			log.Printf("Skipping non-HTML file: %s", task.ContentKey)
			msg.Ack(false)
			continue
		}

		// --- Connect to MinIO for this bucket ---
		source, err := inputMinio.NewMinioSource(minioEndpoint, minioAccessKey, minioSecretKey, minioUseSSL, task.Bucket)
		if err != nil {
			log.Printf("Failed to connect to MinIO bucket '%s': %v", task.Bucket, err)
			msg.Nack(false, true) // requeue
			continue
		}

		// --- Read content and metadata from MinIO ---
		content, err := source.GetContentByKey(task.ContentKey)
		if err != nil {
			log.Printf("Failed to read content '%s': %v", task.ContentKey, err)
			msg.Nack(false, false)
			continue
		}

		meta, err := source.GetMetadataByKey(task.MetadataKey)
		if err != nil {
			log.Printf("Failed to read metadata '%s': %v", task.MetadataKey, err)
			msg.Nack(false, false)
			continue
		}

		// --- Parse the document ---
		parser := service.NewParserService(nil, nil) // source/output not needed for ProcessSingleDocument
		canonicalID, jsonData, err := parser.ProcessSingleDocument(content, meta)
		if err != nil {
			log.Printf("[SKIP] %s: %v", task.ContentKey, err)
			msg.Ack(false) // don't requeue parse failures
			continue
		}

		// --- Save parsed JSON to MinIO under parsed/<topic>/ prefix ---
		parsedKey := fmt.Sprintf("parsed/%s/%s.json", task.Topic, canonicalID)
		if err := source.SaveParsed(parsedKey, jsonData); err != nil {
			log.Printf("Failed to save parsed document '%s': %v", parsedKey, err)
			msg.Nack(false, true) // requeue
			continue
		}

		// --- Publish to parsed_documents queue for the populator ---
		outPayload := map[string]string{
			"parsed_key":   parsedKey,
			"bucket":       task.Bucket,
			"topic":        task.Topic,
			"canonical_id": canonicalID,
		}
		outBytes, _ := json.Marshal(outPayload)

		if err := q.PublishToQueue(outputQueue, outBytes); err != nil {
			log.Printf("Failed to publish to '%s': %v", outputQueue, err)
			// Don't fail the message — the parsed file is saved
		} else {
			log.Printf("Published to %s: %s (topic=%s)", outputQueue, canonicalID, task.Topic)
		}

		msg.Ack(false)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
