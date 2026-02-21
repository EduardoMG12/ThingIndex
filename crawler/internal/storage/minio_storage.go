package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioStorage implements the domain.Storage interface using AWS S3 / MinIO.
type MinioStorage struct {
	client     *minio.Client
	bucketName string
}

// NewMinioStorage initializes and returns a new MinioStorage.
func NewMinioStorage(endpoint, accessKeyID, secretAccessKey string, useSSL bool, bucketName string) (*MinioStorage, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	// Create bucket if it doesn't exist
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created MinIO bucket: %s\n", bucketName)
	}

	return &MinioStorage{
		client:     minioClient,
		bucketName: bucketName,
	}, nil
}

// Save streams content to the MinIO bucket.
// Returns the uploaded object key (filename).
func (s *MinioStorage) Save(filename string, content io.Reader) (string, error) {
	ctx := context.Background()
	
	// Normalize path for S3 (replace backslashes from Windows if any)
	objectName := strings.ReplaceAll(filename, "\\", "/")

	opts := minio.PutObjectOptions{}
	_, err := s.client.PutObject(ctx, s.bucketName, objectName, content, -1, opts)
	if err != nil {
		return "", fmt.Errorf("failed to put object '%s': %w", objectName, err)
	}

	return objectName, nil
}

// Exists checks if an object exists in the MinIO bucket.
func (s *MinioStorage) Exists(filename string) bool {
	ctx := context.Background()
	
	_, err := s.client.StatObject(ctx, s.bucketName, filename, minio.StatObjectOptions{})
	return err == nil
}

// Read reads an object from the MinIO bucket. Returns an io.ReadCloser.
func (s *MinioStorage) Read(filename string) (io.ReadCloser, error) {
	ctx := context.Background()
	
	object, err := s.client.GetObject(ctx, s.bucketName, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object '%s': %w", filename, err)
	}
	
	return object, nil
}

// ReadAt reads a chunk from an object in the MinIO bucket, required by standard Parsers.
func (s *MinioStorage) ReadAt(filename string) (io.ReaderAt, int64, error) {
	ctx := context.Background()
	
	stat, err := s.client.StatObject(ctx, s.bucketName, filename, minio.StatObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat object for ReadAt '%s': %w", filename, err)
	}
	
	// Create an object handle which implements io.ReaderAt
	object, err := s.client.GetObject(ctx, s.bucketName, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object for ReadAt '%s': %w", filename, err)
	}

	return object, stat.Size, nil
}
