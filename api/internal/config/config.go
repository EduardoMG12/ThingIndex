package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	OpenAIKey   string
	JWTSecret   string
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string

	// Storage config
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    string
	MinioBucketName string
}

var AppConfig *Config

func Load() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found. Using system environment variables.")
	}

	AppConfig = &Config{
		Port:       getEnv("PORT", "8000"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "thingindex"),
		OpenAIKey:  getEnv("OPENAI_API_KEY", ""),
		JWTSecret:  getEnv("JWT_SECRET", "supersecret"),
		SMTPHost:   getEnv("SMTP_HOST", "localhost"),
		SMTPPort:   getEnv("SMTP_PORT", "1025"),
		SMTPUser:   getEnv("SMTP_USER", ""),
		SMTPPass:   getEnv("SMTP_PASS", ""),
		SMTPFrom:   getEnv("SMTP_FROM", "no-reply@thingindex.com"),
		MinioEndpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:     getEnv("MINIO_USE_SSL", "false"),
		MinioBucketName: getEnv("MINIO_BUCKET_NAME", "thingindex-docs"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
