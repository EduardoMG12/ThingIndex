package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment        string
	Port               string
	TrustedProxies     string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	CORSAllowedOrigins string

	JWTSecret string

	OpenAIKey string

	AllowPrivateStorageEndpoints bool
	AuthRateLimitRequests        int
	AuthRateLimitWindow          time.Duration

	SMTPHost         string
	SMTPPort         string
	SMTPSecurityMode string
	SMTPUser         string
	SMTPPass         string
	SMTPFrom         string
	ResetPasswordURL string

	// Storage config
	StorageProvider           string
	MinioEndpoint             string
	MinioAccessKey            string
	MinioSecretKey            string
	MinioUseSSL               string
	MinioBucketName           string
	GCPStorageProjectID       string
	GCPStorageBucketName      string
	GCPStorageLocation        string
	GCPStorageCredentialsJSON string
	GCPStorageEmulatorHost    string
	RabbitMQURL               string
	DocumentIndexExchange     string
	DocumentIndexQueue        string
	DocumentIndexRoutingKey   string
	DocumentIndexRetryQueue   string
	DocumentIndexDLQ          string
	OutboxPollInterval        time.Duration
	OutboxBatchSize           int
	ConsumerPrefetchCount     int
	DocumentIndexRetryDelay   time.Duration
	DocumentIndexMaxRetries   int
	PricingSyncInterval       time.Duration
	ModelCatalogSyncInterval  time.Duration
	MaxUploadSizeBytes        int64
	MasterEncryptionKey       string
	ServerReadTimeout         time.Duration
	ServerWriteTimeout        time.Duration
	ServerIdleTimeout         time.Duration
}

var AppConfig *Config

