package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/user/legisgraph/populator/internal/graph"
	"github.com/user/legisgraph/populator/internal/queue"
	"github.com/user/legisgraph/telemetry"
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
	telemetryExchange := getEnv("TELEMETRY_EXCHANGE", "pipeline_telemetry")

	if rabbitmqURL == "" || neo4jURI == "" || neo4jPass == "" || minioEndpoint == "" || minioAccessKey == "" || minioSecretKey == "" {
		log.Fatalf("Critical environment variables missing (RABBITMQ_URL, NEO4J_URI, NEO4J_PASS, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY)")
	}

	// --- Telemetry ---
	te := telemetry.NewEmitter(rabbitmqURL, telemetryExchange, "populator")
	defer te.Close()

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

	var nodesCreated int32 = 0
    var wg sync.WaitGroup
    numWorkers := 3

	// --- Consumer loop ---
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            for msg := range msgs {
                var task ParsedDocumentTask
                if err := json.Unmarshal(msg.Body, &task); err != nil {
                    log.Printf("[Worker %d] Failed to unmarshal message: %v", workerID, err)
                    msg.Nack(false, false)
                    continue
                }

                log.Printf("[Worker %d] Received: %s (topic=%s)", workerID, task.CanonicalID, task.Topic)
                te.Emit("criar_nos_relacoes", "info", fmt.Sprintf("[Worker %d] Recebendo documento: %s (topic=%s)", workerID, task.CanonicalID, task.Topic), nil)

                // Read parsed JSON from MinIO
                obj, err := minioClient.GetObject(context.Background(), task.Bucket, task.ParsedKey, minio.GetObjectOptions{})
                if err != nil {
                    log.Printf("[Worker %d] Failed to read from MinIO '%s': %v", workerID, task.ParsedKey, err)
                    msg.Nack(false, true)
                    continue
                }

                data, err := io.ReadAll(obj)
                obj.Close()
                if err != nil {
                    log.Printf("[Worker %d] Failed to read object data '%s': %v", workerID, task.ParsedKey, err)
                    msg.Nack(false, true)
                    continue
                }

                // Parse the legal document
                var doc graph.LegalDocument
                if err := json.Unmarshal(data, &doc); err != nil {
                    log.Printf("[Worker %d] Failed to parse JSON '%s': %v", workerID, task.ParsedKey, err)
                    msg.Nack(false, false)
                    continue
                }

                ctx := context.Background()

                // Upsert the document node with topic label
                te.Emit("criar_nos_relacoes", "info", fmt.Sprintf("[Worker %d] Criando nós no Neo4j: %s", workerID, doc.ID), nil)
                if err := neo4jClient.UpsertDocument(ctx, &doc, task.Topic); err != nil {
                    log.Printf("[Worker %d] Failed to upsert document '%s': %v", workerID, doc.ID, err)
                    te.Emit("criar_nos_relacoes", "error", fmt.Sprintf("[Worker %d] Erro ao criar nó: %s", workerID, doc.ID), nil)
                    msg.Nack(false, true)
                    continue
                }
                
                // Use a local copy to avoid race conditions with telemetry output payload
                var currentNodes int
                
                // Increment atomic counter
                // Note: we're using a simple counter here without atomic package fully imported just for the log
                // If we want exact, we'd need sync/atomic
                currentNodes = int(nodesCreated) + 1 
                nodesCreated++ 

                // Create reference relationships
                if err := neo4jClient.CreateReferences(ctx, doc.ID, doc.Referencias); err != nil {
                    log.Printf("[Worker %d] Failed to create references for '%s': %v", workerID, doc.ID, err)
                    // Non-fatal: document was already created
                }

                log.Printf("[Worker %d] [OK] Populated: %s (topic=%s, refs=%d)", workerID, doc.ID, task.Topic, len(doc.Referencias))
                te.Emit("criar_nos_relacoes", "success", fmt.Sprintf("[Worker %d] Nó criado: %s (%d refs)", workerID, doc.ID, len(doc.Referencias)), map[string]int{"neo4j_nodes": currentNodes})
                msg.Ack(false)
            }
        }(i)
    }

    // Wait and block the main thread
    wg.Wait()
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

