//go:build integration

package repository

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	appdb "thing-index/db"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var loadIntegrationEnvOnce sync.Once

type integrationPostgresConfig struct {
	host     string
	port     string
	user     string
	password string
	dbName   string
	sslMode  string
}

func openIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	loadIntegrationEnvOnce.Do(func() {
		_ = godotenv.Load("../../../.env")
	})

	cfg := integrationPostgresConfigFromEnv()
	if cfg.host == "" || cfg.port == "" || cfg.user == "" || cfg.dbName == "" {
		t.Skip("integration Postgres config is not available; set THINGINDEX_TEST_DB_* or provide the root .env used by docker compose")
	}

	adminDB, err := openIntegrationGorm(buildIntegrationDSN(cfg, ""))
	if err != nil {
		t.Fatalf("open integration admin database: %v", err)
	}

	schemaName := fmt.Sprintf("integration_%s_%d", sanitizeIdentifier(t.Name()), time.Now().UnixNano())
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

func integrationPostgresConfigFromEnv() integrationPostgresConfig {
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

	return integrationPostgresConfig{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbName:   dbName,
		sslMode:  sslMode,
	}
}

func buildIntegrationDSN(cfg integrationPostgresConfig, searchPath string) string {
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

// Esta lista precisa acompanhar o AutoMigrate de producao em db/connect.go.
// DocumentProcessingAttempt estava faltando, entao
// DeleteDocumentsByCollection falhava com `relation
// "document_processing_attempts" does not exist` — um teste vermelho que
// ninguem via, porque a suite de integracao nao roda sozinha.
func migrateIntegrationModels(db *gorm.DB) error {
	return db.AutoMigrate(appdb.ManagedModels()...)
}

func sanitizeIdentifier(value string) string {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
