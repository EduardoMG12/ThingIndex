package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNearestDotEnvLoadsParentFallbacks(t *testing.T) {
	rootDir := t.TempDir()
	nestedDir := filepath.Join(rootDir, "api", "internal")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(rootDir, ".env"), []byte("TESTUTIL_ROOT_ONLY=from-root\n"), 0o644); err != nil {
		t.Fatalf("write root env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "api", ".env"), []byte("TESTUTIL_CHILD_ONLY=from-child\n"), 0o644); err != nil {
		t.Fatalf("write child env: %v", err)
	}

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(nestedDir); err != nil {
		t.Fatalf("chdir nested dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	unsetEnvKeys(t, "TESTUTIL_ROOT_ONLY", "TESTUTIL_CHILD_ONLY")

	loadNearestDotEnv()

	if got := os.Getenv("TESTUTIL_CHILD_ONLY"); got != "from-child" {
		t.Fatalf("expected child env to load, got %q", got)
	}
	if got := os.Getenv("TESTUTIL_ROOT_ONLY"); got != "from-root" {
		t.Fatalf("expected root env fallback to load, got %q", got)
	}
}

func TestLoadNearestDotEnvKeepsNearestOverride(t *testing.T) {
	rootDir := t.TempDir()
	nestedDir := filepath.Join(rootDir, "api", "internal")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(rootDir, ".env"), []byte("TESTUTIL_SHARED=from-root\n"), 0o644); err != nil {
		t.Fatalf("write root env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "api", ".env"), []byte("TESTUTIL_SHARED=from-child\n"), 0o644); err != nil {
		t.Fatalf("write child env: %v", err)
	}

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(nestedDir); err != nil {
		t.Fatalf("chdir nested dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	unsetEnvKeys(t, "TESTUTIL_SHARED")

	loadNearestDotEnv()

	if got := os.Getenv("TESTUTIL_SHARED"); got != "from-child" {
		t.Fatalf("expected nearest env to win, got %q", got)
	}
}

func unsetEnvKeys(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		originalValue, hadValue := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if !hadValue {
				_ = os.Unsetenv(key)
				return
			}
			_ = os.Setenv(key, originalValue)
		})
	}
}