const (
	developmentEnv             = "development"
	defaultDevelopmentJWT      = "thingindex-local-jwt-secret-not-for-production"
	defaultDevelopmentMaster   = "thingindex-local-master-key-not-for-production"
	defaultDevelopmentMinioKey = "thingindex-local-access-key"
	defaultDevelopmentMinioSec = "thingindex-local-secret-key"
	multipartFormSlackBytes    = int64(1 << 20)
	smtpSecurityNone           = "none"
	smtpSecurityStartTLS       = "starttls"
	smtpSecurityTLS            = "tls"
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
		Environment:                  environment,
		Port:                         getEnv("PORT", "8000"),
		TrustedProxies:               getEnv("TRUSTED_PROXIES", ""),
		DBHost:                       getEnv("DB_HOST", "localhost"),
		DBPort:                       getEnv("DB_PORT", "5432"),
		DBUser:                       getEnv("DB_USER", "postgres"),
		DBPassword:                   getEnv("DB_PASSWORD", "postgres"),
		DBName:                       getEnv("DB_NAME", "thingindex"),
		DBSSLMode:                    getEnv("DB_SSLMODE", defaultDBSSLMode),
		CORSAllowedOrigins:           getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		OpenAIKey:                    getEnv("OPENAI_API_KEY", ""),
		AllowPrivateStorageEndpoints: getEnvBool("ALLOW_PRIVATE_STORAGE_ENDPOINTS", false),
		JWTSecret:                    getEnv("JWT_SECRET", defaultValueForEnvironment(environment, defaultDevelopmentJWT)),
		AuthRateLimitRequests:        getEnvInt("AUTH_RATE_LIMIT_REQUESTS", 10),
		AuthRateLimitWindow:          getEnvDurationSeconds("AUTH_RATE_LIMIT_WINDOW_SECONDS", 60),
		SMTPHost:                     getEnv("SMTP_HOST", "localhost"),
		SMTPPort:                     getEnv("SMTP_PORT", "1025"),
		SMTPSecurityMode:             getEnv("SMTP_SECURITY_MODE", defaultSMTPSecurityMode(environment)),
		SMTPUser:                     getEnv("SMTP_USER", ""),
		SMTPPass:                     getEnv("SMTP_PASS", ""),
		SMTPFrom:                     getEnv("SMTP_FROM", "no-reply@thingindex.com"),
		ResetPasswordURL:             getEnv("RESET_PASSWORD_URL", "http://localhost:3000/reset-password"),
		StorageProvider:              getEnv("STORAGE_PROVIDER", "MINIO"),
		MinioEndpoint:                getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:               getEnv("MINIO_ACCESS_KEY", defaultValueForEnvironment(environment, defaultDevelopmentMinioKey)),
		MinioSecretKey:               getEnv("MINIO_SECRET_KEY", defaultValueForEnvironment(environment, defaultDevelopmentMinioSec)),
		MinioUseSSL:                  getEnv("MINIO_USE_SSL", "false"),
		MinioBucketName:              getEnv("MINIO_BUCKET_NAME", "thingindex-docs"),
		GCPStorageProjectID:          getEnv("GCP_STORAGE_PROJECT_ID", ""),
		GCPStorageBucketName:         getEnv("GCP_STORAGE_BUCKET_NAME", "thingindex-docs"),
		GCPStorageLocation:           getEnv("GCP_STORAGE_LOCATION", ""),
		GCPStorageCredentialsJSON:    getEnv("GCP_STORAGE_CREDENTIALS_JSON", ""),
		GCPStorageEmulatorHost:       getEnv("GCP_STORAGE_EMULATOR_HOST", ""),
		RabbitMQURL:                  getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		DocumentIndexExchange:        getEnv("DOCUMENT_INDEX_EXCHANGE", "thingindex.document.index"),
		DocumentIndexQueue:           getEnv("DOCUMENT_INDEX_QUEUE", "thingindex.document.index.queue"),
		DocumentIndexRoutingKey:      getEnv("DOCUMENT_INDEX_ROUTING_KEY", "document.index.requested"),
		DocumentIndexRetryQueue:      getEnv("DOCUMENT_INDEX_RETRY_QUEUE", "thingindex.document.index.retry"),
		DocumentIndexDLQ:             getEnv("DOCUMENT_INDEX_DLQ", "thingindex.document.index.dlq"),
		OutboxPollInterval:           getEnvDurationSeconds("OUTBOX_POLL_INTERVAL_SECONDS", 5),
		OutboxBatchSize:              getEnvInt("OUTBOX_BATCH_SIZE", 20),
		ConsumerPrefetchCount:        getEnvInt("CONSUMER_PREFETCH_COUNT", 10),
		DocumentIndexRetryDelay:      getEnvDurationSeconds("DOCUMENT_INDEX_RETRY_DELAY_SECONDS", 30),
		DocumentIndexMaxRetries:      getEnvInt("DOCUMENT_INDEX_MAX_RETRIES", 5),
		PricingSyncInterval:          getEnvDurationSeconds("PRICING_SYNC_INTERVAL_SECONDS", 86400),
		ModelCatalogSyncInterval:     getEnvDurationSeconds("MODEL_CATALOG_SYNC_INTERVAL_SECONDS", 86400),
		MaxUploadSizeBytes:           getEnvInt64("MAX_UPLOAD_SIZE_BYTES", 20*1024*1024),
		MasterEncryptionKey:          getEnv("MASTER_ENCRYPTION_KEY", defaultValueForEnvironment(environment, defaultDevelopmentMaster)),
		ServerReadTimeout:            getEnvDurationSeconds("SERVER_READ_TIMEOUT_SECONDS", 15),
		ServerWriteTimeout:           getEnvDurationSeconds("SERVER_WRITE_TIMEOUT_SECONDS", 30),
		ServerIdleTimeout:            getEnvDurationSeconds("SERVER_IDLE_TIMEOUT_SECONDS", 60),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDurationSeconds(key string, fallbackSeconds int) time.Duration {
	return time.Duration(getEnvInt(key, fallbackSeconds)) * time.Second
}

func getEnvBool(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
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
	if strings.TrimSpace(c.DBSSLMode) == "" {
		missing = append(missing, "DB_SSLMODE")
	}
	if strings.TrimSpace(c.ResetPasswordURL) == "" {
		missing = append(missing, "RESET_PASSWORD_URL")
	}
	if strings.TrimSpace(c.SMTPSecurityMode) == "" {
		missing = append(missing, "SMTP_SECURITY_MODE")
	}
	if len(c.CorsAllowedOrigins()) == 0 {
		missing = append(missing, "CORS_ALLOWED_ORIGINS")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	if containsWildcardOrigin(c.CorsAllowedOrigins()) {
		return errors.New("CORS_ALLOWED_ORIGINS cannot contain '*' when credentials are enabled")
	}
	if c.AuthRateLimitRequests <= 0 {
		return errors.New("AUTH_RATE_LIMIT_REQUESTS must be greater than zero")
	}
	if c.AuthRateLimitWindow <= 0 {
		return errors.New("AUTH_RATE_LIMIT_WINDOW_SECONDS must be greater than zero")
	}
	if c.MaxUploadSizeBytes <= 0 {
		return errors.New("MAX_UPLOAD_SIZE_BYTES must be greater than zero")
	}
	if strings.TrimSpace(c.RabbitMQURL) == "" {
		return errors.New("RABBITMQ_URL is required")
	}
	if strings.TrimSpace(c.DocumentIndexExchange) == "" || strings.TrimSpace(c.DocumentIndexQueue) == "" || strings.TrimSpace(c.DocumentIndexRoutingKey) == "" {
		return errors.New("document indexing queue configuration is required")
	}
	if strings.TrimSpace(c.DocumentIndexRetryQueue) == "" || strings.TrimSpace(c.DocumentIndexDLQ) == "" {
		return errors.New("document indexing retry queue configuration is required")
	}
	if c.OutboxPollInterval <= 0 {
		return errors.New("OUTBOX_POLL_INTERVAL_SECONDS must be greater than zero")
	}
	if c.OutboxBatchSize <= 0 {
		return errors.New("OUTBOX_BATCH_SIZE must be greater than zero")
	}
	if c.ConsumerPrefetchCount <= 0 {
		return errors.New("CONSUMER_PREFETCH_COUNT must be greater than zero")
	}
	if c.DocumentIndexRetryDelay <= 0 {
		return errors.New("DOCUMENT_INDEX_RETRY_DELAY_SECONDS must be greater than zero")
	}
	if c.PricingSyncInterval <= 0 {
		return errors.New("PRICING_SYNC_INTERVAL_SECONDS must be greater than zero")
	}
	if c.DocumentIndexMaxRetries <= 0 {
		return errors.New("DOCUMENT_INDEX_MAX_RETRIES must be greater than zero")
	}
	if c.ServerReadTimeout <= 0 || c.ServerWriteTimeout <= 0 || c.ServerIdleTimeout <= 0 {
		return errors.New("server timeout values must be greater than zero")
	}
	if err := c.validateDefaultStorageConfiguration(); err != nil {
		return err
	}
	if err := c.validateResetPasswordURL(); err != nil {
		return err
	}
	if err := c.validateSMTPSecurityMode(); err != nil {
		return err
	}
	if (strings.TrimSpace(c.SMTPUser) == "") != (strings.TrimSpace(c.SMTPPass) == "") {
		return errors.New("SMTP_USER and SMTP_PASS must be provided together")
	}

	if c.IsDevelopment() {
		return nil
	}

	var insecure []string
	if c.AllowPrivateStorageEndpoints {
		insecure = append(insecure, "ALLOW_PRIVATE_STORAGE_ENDPOINTS")
	}
	if c.JWTSecret == defaultDevelopmentJWT {
		insecure = append(insecure, "JWT_SECRET")
	}
	if c.MasterEncryptionKey == defaultDevelopmentMaster {
		insecure = append(insecure, "MASTER_ENCRYPTION_KEY")
	}
	if strings.EqualFold(c.DefaultStorageProviderType(), "MINIO") {
		if c.MinioAccessKey == defaultDevelopmentMinioKey {
			insecure = append(insecure, "MINIO_ACCESS_KEY")
		}
		if c.MinioSecretKey == defaultDevelopmentMinioSec {
			insecure = append(insecure, "MINIO_SECRET_KEY")
		}
	}
	if strings.EqualFold(c.DBSSLMode, "disable") {
		insecure = append(insecure, "DB_SSLMODE")
	}
	if c.NormalizedSMTPSecurityMode() == smtpSecurityNone {
		insecure = append(insecure, "SMTP_SECURITY_MODE")
	}
	if resetURL, err := c.ResetPasswordURLParsed(); err != nil || !strings.EqualFold(resetURL.Scheme, "https") {
		insecure = append(insecure, "RESET_PASSWORD_URL")
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

func (c *Config) CorsAllowedOrigins() []string {
	if c == nil {
		return nil
	}

	return parseCSV(c.CORSAllowedOrigins)
}

func (c *Config) TrustedProxyList() []string {
	if c == nil {
		return nil
	}

	proxies := parseCSV(c.TrustedProxies)
	if len(proxies) == 0 {
		return nil
	}

	return proxies
}

func (c *Config) UploadRequestLimitBytes() int64 {
	if c == nil || c.MaxUploadSizeBytes <= 0 {
		return multipartFormSlackBytes
	}

	return c.MaxUploadSizeBytes + multipartFormSlackBytes
}

func (c *Config) NormalizedSMTPSecurityMode() string {
	if c == nil {
		return smtpSecurityNone
	}

	return strings.ToLower(strings.TrimSpace(c.SMTPSecurityMode))
}

func (c *Config) ResetPasswordURLParsed() (*url.URL, error) {
	if c == nil {
		return nil, errors.New("config is not loaded")
	}

	parsed, err := url.Parse(strings.TrimSpace(c.ResetPasswordURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("RESET_PASSWORD_URL must include scheme and host")
	}

	return parsed, nil
}

func defaultValueForEnvironment(environment, developmentValue string) string {
	if isDevelopmentEnvironment(environment) {
		return developmentValue
	}

	return ""
}

func defaultSMTPSecurityMode(environment string) string {
	if isDevelopmentEnvironment(environment) {
		return smtpSecurityNone
	}

	return smtpSecurityStartTLS
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

func containsWildcardOrigin(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return true
		}
	}

	return false
}

func (c *Config) validateResetPasswordURL() error {
	if _, err := c.ResetPasswordURLParsed(); err != nil {
		return fmt.Errorf("invalid RESET_PASSWORD_URL: %w", err)
	}

	return nil
}

func (c *Config) validateSMTPSecurityMode() error {
	switch c.NormalizedSMTPSecurityMode() {
	case smtpSecurityNone, smtpSecurityStartTLS, smtpSecurityTLS:
		return nil
	default:
		return errors.New("SMTP_SECURITY_MODE must be one of: none, starttls, tls")
	}
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}
