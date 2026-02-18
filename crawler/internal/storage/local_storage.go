package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/user/legisgraph-crawler/internal/domain"
)

type Config struct {
	Type     string
	LocalDir string
	// Add other config fields for S3/GCS here
}

func NewStorage(cfg Config) (domain.Storage, error) {
	switch cfg.Type {
	case "local":
		return NewLocalStorage(cfg.LocalDir), nil
	case "s3":
		return nil, fmt.Errorf("s3 storage not implemented yet")
	case "gcs":
		return nil, fmt.Errorf("gcs storage not implemented yet")
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
}

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{
		baseDir: baseDir,
	}
}

func (s *LocalStorage) Save(filename string, content io.Reader) (string, error) {
	fullPath := filepath.Join(s.baseDir, filename)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (s *LocalStorage) Read(filename string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, filename)
	return os.Open(fullPath)
}

func (s *LocalStorage) Exists(filename string) bool {
	fullPath := filepath.Join(s.baseDir, filename)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}

func (s *LocalStorage) ReadAt(filename string) (io.ReaderAt, int64, error) {
	fullPath := filepath.Join(s.baseDir, filename)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}
