package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectError bool
		expectedMsg string
	}{
		{
			name:        "Success with LocalStorage",
			cfg:         Config{Type: "local", LocalDir: "./tmp"},
			expectError: false,
		},
		{
			name:        "Error with S3 (not implemented)",
			cfg:         Config{Type: "s3"},
			expectError: true,
			expectedMsg: "s3 storage not implemented",
		},
		{
			name:        "Error with unknown type",
			cfg:         Config{Type: "azure"},
			expectError: true,
			expectedMsg: "unknown storage type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewStorage(tt.cfg)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, but got nil")
				}
				if !strings.Contains(err.Error(), tt.expectedMsg) {
					t.Errorf("Wrong error message. Expected to contain: %s, Got: %s", tt.expectedMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if s == nil {
					t.Error("Storage should not be nil on success")
				}
			}
		})
	}
}

func TestLocalStorage_Workflow(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalStorage(tmpDir)

	filename := "documents/law-123.txt"
	contentBody := "Official law content"

	t.Run("Should save a file and create subdirectories", func(t *testing.T) {
		reader := strings.NewReader(contentBody)
		path, err := storage.Save(filename, reader)

		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		expectedPath := filepath.Join(tmpDir, filename)
		if path != expectedPath {
			t.Errorf("Wrong returned path. Expected: %s, Got: %s", expectedPath, path)
		}
	})

	t.Run("Should check if file exists", func(t *testing.T) {
		if !storage.Exists(filename) {
			t.Error("File should exist, but Exists returned false")
		}

		if storage.Exists("ghost-file.txt") {
			t.Error("Non-existent file returned true")
		}
	})

	t.Run("Should read content correctly", func(t *testing.T) {
		readCloser, err := storage.Read(filename)
		if err != nil {
			t.Fatalf("Error reading file: %v", err)
		}
		defer readCloser.Close()

		data, err := io.ReadAll(readCloser)
		if err != nil {
			t.Fatalf("Error reading bytes: %v", err)
		}

		if string(data) != contentBody {
			t.Errorf("Content mismatch. Expected: %s, Got: %s", contentBody, string(data))
		}
	})

	t.Run("Should return error when reading non-existent file", func(t *testing.T) {
		_, err := storage.Read("does-not-exist.txt")
		if err == nil {
			t.Error("Should have returned error for non-existent file")
		}
		if !os.IsNotExist(err) {
			t.Errorf("Error should be of type 'IsNotExist', got: %v", err)
		}
	})
}