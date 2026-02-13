package local

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileSystemOutput writes parsed JSON documents to a local directory.
// It implements domain.OutputProvider.
type FileSystemOutput struct {
	baseDir string
}

// NewFileSystemOutput creates a new FileSystemOutput rooted at the given directory.
func NewFileSystemOutput(baseDir string) *FileSystemOutput {
	return &FileSystemOutput{baseDir: baseDir}
}

// Save writes the given data to a JSON file named after the canonical ID.
func (fo *FileSystemOutput) Save(id string, data []byte) error {
	if err := os.MkdirAll(fo.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	path := filepath.Join(fo.baseDir, id+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}
