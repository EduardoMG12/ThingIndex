package testutil

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	appdb "thing-index/db"
	storagepkg "thing-index/internal/pkg/storage"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type IntegrationPostgresConfig struct {
	host     string
	port     string
	user     string
	password string
	dbName   string
	sslMode  string
}

type IntegrationMinioConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
}

func OpenIsolatedPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()

	loadRootEnvOnce.Do(loadNearestDotEnv)

	cfg := integrationPostgresConfigFromEnv()
	if cfg.host == "" || cfg.port == "" || cfg.user == "" || cfg.dbName == "" {
		t.Skip("integration Postgres config is not available; set THINGINDEX_TEST_DB_* or provide the root .env used by docker compose")
	}

	adminDB, err := openIntegrationGorm(buildIntegrationDSN(cfg, ""))
	if err != nil {
		t.Fatalf("open integration admin database: %v", err)
	}

	schemaName := fmt.Sprintf("integration_%s_%d", sanitizeIntegrationIdentifier(t.Name()), time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName)).Error; err != nil {
		t.Fatalf("create integration schema: %v", err)
	}

	testDB, err := openIntegrationGorm(buildIntegrationDSN(cfg, schemaName))
	if err != nil {
		t.Fatalf("open integration schema database: %v", err)
	}

	if err := migrateIntegrationModels(testDB); err != nil {
		t.Fatalf("migrate integration schema: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := testDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}

		if err := adminDB.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName)).Error; err != nil {
			t.Fatalf("drop integration schema: %v", err)
		}

		adminSQLDB, err := adminDB.DB()
		if err == nil {
			_ = adminSQLDB.Close()
		}
	})

	return testDB
}

func NewIntegrationMinioStorage(t *testing.T) (*storagepkg.MinioStorage, IntegrationMinioConfig) {
	t.Helper()

	loadRootEnvOnce.Do(loadNearestDotEnv)

	cfg := integrationMinioConfigFromEnv(t)
	storage, err := storagepkg.NewMinioStorage(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.BucketName, cfg.UseSSL)
	if err != nil {
		t.Fatalf("create integration minio storage: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure:       cfg.UseSSL,
			BucketLookup: minio.BucketLookupPath,
		})
		if err != nil {
			t.Fatalf("cleanup minio client: %v", err)
		}

		for object := range client.ListObjects(cleanupCtx, cfg.BucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				t.Fatalf("cleanup list objects: %v", object.Err)
			}
			if err := client.RemoveObject(cleanupCtx, cfg.BucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Fatalf("cleanup remove object %s: %v", object.Key, err)
			}
		}

		if err := client.RemoveBucket(cleanupCtx, cfg.BucketName); err != nil {
			t.Fatalf("cleanup remove bucket %s: %v", cfg.BucketName, err)
		}
	})

	return storage, cfg
}

func integrationPostgresConfigFromEnv() IntegrationPostgresConfig {
	host := firstNonEmpty(os.Getenv("THINGINDEX_TEST_DB_HOST"), os.Getenv("DB_HOST"))
	port := firstNonEmpty(os.Getenv("THINGINDEX_TEST_DB_PORT"), os.Getenv("DB_PORT"))
	user := firstNonEmpty(os.Getenv("THINGINDEX_TEST_DB_USER"), os.Getenv("DB_USER"), os.Getenv("POSTGRES_USER"))
	password := firstNonEmpty(os.Getenv("THINGINDEX_TEST_DB_PASSWORD"), os.Getenv("DB_PASSWORD"), os.Getenv("POSTGRES_PASSWORD"))
	dbName := firstNonEmpty(os.Getenv("THINGINDEX_TEST_DB_NAME"), os.Getenv("DB_NAME"), os.Getenv("POSTGRES_DB"))
	sslMode := firstNonEmpty(os.Getenv("THINGINDEX_TEST_DB_SSLMODE"), os.Getenv("DB_SSLMODE"), "disable")

	if host == "" && user != "" && dbName != "" {
		host = "127.0.0.1"
	}
	if port == "" && user != "" && dbName != "" {
		port = "5434"
	}

	return IntegrationPostgresConfig{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbName:   dbName,
		sslMode:  sslMode,
	}
}

func integrationMinioConfigFromEnv(t *testing.T) IntegrationMinioConfig {
	t.Helper()

	endpoint := firstNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_ENDPOINT"), os.Getenv("MINIO_ENDPOINT"))
	endpoint = normalizeMinioEndpoint(endpoint)
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}

	accessKey := firstNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_ACCESS_KEY"), os.Getenv("MINIO_ACCESS_KEY"), "admin")
	secretKey := firstNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_SECRET_KEY"), os.Getenv("MINIO_SECRET_KEY"), "admin1234")
	useSSL, err := strconv.ParseBool(firstNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_USE_SSL"), os.Getenv("MINIO_USE_SSL"), "false"))
	if err != nil {
		t.Fatalf("parse minio ssl flag: %v", err)
	}

	return IntegrationMinioConfig{
		Endpoint:   endpoint,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		BucketName: fmt.Sprintf("thingindex-integration-%d", time.Now().UnixNano()),
		UseSSL:     useSSL,
	}
}

func buildIntegrationDSN(cfg IntegrationPostgresConfig, searchPath string) string {
	parts := []string{
		fmt.Sprintf("host=%s", cfg.host),
		fmt.Sprintf("port=%s", cfg.port),
		fmt.Sprintf("user=%s", cfg.user),
		fmt.Sprintf("password=%s", cfg.password),
		fmt.Sprintf("dbname=%s", cfg.dbName),
		fmt.Sprintf("sslmode=%s", cfg.sslMode),
	}

	if searchPath != "" {
		parts = append(parts, fmt.Sprintf("search_path=%s", searchPath))
	}

	return strings.Join(parts, " ")
}

func openIntegrationGorm(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
}

func migrateIntegrationModels(db *gorm.DB) error {
	return db.AutoMigrate(appdb.ManagedModels()...)
}

func sanitizeIntegrationIdentifier(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	return strings.Trim(b.String(), "_")
}

func normalizeMinioEndpoint(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}

	switch trimmed {
	case "minio", "minio:9000":
		return "127.0.0.1:9000"
	}

	if strings.HasPrefix(trimmed, "minio:") {
		return "127.0.0.1" + strings.TrimPrefix(trimmed, "minio")
	}

	return trimmed
}
