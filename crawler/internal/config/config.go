package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Timeout        time.Duration
	MaxDepth       int
	StorageType    string
	LocalDir       string
	IgnoreKeywords []string
	FilterName     string
	RabbitURL      string
	QueueName      string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
}

func LoadConfig() AppConfig {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults or system env vars")
	}

	timeoutStr := os.Getenv("CRAWLER_TIMEOUT")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 10 * time.Second
	}

	maxDepthStr := os.Getenv("CRAWLER_MAX_DEPTH")
	maxDepth, err := strconv.Atoi(maxDepthStr)
	if err != nil {
		maxDepth = 20
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

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	queueName := os.Getenv("RABBITMQ_QUEUE")
	if queueName == "" {
		queueName = "crawler_tasks"
	}

	return AppConfig{
		Timeout:        timeout,
		MaxDepth:       maxDepth,
		StorageType:    storageType,
		LocalDir:       localDir,
		IgnoreKeywords: ignoreKeywords,
		FilterName:     filterName,
		RabbitURL:      rabbitURL,
		QueueName:      queueName,
		MinioEndpoint:  os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey: os.Getenv("MINIO_SECRET_KEY"),
		MinioUseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
	}
}
