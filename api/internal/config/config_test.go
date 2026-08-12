package config

import (
	"testing"
	"time"
)

func TestValidateAllowsDevelopmentSecurityShortcuts(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Environment = "development"
	cfg.AllowPrivateStorageEndpoints = true
	cfg.ResetPasswordURL = "http://localhost:3000/reset-password"
	cfg.SMTPSecurityMode = smtpSecurityNone

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected development config to be valid, got %v", err)
	}
}

func TestValidateRejectsInsecureProductionResetURL(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Environment = "production"
	cfg.ResetPasswordURL = "http://app.example.com/reset-password"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config to reject http reset url")
	}
}

func TestValidateRejectsPrivateStorageOverrideOutsideDevelopment(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Environment = "production"
	cfg.AllowPrivateStorageEndpoints = true

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config to reject private storage override")
	}
}

func TestValidateRejectsInsecureSMTPOutsideDevelopment(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Environment = "production"
	cfg.SMTPSecurityMode = smtpSecurityNone

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config to reject insecure smtp mode")
	}
}

func TestValidateRequiresPairedSMTPCredentials(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.SMTPUser = "mailer"
	cfg.SMTPPass = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected config to require smtp user and password together")
	}
}

func TestValidateAcceptsGCPStorageWithEmulatorHost(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.Environment = "development"
	cfg.StorageProvider = "GCP_STORAGE"
	cfg.GCPStorageProjectID = "thingindex-dev"
	cfg.GCPStorageBucketName = "thingindex-docs"
	cfg.GCPStorageEmulatorHost = "127.0.0.1:4443"
	cfg.GCPStorageCredentialsJSON = ""

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected gcp storage config with emulator host to be valid, got %v", err)
	}
}

func TestValidateRejectsGCPStorageWithoutCredentialsOrEmulator(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	cfg.StorageProvider = "GCP_STORAGE"
	cfg.GCPStorageProjectID = "thingindex-dev"
	cfg.GCPStorageBucketName = "thingindex-docs"
	cfg.GCPStorageCredentialsJSON = ""
	cfg.GCPStorageEmulatorHost = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected gcp storage config without credentials or emulator host to be rejected")
	}
}

func validConfig() *Config {
	return &Config{
		Environment:                  "production",
		Port:                         "8000",
		TrustedProxies:               "",
		DBHost:                       "localhost",
		DBPort:                       "5432",
		DBUser:                       "postgres",
		DBPassword:                   "postgres",
		DBName:                       "thingindex",
		DBSSLMode:                    "require",
		CORSAllowedOrigins:           "https://app.example.com",
		JWTSecret:                    "super-secret-jwt",
		OpenAIKey:                    "",
		AllowPrivateStorageEndpoints: false,
		AuthRateLimitRequests:        10,
		AuthRateLimitWindow:          time.Minute,
		SMTPHost:                     "smtp.example.com",
		SMTPPort:                     "587",
		SMTPSecurityMode:             smtpSecurityStartTLS,
		SMTPUser:                     "",
		SMTPPass:                     "",
		SMTPFrom:                     "no-reply@example.com",
		ResetPasswordURL:             "https://app.example.com/reset-password",
		StorageProvider:              "MINIO",
		MinioEndpoint:                "storage.example.com:9000",
		MinioAccessKey:               "access-key",
		MinioSecretKey:               "secret-key",
		MinioUseSSL:                  "true",
		MinioBucketName:              "thingindex-docs",
		RabbitMQURL:                  "amqp://guest:guest@localhost:5672/",
		DocumentIndexExchange:        "thingindex.document.index",
		DocumentIndexQueue:           "thingindex.document.index.queue",
		DocumentIndexRoutingKey:      "document.index.requested",
		DocumentIndexRetryQueue:      "thingindex.document.index.retry",
		DocumentIndexDLQ:             "thingindex.document.index.dlq",
		OutboxPollInterval:           5 * time.Second,
		OutboxBatchSize:              20,
		ConsumerPrefetchCount:        10,
		DocumentIndexRetryDelay:      30 * time.Second,
		DocumentIndexMaxRetries:      5,
		PricingSyncInterval:          24 * time.Hour,
		MaxUploadSizeBytes:           20 * 1024 * 1024,
		MasterEncryptionKey:          "master-key",
		ServerReadTimeout:            15 * time.Second,
		ServerWriteTimeout:           30 * time.Second,
		ServerIdleTimeout:            60 * time.Second,
	}
}
