package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type GCSStorage struct {
	client     *gcs.Client
	bucket     *gcs.BucketHandle
	bucketName string
}

var gcsClientEnvMutex sync.Mutex

func NewGCSStorage(projectID, bucketName, location, credentialsJSON, emulatorHost string) (*GCSStorage, error) {
	cfg := GCSStorageConfig{
		ProjectID:       projectID,
		BucketName:      bucketName,
		Location:        location,
		CredentialsJSON: credentialsJSON,
		EmulatorHost:    emulatorHost,
	}

	return newGCSStorage(context.Background(), cfg)
}

type GCSStorageConfig struct {
	ProjectID       string
	BucketName      string
	Location        string
	CredentialsJSON string
	EmulatorHost    string
}

func newGCSStorage(ctx context.Context, cfg GCSStorageConfig) (*GCSStorage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	projectID := strings.TrimSpace(cfg.ProjectID)
	bucketName := strings.TrimSpace(cfg.BucketName)
	if projectID == "" || bucketName == "" {
		return nil, errors.New("gcs project_id and bucket_name are required")
	}

	client, err := newGCSClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCS client: %w", err)
	}

	bucket := client.Bucket(bucketName)
	_, err = bucket.Attrs(ctx)
	switch {
	case err == nil:
	case errors.Is(err, gcs.ErrBucketNotExist):
		attrs := &gcs.BucketAttrs{}
		if location := strings.TrimSpace(cfg.Location); location != "" {
			attrs.Location = location
		}
		if err := bucket.Create(ctx, projectID, attrs); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("failed to create GCS bucket %s: %w", bucketName, err)
		}
	default:
		_ = client.Close()
		return nil, fmt.Errorf("failed to inspect GCS bucket %s: %w", bucketName, err)
	}

	return &GCSStorage{
		client:     client,
		bucket:     bucket,
		bucketName: bucketName,
	}, nil
}

func newGCSClient(ctx context.Context, cfg GCSStorageConfig) (*gcs.Client, error) {
	if emulatorHost := strings.TrimSpace(cfg.EmulatorHost); emulatorHost != "" {
		return newGCSEmulatorClient(ctx, emulatorHost)
	}

	opts := make([]option.ClientOption, 0, 1)
	if credentialsJSON := strings.TrimSpace(cfg.CredentialsJSON); credentialsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credentialsJSON)))
	}

	return gcs.NewClient(ctx, opts...)
}

func newGCSEmulatorClient(ctx context.Context, emulatorHost string) (*gcs.Client, error) {
	normalizedHost := normalizeGCSEmulatorHost(emulatorHost)

	gcsClientEnvMutex.Lock()
	defer gcsClientEnvMutex.Unlock()

	previousValue, hadPreviousValue := os.LookupEnv("STORAGE_EMULATOR_HOST")
	if err := os.Setenv("STORAGE_EMULATOR_HOST", normalizedHost); err != nil {
		return nil, fmt.Errorf("set STORAGE_EMULATOR_HOST: %w", err)
	}
	defer func() {
		if hadPreviousValue {
			_ = os.Setenv("STORAGE_EMULATOR_HOST", previousValue)
			return
		}
		_ = os.Unsetenv("STORAGE_EMULATOR_HOST")
	}()

	return gcs.NewClient(ctx)
}

func normalizeGCSEmulatorHost(rawHost string) string {
	trimmed := strings.TrimSpace(rawHost)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}

	return "http://" + trimmed
}

func (g *GCSStorage) UploadFile(ctx context.Context, file io.Reader, _ int64, contentType string, objectName string) (string, error) {
	objectKey, err := objectKeyFromPath("", objectName)
	if err != nil {
		return "", err
	}

	writer := g.bucket.Object(objectKey).NewWriter(ctx)
	writer.ContentType = contentType
	if _, err := io.Copy(writer, file); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("failed to upload file to GCS: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize GCS upload: %w", err)
	}

	return objectPathForStorage(g.bucketName, objectKey), nil
}

func (g *GCSStorage) OpenFile(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	objectKey, err := objectKeyFromPath(g.bucketName, objectPath)
	if err != nil {
		return nil, err
	}

	reader, err := g.bucket.Object(objectKey).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open file from GCS: %w", err)
	}

	return reader, nil
}

func (g *GCSStorage) DeleteFile(ctx context.Context, objectPath string) error {
	objectKey, err := objectKeyFromPath(g.bucketName, objectPath)
	if err != nil {
		return err
	}

	if err := g.bucket.Object(objectKey).Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file from GCS: %w", err)
	}

	return nil
}

func (g *GCSStorage) FileExists(ctx context.Context, objectPath string) (bool, error) {
	objectKey, err := objectKeyFromPath(g.bucketName, objectPath)
	if err != nil {
		return false, err
	}

	_, err = g.bucket.Object(objectKey).Attrs(ctx)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gcs.ErrObjectNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("failed to stat file from GCS: %w", err)
}
