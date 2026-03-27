//go:build integration

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
)

var loadStorageIntegrationEnvOnce sync.Once

func TestMinioStorageRoundTripUploadReadExistsDelete(t *testing.T) {
	loadStorageIntegrationEnvOnce.Do(func() {
		_ = godotenv.Load("../../../../.env")
	})

	cfg := minioIntegrationConfigFromEnv(t)
	storage, err := NewMinioStorage(cfg.endpoint, cfg.accessKey, cfg.secretKey, cfg.bucketName, cfg.useSSL)
	if err != nil {
		t.Fatalf("NewMinioStorage returned error: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for object := range storage.client.ListObjects(cleanupCtx, storage.bucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				t.Fatalf("ListObjects returned error: %v", object.Err)
			}
			if err := storage.client.RemoveObject(cleanupCtx, storage.bucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Fatalf("RemoveObject returned error: %v", err)
			}
		}
		if err := storage.client.RemoveBucket(cleanupCtx, storage.bucketName); err != nil {
			t.Fatalf("RemoveBucket returned error: %v", err)
		}
	})

	ctx := context.Background()
	objectName := fmt.Sprintf("integration/%d/document.txt", time.Now().UnixNano())
	payload := []byte("thingindex integration payload")

	fileURL, err := storage.UploadFile(ctx, bytes.NewReader(payload), int64(len(payload)), "text/plain", objectName)
	if err != nil {
		t.Fatalf("UploadFile returned error: %v", err)
	}
	if fileURL != cfg.bucketName+"/"+objectName {
		t.Fatalf("expected file URL %q, got %q", cfg.bucketName+"/"+objectName, fileURL)
	}

	exists, err := storage.FileExists(ctx, fileURL)
	if err != nil {
		t.Fatalf("FileExists returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected uploaded file to exist")
	}

	reader, err := storage.OpenFile(ctx, fileURL)
	if err != nil {
		t.Fatalf("OpenFile returned error: %v", err)
	}
	defer reader.Close()

	readBack, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if !bytes.Equal(readBack, payload) {
		t.Fatalf("expected read payload %q, got %q", string(payload), string(readBack))
	}

	if err := storage.DeleteFile(ctx, fileURL); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}

	exists, err = storage.FileExists(ctx, fileURL)
	if err != nil {
		t.Fatalf("FileExists after delete returned error: %v", err)
	}
	if exists {
		t.Fatalf("expected deleted file to be absent")
	}
}

type minioIntegrationConfig struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	useSSL     bool
}

func minioIntegrationConfigFromEnv(t *testing.T) minioIntegrationConfig {
	t.Helper()

	endpoint := firstStorageNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_ENDPOINT"), os.Getenv("MINIO_ENDPOINT"))
	endpoint = normalizeMinioEndpoint(endpoint)
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}

	accessKey := firstStorageNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_ACCESS_KEY"), os.Getenv("MINIO_ACCESS_KEY"), "admin")
	secretKey := firstStorageNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_SECRET_KEY"), os.Getenv("MINIO_SECRET_KEY"), "admin1234")
	useSSL, err := strconv.ParseBool(firstStorageNonEmpty(os.Getenv("THINGINDEX_TEST_MINIO_USE_SSL"), os.Getenv("MINIO_USE_SSL"), "false"))
	if err != nil {
		t.Fatalf("parse minio ssl flag: %v", err)
	}

	return minioIntegrationConfig{
		endpoint:   endpoint,
		accessKey:  accessKey,
		secretKey:  secretKey,
		bucketName: fmt.Sprintf("thingindex-integration-%d", time.Now().UnixNano()),
		useSSL:     useSSL,
	}
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

func firstStorageNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
