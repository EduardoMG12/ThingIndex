//go:build integration

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	giterator "google.golang.org/api/iterator"
)

func TestGCSStorageRoundTripUploadReadExistsDelete(t *testing.T) {
	loadStorageIntegrationEnvOnce.Do(func() {
		_ = godotenv.Load("../../../../.env")
	})

	cfg := gcsIntegrationConfigFromEnv()
	storage, err := NewGCSStorage(cfg.projectID, cfg.bucketName, cfg.location, "", cfg.emulatorHost)
	if err != nil {
		if shouldSkipGCSIntegration(err) {
			t.Skipf("gcs integration emulator is not available: %v", err)
		}
		t.Fatalf("NewGCSStorage returned error: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		objects := storage.bucket.Objects(cleanupCtx, nil)
		for {
			attrs, err := objects.Next()
			if err == giterator.Done {
				break
			}
			if err != nil {
				t.Fatalf("Objects iterator returned error: %v", err)
			}
			if err := storage.bucket.Object(attrs.Name).Delete(cleanupCtx); err != nil {
				t.Fatalf("Delete returned error during cleanup: %v", err)
			}
		}
		if err := storage.bucket.Delete(cleanupCtx); err != nil {
			t.Fatalf("Bucket delete returned error: %v", err)
		}
		_ = storage.client.Close()
	})

	ctx := context.Background()
	objectName := fmt.Sprintf("integration/%d/document.txt", time.Now().UnixNano())
	payload := []byte("thingindex gcs integration payload")

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
		t.Fatal("expected uploaded file to exist")
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
		t.Fatal("expected deleted file to be absent")
	}
}

type gcsIntegrationConfig struct {
	projectID    string
	bucketName   string
	location     string
	emulatorHost string
}

func gcsIntegrationConfigFromEnv() gcsIntegrationConfig {
	emulatorHost := firstStorageNonEmpty(
		os.Getenv("THINGINDEX_TEST_GCP_EMULATOR_HOST"),
		os.Getenv("GCP_STORAGE_EMULATOR_HOST"),
		"127.0.0.1:4443",
	)

	return gcsIntegrationConfig{
		projectID:    firstStorageNonEmpty(os.Getenv("THINGINDEX_TEST_GCP_PROJECT_ID"), os.Getenv("GCP_STORAGE_PROJECT_ID"), "thingindex-dev"),
		bucketName:   fmt.Sprintf("thingindex-gcs-integration-%d", time.Now().UnixNano()),
		location:     firstStorageNonEmpty(os.Getenv("THINGINDEX_TEST_GCP_LOCATION"), os.Getenv("GCP_STORAGE_LOCATION"), "us-central1"),
		emulatorHost: emulatorHost,
	}
}

func shouldSkipGCSIntegration(err error) bool {
	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	patterns := []string{
		"connection refused",
		"no such host",
		"server misbehaving",
		"lookup ",
		"dial tcp",
		"deadline exceeded",
	}

	for _, pattern := range patterns {
		if strings.Contains(lowerErr, pattern) {
			return true
		}
	}

	return false
}
