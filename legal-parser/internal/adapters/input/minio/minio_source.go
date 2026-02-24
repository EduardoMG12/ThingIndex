package minio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/user/legisgraph/legal-parser/internal/domain"
)

// MinioSource implements domain.SourceProvider by reading crawler output from MinIO.
type MinioSource struct {
	client     *minio.Client
	bucketName string

	// Cache: baseName -> full object key (populated during ListDocuments)
	mu           sync.RWMutex
	contentKeys  map[string]string // baseName -> .htm/.html key
	metadataKeys map[string]string // baseName -> .json key
}

// NewMinioSource connects to MinIO and returns a new MinioSource.
func NewMinioSource(endpoint, accessKeyID, secretAccessKey string, useSSL bool, bucketName string) (*MinioSource, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	log.Printf("MinIO source connected to bucket: %s", bucketName)

	return &MinioSource{
		client:       minioClient,
		bucketName:   bucketName,
		contentKeys:  make(map[string]string),
		metadataKeys: make(map[string]string),
	}, nil
}

// ListDocuments lists all paired .htm + .json files in the entire bucket.
func (ms *MinioSource) ListDocuments() ([]domain.RawDocument, error) {
	return ms.listDocumentsFromPrefix("")
}

// ListDocumentsInDir lists all paired .htm + .json files under a specific directory prefix.
func (ms *MinioSource) ListDocumentsInDir(dirName string) ([]domain.RawDocument, error) {
	prefix := dirName
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return ms.listDocumentsFromPrefix(prefix)
}

func (ms *MinioSource) listDocumentsFromPrefix(prefix string) ([]domain.RawDocument, error) {
	ctx := context.Background()

	htmFiles := map[string]string{}  // baseName -> objectKey
	jsonFiles := map[string]string{}

	objectsCh := ms.client.ListObjects(ctx, ms.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objectsCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", obj.Err)
		}

		key := obj.Key

		// Extract filename from the object key
		fileName := key
		if idx := strings.LastIndex(fileName, "/"); idx != -1 {
			fileName = fileName[idx+1:]
		}

		ext := ""
		baseName := fileName
		if dotIdx := strings.LastIndex(baseName, "."); dotIdx != -1 {
			ext = strings.ToLower(baseName[dotIdx:])
			baseName = baseName[:dotIdx]
		}

		switch {
		case ext == ".htm" || ext == ".html":
			htmFiles[baseName] = key
		case ext == ".json":
			jsonFiles[baseName] = key
		}
	}

	// Build document pairs and cache the keys for fast lookups
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var docs []domain.RawDocument
	for baseName, htmKey := range htmFiles {
		jsonKey, hasJSON := jsonFiles[baseName]
		if !hasJSON {
			continue
		}

		// Cache the keys
		ms.contentKeys[baseName] = htmKey
		ms.metadataKeys[baseName] = jsonKey

		docs = append(docs, domain.RawDocument{
			ID:           baseName,
			ContentFile:  htmKey,
			MetadataFile: jsonKey,
		})
	}

	log.Printf("Found %d document pairs in MinIO (bucket=%s, prefix=%s)", len(docs), ms.bucketName, prefix)
	return docs, nil
}

// GetContent reads the raw HTML bytes from MinIO using cached key.
func (ms *MinioSource) GetContent(id string) ([]byte, error) {
	ms.mu.RLock()
	objectKey, ok := ms.contentKeys[id]
	ms.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("content file not found in cache for id: %s", id)
	}

	ctx := context.Background()
	object, err := ms.client.GetObject(ctx, ms.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object '%s': %w", objectKey, err)
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object '%s': %w", objectKey, err)
	}

	return data, nil
}

// GetMetadata reads and parses the crawler metadata JSON from MinIO using cached key.
func (ms *MinioSource) GetMetadata(id string) (*domain.CrawlerMetadata, error) {
	ms.mu.RLock()
	objectKey, ok := ms.metadataKeys[id]
	ms.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("metadata file not found in cache for id: %s", id)
	}

	ctx := context.Background()
	object, err := ms.client.GetObject(ctx, ms.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata object '%s': %w", objectKey, err)
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata object '%s': %w", objectKey, err)
	}

	var meta domain.CrawlerMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata for id %s: %w", id, err)
	}

	return &meta, nil
}

// GetContentByKey reads raw content directly by its MinIO object key.
func (ms *MinioSource) GetContentByKey(key string) ([]byte, error) {
	ctx := context.Background()
	object, err := ms.client.GetObject(ctx, ms.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object '%s': %w", key, err)
	}
	defer object.Close()
	return io.ReadAll(object)
}

// GetMetadataByKey reads and parses crawler metadata JSON directly by key.
func (ms *MinioSource) GetMetadataByKey(key string) (*domain.CrawlerMetadata, error) {
	data, err := ms.GetContentByKey(key)
	if err != nil {
		return nil, err
	}
	var meta domain.CrawlerMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata from key %s: %w", key, err)
	}
	return &meta, nil
}

// SaveParsed saves parsed JSON output back to MinIO under a given key.
func (ms *MinioSource) SaveParsed(key string, data []byte) error {
	ctx := context.Background()
	reader := strings.NewReader(string(data))
	_, err := ms.client.PutObject(ctx, ms.bucketName, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to save parsed object '%s': %w", key, err)
	}
	return nil
}
