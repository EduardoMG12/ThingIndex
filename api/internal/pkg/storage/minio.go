package storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"thing-index/internal/pkg/netsec"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client     *minio.Client
	bucketName string
}

func NewMinioStorage(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinioStorage, error) {
	return newMinioStorage(endpoint, accessKey, secretKey, bucketName, useSSL, nil)
}

func NewValidatedMinioStorage(endpoint, accessKey, secretKey, bucketName string, useSSL bool, endpointValidator *netsec.StorageEndpointValidator) (*MinioStorage, error) {
	if endpointValidator == nil {
		return nil, errors.New("storage endpoint validator is required")
	}

	return newMinioStorage(endpoint, accessKey, secretKey, bucketName, useSSL, endpointValidator)
}

func newMinioStorage(endpoint, accessKey, secretKey, bucketName string, useSSL bool, endpointValidator *netsec.StorageEndpointValidator) (*MinioStorage, error) {
	transport, err := minioTransport(endpoint, useSSL, endpointValidator)
	if err != nil {
		return nil, err
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Transport:    transport,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx := context.Background()
	exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
	if errBucketExists != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", errBucketExists)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		log.Printf("Successfully created bucket %s\n", bucketName)
	}

	return &MinioStorage{
		client:     minioClient,
		bucketName: bucketName,
	}, nil
}

func minioTransport(endpoint string, useSSL bool, endpointValidator *netsec.StorageEndpointValidator) (*http.Transport, error) {
	transport, err := minio.DefaultTransport(useSSL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO transport: %w", err)
	}

	if endpointValidator == nil {
		return transport, nil
	}

	defaultPort := "80"
	if useSSL {
		defaultPort = "443"
	}

	resolved, err := endpointValidator.Resolve(context.Background(), endpoint, defaultPort)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage endpoint: %w", err)
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	dialContext, err := endpointValidator.NewDialContext(endpoint, defaultPort, dialer.DialContext)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare storage dialer: %w", err)
	}
	transport.DialContext = dialContext

	if useSSL && resolved.ServerName != "" {
		transport.TLSClientConfig = cloneTLSConfig(transport.TLSClientConfig)
		transport.TLSClientConfig.ServerName = resolved.ServerName
	}

	return transport, nil
}

func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return cfg.Clone()
}

func (m *MinioStorage) UploadFile(ctx context.Context, file io.Reader, objectSize int64, contentType string, objectName string) (string, error) {
	info, err := m.client.PutObject(ctx, m.bucketName, objectName, file, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to minio: %w", err)
	}

	fileUrl := fmt.Sprintf("%s/%s", m.bucketName, info.Key)
	return fileUrl, nil
}

func (m *MinioStorage) OpenFile(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	objectKey := strings.TrimPrefix(strings.TrimSpace(objectPath), m.bucketName+"/")
	if objectKey == "" {
		return nil, errors.New("object path is required")
	}

	object, err := m.client.GetObject(ctx, m.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to open file from minio: %w", err)
	}

	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("failed to stat file from minio: %w", err)
	}

	return object, nil
}
