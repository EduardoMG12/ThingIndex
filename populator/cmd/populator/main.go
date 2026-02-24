package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/user/legisgraph/populator/internal/graph"
	"github.com/user/legisgraph/populator/internal/queue"
)

// ParsedDocumentTask is the message from the parser
type ParsedDocumentTask struct {
	ParsedKey   string `json:"parsed_key"`
	Bucket      string `json:"bucket"`
	Topic       string `json:"topic"`
	CanonicalID string `json:"canonical_id"`
}

func main() {
	log.Printf("Populator starting...")

	// --- Environment ---
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	queueName := getEnv("POPULATOR_QUEUE", "parsed_documents")
	neo4jURI := os.Getenv("NEO4J_URI")
	neo4jUser := os.Getenv("NEO4J_USER")
	neo4jPass := os.Getenv("NEO4J_PASS")
	neo4jDB := getEnv("NEO4J_DATABASE", "neo4j")
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	minioUseSSL := strings.ToLower(os.Getenv("MINIO_USE_SSL")) == "true"

	if rabbitmqURL == "" || neo4jURI == "" || neo4jPass == "" || minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" {
		log.Fatalf("Critical environment variables missing (RABBITMQ_URL, NEO4J_URI, NEO4J_PASS, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY)")
	}

	// --- Neo4j ---
	neo4jClient, err := graph.NewNeo4jClient(neo4jURI, neo4jUser, neo4jPass, neo4jDB)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer neo4jClient.Close(context.Background())

	// --- MinIO ---
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: minioUseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	// --- RabbitMQ ---
	q := queue.NewRabbitMQClient(rabbitmqURL, queueName)
	if err := q.Connect(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer q.Close()

	msgs, err := q.Consume()
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	log.Printf("Waiting for parsed documents on queue '%s'...", queueName)

	// --- Consumer loop ---
	for msg := range msgs {
		var task ParsedDocumentTask
		if err := json.Unmarshal(msg.Body, &task); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			msg.Nack(false, false)
			continue
		}

		log.Printf("Received: %s (topic=%s)", task.CanonicalID, task.Topic)

		// Read parsed JSON from MinIO
		obj, err := minioClient.GetObject(context.Background(), task.Bucket, task.ParsedKey, minio.GetObjectOptions{})
		if err != nil {
			log.Printf("Failed to read from MinIO '%s': %v", task.ParsedKey, err)
			msg.Nack(false, true)
			continue
		}

		data, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			log.Printf("Failed to read object data '%s': %v", task.ParsedKey, err)
			msg.Nack(false, true)
			continue
		}

		// Parse the legal document
		var doc graph.LegalDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			log.Printf("Failed to parse JSON '%s': %v", task.ParsedKey, err)
			msg.Nack(false, false)
			continue
		}

		ctx := context.Background()

		// Upsert the document node with topic label
		if err := neo4jClient.UpsertDocument(ctx, &doc, task.Topic); err != nil {
			log.Printf("Failed to upsert document '%s': %v", doc.ID, err)
			msg.Nack(false, true)
			continue
		}

		// Create reference relationships
		if err := neo4jClient.CreateReferences(ctx, doc.ID, doc.Referencias); err != nil {
			log.Printf("Failed to create references for '%s': %v", doc.ID, err)
			// Non-fatal: document was already created
		}

		log.Printf("[OK] Populated: %s (topic=%s, refs=%d)", doc.ID, task.Topic, len(doc.Referencias))
		msg.Ack(false)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

// neo4jPasswordFromAuth extracts the password from NEO4J_AUTH format "user/password"
func neo4jPasswordFromAuth(auth string) string {
	parts := strings.SplitN(auth, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return auth
}

func init() {
	// If NEO4J_AUTH is set (docker-compose format), extract user/pass
	if auth := os.Getenv("NEO4J_AUTH"); auth != "" {
		parts := strings.SplitN(auth, "/", 2)
		if len(parts) == 2 {
			if os.Getenv("NEO4J_USER") == "" {
				os.Setenv("NEO4J_USER", parts[0])
			}
			if os.Getenv("NEO4J_PASS") == "" {
				os.Setenv("NEO4J_PASS", parts[1])
			}
		}
	}
}
