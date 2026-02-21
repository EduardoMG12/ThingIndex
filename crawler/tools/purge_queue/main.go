package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/user/legisgraph-crawler/internal/queue"
)

func main() {
	log.Println("Starting queue purge tool...")

	// Assuming running from project root
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found in current directory, looking in parent...")
		// Fallback for running from subdirectory
		if err := godotenv.Load("../../.env"); err != nil {
			log.Println("No .env file found, using defaults")
		}
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	queueName := os.Getenv("RABBITMQ_QUEUE")
	if queueName == "" {
		queueName = "crawler_tasks"
	}

	q, err := queue.NewRabbitMQClient(rabbitURL, queueName)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}

	if err := q.Connect(); err != nil {
		log.Fatalf("Failed to connect to queue: %v", err)
	}
	defer q.Close()

	log.Printf("Purging queue: %s", queueName)
	if err := q.Purge(); err != nil {
		log.Fatalf("Failed to purge queue: %v", err)
	}
	log.Println("Queue purged successfully.")
}
