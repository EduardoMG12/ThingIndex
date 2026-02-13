package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/legisgraph/legal-parser/internal/domain"
)

// FileSystemSource reads raw crawler output from a local directory.
// It implements domain.SourceProvider.
type FileSystemSource struct {
	baseDir string
}

// NewFileSystemSource creates a new FileSystemSource rooted at the given directory.
func NewFileSystemSource(baseDir string) *FileSystemSource {
	return &FileSystemSource{baseDir: baseDir}
}

// ListDocuments scans the base directory for paired .htm + .json files.
// Files are paired by their hash prefix (e.g., "292ac8fd96_name.htm" + "292ac8fd96_name.json").
func (fs *FileSystemSource) ListDocuments() ([]domain.RawDocument, error) {
	htmFiles := map[string]string{} // hash -> full path
	jsonFiles := map[string]string{}

	err := filepath.Walk(fs.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		ext := strings.ToLower(filepath.Ext(name))
		baseName := strings.TrimSuffix(name, filepath.Ext(name))

		switch ext {
		case ".htm", ".html":
			htmFiles[baseName] = path
		case ".json":
			jsonFiles[baseName] = path
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", fs.baseDir, err)
	}

	var docs []domain.RawDocument
	for baseName, htmPath := range htmFiles {
		jsonPath, hasJSON := jsonFiles[baseName]
		if !hasJSON {
			continue // Skip HTML files without matching metadata
		}

		docs = append(docs, domain.RawDocument{
			ID:           baseName,
			ContentFile:  htmPath,
			MetadataFile: jsonPath,
		})
	}

	return docs, nil
}

// GetContent reads the raw HTML bytes for a document.
func (fs *FileSystemSource) GetContent(id string) ([]byte, error) {
	// Try common extensions
	for _, ext := range []string{".htm", ".html"} {
		path := filepath.Join(fs.baseDir, id+ext)
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("content file not found for id: %s", id)
}

// GetMetadata reads and parses the crawler metadata JSON.
func (fs *FileSystemSource) GetMetadata(id string) (*domain.CrawlerMetadata, error) {
	path := filepath.Join(fs.baseDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("metadata file not found for id %s: %w", id, err)
	}

	var meta domain.CrawlerMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata for id %s: %w", id, err)
	}

	return &meta, nil
}
