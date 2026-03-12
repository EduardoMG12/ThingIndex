package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string

	JWTSecret string

	OpenAIKey string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Storage config
	MinioEndpoint       string
	MinioAccessKey      string
	MinioSecretKey      string
	MinioUseSSL         string
	MinioBucketName     string
	MasterEncryptionKey string
}

var AppConfig *Config

const (
	developmentEnv             = "development"
	defaultDevelopmentJWT      = "thingindex-local-jwt-secret-not-for-production"
	defaultDevelopmentMaster   = "thingindex-local-master-key-not-for-production"
	defaultDevelopmentMinioKey = "thingindex-local-access-key"
	defaultDevelopmentMinioSec = "thingindex-local-secret-key"
)

func Load() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found. Using system environment variables.")
	}

	environment := normalizeEnvironment(getEnv("APP_ENV", developmentEnv))
	defaultDBSSLMode := "require"
	if isDevelopmentEnvironment(environment) {
		defaultDBSSLMode = "disable"
	}

	AppConfig = &Config{
		Environment:         environment,
		Port:                getEnv("PORT", "8000"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "postgres"),
		DBName:              getEnv("DB_NAME", "thingindex"),
		DBSSLMode:           getEnv("DB_SSLMODE", defaultDBSSLMode),
		OpenAIKey:           getEnv("OPENAI_API_KEY", ""),
		JWTSecret:           getEnv("JWT_SECRET", defaultValueForEnvironment(environment, defaultDevelopmentJWT)),
		SMTPHost:            getEnv("SMTP_HOST", "localhost"),
		SMTPPort:            getEnv("SMTP_PORT", "1025"),
		SMTPUser:            getEnv("SMTP_USER", ""),
		SMTPPass:            getEnv("SMTP_PASS", ""),
		SMTPFrom:            getEnv("SMTP_FROM", "no-reply@thingindex.com"),
		MinioEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:      getEnv("MINIO_ACCESS_KEY", defaultValueForEnvironment(environment, defaultDevelopmentMinioKey)),
		MinioSecretKey:      getEnv("MINIO_SECRET_KEY", defaultValueForEnvironment(environment, defaultDevelopmentMinioSec)),
		MinioUseSSL:         getEnv("MINIO_USE_SSL", "false"),
		MinioBucketName:     getEnv("MINIO_BUCKET_NAME", "thingindex-docs"),
		MasterEncryptionKey: getEnv("MASTER_ENCRYPTION_KEY", defaultValueForEnvironment(environment, defaultDevelopmentMaster)),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is not loaded")
	}

	var missing []string
	if strings.TrimSpace(c.JWTSecret) == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if strings.TrimSpace(c.MasterEncryptionKey) == "" {
		missing = append(missing, "MASTER_ENCRYPTION_KEY")
	}
	if strings.TrimSpace(c.MinioAccessKey) == "" {
		missing = append(missing, "MINIO_ACCESS_KEY")
	}
	if strings.TrimSpace(c.MinioSecretKey) == "" {
		missing = append(missing, "MINIO_SECRET_KEY")
	}
	if strings.TrimSpace(c.DBSSLMode) == "" {
		missing = append(missing, "DB_SSLMODE")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	if c.IsDevelopment() {
		return nil
	}

	var insecure []string
	if c.JWTSecret == defaultDevelopmentJWT {
		insecure = append(insecure, "JWT_SECRET")
	}
	if c.MasterEncryptionKey == defaultDevelopmentMaster {
		insecure = append(insecure, "MASTER_ENCRYPTION_KEY")
	}
	if c.MinioAccessKey == defaultDevelopmentMinioKey {
		insecure = append(insecure, "MINIO_ACCESS_KEY")
	}
	if c.MinioSecretKey == defaultDevelopmentMinioSec {
		insecure = append(insecure, "MINIO_SECRET_KEY")
	}
	if strings.EqualFold(c.DBSSLMode, "disable") {
		insecure = append(insecure, "DB_SSLMODE")
	}

	if len(insecure) > 0 {
		return fmt.Errorf("insecure production configuration: %s", strings.Join(insecure, ", "))
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	if c == nil {
		return false
	}

	return isDevelopmentEnvironment(c.Environment)
}

func defaultValueForEnvironment(environment, developmentValue string) string {
	if isDevelopmentEnvironment(environment) {
		return developmentValue
	}

	return ""
}

func normalizeEnvironment(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return developmentEnv
	}

	return normalized
}

func isDevelopmentEnvironment(environment string) bool {
	switch normalizeEnvironment(environment) {
	case "local", "development", "dev", "test":
		return true
	default:
		return false
	}
}
