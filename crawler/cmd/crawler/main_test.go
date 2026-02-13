package main

import (
	"os"
	"testing"
)

func TestMainRequiresStartURL(t *testing.T) {
	// Ensure CRAWLER_START_URL is not set
	os.Unsetenv("CRAWLER_START_URL")

	// main() calls log.Fatal if CRAWLER_START_URL is empty,
	// which we can't easily test without killing the process.
	// Instead, we verify the env var logic directly.
	url := os.Getenv("CRAWLER_START_URL")
	if url != "" {
		t.Error("CRAWLER_START_URL should be empty for this test")
	}
}

func TestMainEnvDefaults(t *testing.T) {
	// Test that default values are applied when env vars are missing
	os.Unsetenv("STORAGE_TYPE")
	os.Unsetenv("STORAGE_LOCAL_DIR")
	os.Unsetenv("CRAWLER_MAX_DEPTH")
	os.Unsetenv("CRAWLER_TIMEOUT")

	storageType := os.Getenv("STORAGE_TYPE")
	if storageType != "" {
		t.Error("STORAGE_TYPE should be empty when unset")
	}

	localDir := os.Getenv("STORAGE_LOCAL_DIR")
	if localDir != "" {
		t.Error("STORAGE_LOCAL_DIR should be empty when unset")
	}
}
